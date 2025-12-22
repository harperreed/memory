// ABOUTME: Main storage implementation for HMLR memory system
// ABOUTME: Uses Charm KV for cloud-synced storage with SSH key auth
package storage

import (
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/harper/remember-standalone/internal/charm"
	"github.com/harper/remember-standalone/internal/models"
)

// Storage manages all persistent data for HMLR using Charm KV
type Storage struct {
	charm         *charm.Client
	vectorStorage *VectorStorage
	openaiClient  interface {
		GenerateEmbedding(text string) ([]float64, error)
	}
	chunkEngine interface {
		ChunkTurn(text string, turnID string) ([]models.Chunk, error)
	}
	mu sync.RWMutex // Protects concurrent access to StoreTurn and block operations
}

// BridgeBlockInfo contains summary information about a Bridge Block
type BridgeBlockInfo struct {
	BlockID   string
	Topic     string
	Status    string
	TurnCount int
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewStorage initializes storage with Charm KV backend
func NewStorage() (*Storage, error) {
	// Initialize charm client
	charmClient, err := charm.GetClient()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize charm client: %w", err)
	}

	// Initialize vector storage (still uses charm under the hood)
	vectorStorage, err := NewVectorStorage(charmClient)
	if err != nil {
		return nil, fmt.Errorf("failed to create vector storage: %w", err)
	}

	return &Storage{
		charm:         charmClient,
		vectorStorage: vectorStorage,
	}, nil
}

// Close closes the charm client
func (s *Storage) Close() error {
	if s.charm != nil {
		return s.charm.Close()
	}
	return nil
}

// GetVectorStorage returns the underlying VectorStorage instance
func (s *Storage) GetVectorStorage() *VectorStorage {
	return s.vectorStorage
}

// SetOpenAIClient sets the OpenAI client for embeddings
func (s *Storage) SetOpenAIClient(client interface {
	GenerateEmbedding(text string) ([]float64, error)
}) {
	s.openaiClient = client
}

// SetChunkEngine sets the chunk engine for text chunking
func (s *Storage) SetChunkEngine(engine interface {
	ChunkTurn(text string, turnID string) ([]models.Chunk, error)
}) {
	s.chunkEngine = engine
}

// StoreTurn stores a conversation turn and creates/updates a Bridge Block
// INVARIANT: Only ONE block can be ACTIVE at a time.
// If multiple active blocks are found (can happen with distributed sync), auto-repairs by keeping newest.
func (s *Storage) StoreTurn(turn *models.Turn) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Get active blocks and auto-repair if invariant violated
	activeBlocks, err := s.getBridgeBlocksByStatusUnlocked(models.StatusActive)
	if err != nil {
		return "", fmt.Errorf("failed to check active blocks: %w", err)
	}

	// Auto-repair: if multiple active blocks exist (distributed sync race), pause all but newest
	if len(activeBlocks) > 1 {
		var newestBlock *models.BridgeBlock
		for i := range activeBlocks {
			block := &activeBlocks[i]
			if newestBlock == nil || block.UpdatedAt.After(newestBlock.UpdatedAt) {
				newestBlock = block
			}
		}
		// Pause all except newest
		for i := range activeBlocks {
			if activeBlocks[i].BlockID != newestBlock.BlockID {
				activeBlocks[i].Status = models.StatusPaused
				activeBlocks[i].UpdatedAt = time.Now()
				if saveErr := s.saveBridgeBlock(&activeBlocks[i]); saveErr != nil {
					return "", fmt.Errorf("failed to auto-repair block %s: %w", activeBlocks[i].BlockID, saveErr)
				}
			}
		}
		// Re-fetch to get updated list
		activeBlocks, err = s.getBridgeBlocksByStatusUnlocked(models.StatusActive)
		if err != nil {
			return "", fmt.Errorf("failed to re-check active blocks: %w", err)
		}
	}

	// Pause any existing active block (should be 0 or 1 now)
	if len(activeBlocks) == 1 {
		existingBlock := activeBlocks[0]
		existingBlock.Status = models.StatusPaused
		existingBlock.UpdatedAt = time.Now()
		if saveErr := s.saveBridgeBlock(&existingBlock); saveErr != nil {
			return "", fmt.Errorf("failed to pause existing active block: %w", saveErr)
		}
	}

	today := time.Now().Format("2006-01-02")
	blockID := fmt.Sprintf("block_%s_%s", time.Now().Format("20060102_150405"), uuid.New().String()[:8])

	block := &models.BridgeBlock{
		BlockID:    blockID,
		DayID:      today,
		TopicLabel: inferTopicLabel(turn),
		Keywords:   turn.Keywords,
		Status:     models.StatusActive,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		Turns:      []models.Turn{*turn},
		TurnCount:  1,
	}

	// Save Bridge Block to Charm KV
	if err := s.saveBridgeBlock(block); err != nil {
		return "", fmt.Errorf("failed to save bridge block: %w", err)
	}

	// Extract and save facts (placeholder - handled by FactScrubber)
	if err := s.extractAndSaveFacts(turn, blockID); err != nil {
		return "", fmt.Errorf("failed to save facts: %w", err)
	}

	// Generate and save embeddings if OpenAI client and ChunkEngine are configured
	if s.openaiClient != nil && s.chunkEngine != nil {
		if err := s.generateAndSaveEmbeddings(turn, blockID); err != nil {
			log.Printf("[Storage] failed to generate embeddings: %v", err)
		}
	}

	return blockID, nil
}

// generateAndSaveEmbeddings generates and saves embeddings for a turn
func (s *Storage) generateAndSaveEmbeddings(turn *models.Turn, blockID string) error {
	fullText := turn.UserMessage + " " + turn.AIResponse

	chunks, err := s.chunkEngine.ChunkTurn(fullText, turn.TurnID)
	if err != nil {
		return fmt.Errorf("failed to chunk turn: %w", err)
	}

	for _, chunk := range chunks {
		embedding, err := s.openaiClient.GenerateEmbedding(chunk.Content)
		if err != nil {
			return fmt.Errorf("failed to generate embedding for chunk %s: %w", chunk.ChunkID, err)
		}

		if err := s.vectorStorage.SaveEmbedding(chunk.ChunkID, turn.TurnID, blockID, embedding); err != nil {
			return fmt.Errorf("failed to save embedding for chunk %s: %w", chunk.ChunkID, err)
		}
	}

	return nil
}

// saveBridgeBlock writes a Bridge Block to Charm KV
func (s *Storage) saveBridgeBlock(block *models.BridgeBlock) error {
	key := charm.BlockKey(block.BlockID)
	return s.charm.SetJSON(key, block)
}

// GetBridgeBlock retrieves a Bridge Block from Charm KV
func (s *Storage) GetBridgeBlock(blockID string) (*models.BridgeBlock, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.getBridgeBlockUnlocked(blockID)
}

// getBridgeBlockUnlocked retrieves a Bridge Block without locking (internal use)
func (s *Storage) getBridgeBlockUnlocked(blockID string) (*models.BridgeBlock, error) {
	key := charm.BlockKey(blockID)
	var block models.BridgeBlock
	err := s.charm.GetJSON(key, &block)
	if err != nil {
		return nil, fmt.Errorf("block not found: %s", blockID)
	}
	return &block, nil
}

// SearchMemory searches for relevant blocks based on query
func (s *Storage) SearchMemory(query string, maxResults int) ([]models.MemorySearchResult, error) {
	var allResults []models.MemorySearchResult
	blockScores := make(map[string]float64)

	// 1. Keyword-based search
	keywordResults := s.keywordSearch(query, maxResults)
	for _, result := range keywordResults {
		blockScores[result.BlockID] = result.RelevanceScore
		allResults = append(allResults, result)
	}

	// 2. Semantic search (if OpenAI client is available)
	if s.openaiClient != nil && s.vectorStorage != nil {
		semanticResults, err := s.semanticSearch(query, maxResults)
		if err != nil {
			log.Printf("[Storage] semantic search failed: %v", err)
		} else {
			for _, result := range semanticResults {
				if existingScore, exists := blockScores[result.BlockID]; exists {
					blockScores[result.BlockID] = (existingScore + result.RelevanceScore) / 2
				} else {
					blockScores[result.BlockID] = result.RelevanceScore
					allResults = append(allResults, result)
				}
			}
		}
	}

	// Update scores and sort
	for i := range allResults {
		if score, exists := blockScores[allResults[i].BlockID]; exists {
			allResults[i].RelevanceScore = score
		}
	}

	sort.Slice(allResults, func(i, j int) bool {
		return allResults[i].RelevanceScore > allResults[j].RelevanceScore
	})

	// Deduplicate
	seenBlocks := make(map[string]bool)
	var uniqueResults []models.MemorySearchResult
	for _, result := range allResults {
		if !seenBlocks[result.BlockID] {
			seenBlocks[result.BlockID] = true
			uniqueResults = append(uniqueResults, result)
		}
	}

	if len(uniqueResults) > maxResults {
		uniqueResults = uniqueResults[:maxResults]
	}

	return uniqueResults, nil
}

// keywordSearch performs keyword-based search across all blocks
func (s *Storage) keywordSearch(query string, maxResults int) []models.MemorySearchResult {
	results := []models.MemorySearchResult{}

	// Get all block keys
	keys, err := s.charm.ListKeys(charm.BlockPrefix)
	if err != nil {
		return results
	}

	for _, key := range keys {
		var block models.BridgeBlock
		if err := s.charm.GetJSON(key, &block); err != nil {
			continue
		}

		if matchesQuery(&block, query) {
			results = append(results, models.MemorySearchResult{
				BlockID:        block.BlockID,
				TopicLabel:     block.TopicLabel,
				RelevanceScore: 0.5,
				Summary:        block.Summary,
				Turns:          block.Turns,
			})
		}

		if len(results) >= maxResults*2 {
			break
		}
	}

	return results
}

// semanticSearch performs vector-based semantic search
func (s *Storage) semanticSearch(query string, maxResults int) ([]models.MemorySearchResult, error) {
	queryEmbedding, err := s.openaiClient.GenerateEmbedding(query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	vectorResults, err := s.vectorStorage.SearchSimilar(queryEmbedding, maxResults*3)
	if err != nil {
		return nil, fmt.Errorf("failed to search similar: %w", err)
	}

	blockScores := make(map[string]float64)
	for _, vr := range vectorResults {
		if existingScore, exists := blockScores[vr.BlockID]; !exists || vr.SimilarityScore > existingScore {
			blockScores[vr.BlockID] = vr.SimilarityScore
		}
	}

	var results []models.MemorySearchResult
	for blockID, score := range blockScores {
		block, err := s.GetBridgeBlock(blockID)
		if err != nil {
			continue
		}

		results = append(results, models.MemorySearchResult{
			BlockID:        block.BlockID,
			TopicLabel:     block.TopicLabel,
			RelevanceScore: score,
			Summary:        block.Summary,
			Turns:          block.Turns,
		})
	}

	return results, nil
}

// GetFactsForBlock retrieves all facts for a specific block
func (s *Storage) GetFactsForBlock(blockID string) ([]models.Fact, error) {
	facts := []models.Fact{}

	keys, err := s.charm.ListKeys(charm.FactPrefix)
	if err != nil {
		return nil, fmt.Errorf("failed to list fact keys: %w", err)
	}

	for _, key := range keys {
		// Skip bykey lookup entries
		if strings.Contains(key, ":bykey:") {
			continue
		}

		var fact models.Fact
		if err := s.charm.GetJSON(key, &fact); err != nil {
			continue
		}

		if fact.BlockID == blockID {
			facts = append(facts, fact)
		}
	}

	// Sort by created_at desc
	sort.Slice(facts, func(i, j int) bool {
		return facts[i].CreatedAt.After(facts[j].CreatedAt)
	})

	return facts, nil
}

// SearchFacts searches for facts relevant to a query string
func (s *Storage) SearchFacts(query string, maxResults int) ([]models.Fact, error) {
	facts := []models.Fact{}
	queryLower := strings.ToLower(query)

	keys, err := s.charm.ListKeys(charm.FactPrefix)
	if err != nil {
		return nil, fmt.Errorf("failed to list fact keys: %w", err)
	}

	for _, key := range keys {
		if strings.Contains(key, ":bykey:") {
			continue
		}

		var fact models.Fact
		if err := s.charm.GetJSON(key, &fact); err != nil {
			continue
		}

		// Check if query matches key or value
		if strings.Contains(strings.ToLower(fact.Key), queryLower) ||
			strings.Contains(strings.ToLower(fact.Value), queryLower) {
			facts = append(facts, fact)
		}

		if len(facts) >= maxResults {
			break
		}
	}

	// Sort by confidence desc, then created_at desc
	sort.Slice(facts, func(i, j int) bool {
		if facts[i].Confidence != facts[j].Confidence {
			return facts[i].Confidence > facts[j].Confidence
		}
		return facts[i].CreatedAt.After(facts[j].CreatedAt)
	})

	return facts, nil
}

// extractAndSaveFacts is a placeholder handled by FactScrubber
func (s *Storage) extractAndSaveFacts(turn *models.Turn, blockID string) error {
	return nil
}

// SaveFacts saves a slice of facts to Charm KV
func (s *Storage) SaveFacts(facts []models.Fact) error {
	for _, fact := range facts {
		if err := s.SaveFact(&fact); err != nil {
			return err
		}
	}
	return nil
}

// Helper functions

func inferTopicLabel(turn *models.Turn) string {
	if len(turn.Topics) > 0 {
		return turn.Topics[0]
	}
	return "General Discussion"
}

func matchesQuery(block *models.BridgeBlock, query string) bool {
	queryLower := strings.ToLower(query)
	for _, keyword := range block.Keywords {
		if strings.Contains(queryLower, strings.ToLower(keyword)) {
			return true
		}
	}
	// Also check topic label
	if strings.Contains(strings.ToLower(block.TopicLabel), queryLower) {
		return true
	}
	return false
}

// GetActiveBridgeBlocks retrieves all Bridge Blocks with ACTIVE status
func (s *Storage) GetActiveBridgeBlocks() ([]models.BridgeBlock, error) {
	return s.getBridgeBlocksByStatus(models.StatusActive)
}

// GetPausedBridgeBlocks retrieves all Bridge Blocks with PAUSED status
func (s *Storage) GetPausedBridgeBlocks() ([]models.BridgeBlock, error) {
	return s.getBridgeBlocksByStatus(models.StatusPaused)
}

// getBridgeBlocksByStatus retrieves all Bridge Blocks with a specific status
func (s *Storage) getBridgeBlocksByStatus(status models.BridgeBlockStatus) ([]models.BridgeBlock, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.getBridgeBlocksByStatusUnlocked(status)
}

// getBridgeBlocksByStatusUnlocked retrieves all Bridge Blocks with a specific status without locking (internal use)
func (s *Storage) getBridgeBlocksByStatusUnlocked(status models.BridgeBlockStatus) ([]models.BridgeBlock, error) {
	blocks := []models.BridgeBlock{}

	keys, err := s.charm.ListKeys(charm.BlockPrefix)
	if err != nil {
		return nil, fmt.Errorf("failed to list block keys: %w", err)
	}

	for _, key := range keys {
		var block models.BridgeBlock
		if err := s.charm.GetJSON(key, &block); err != nil {
			continue
		}

		if block.Status == status {
			blocks = append(blocks, block)
		}
	}

	return blocks, nil
}

// UpdateBridgeBlockStatus updates the status of a Bridge Block
func (s *Storage) UpdateBridgeBlockStatus(blockID string, status models.BridgeBlockStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	block, err := s.getBridgeBlockUnlocked(blockID)
	if err != nil {
		return fmt.Errorf("failed to get block: %w", err)
	}

	block.Status = status
	block.UpdatedAt = time.Now()

	return s.saveBridgeBlock(block)
}

// AppendTurnToBlock appends a turn to an existing Bridge Block
func (s *Storage) AppendTurnToBlock(blockID string, turn *models.Turn) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	block, err := s.getBridgeBlockUnlocked(blockID)
	if err != nil {
		return fmt.Errorf("failed to get block: %w", err)
	}

	block.Turns = append(block.Turns, *turn)
	block.TurnCount = len(block.Turns)
	block.UpdatedAt = time.Now()

	// Update keywords from turn
	for _, keyword := range turn.Keywords {
		found := false
		for _, existing := range block.Keywords {
			if existing == keyword {
				found = true
				break
			}
		}
		if !found {
			block.Keywords = append(block.Keywords, keyword)
		}
	}

	return s.saveBridgeBlock(block)
}

// GetUserProfile loads the user profile from Charm KV
func (s *Storage) GetUserProfile() (*models.UserProfile, error) {
	key := charm.ProfileKey()
	var profile models.UserProfile
	err := s.charm.GetJSON(key, &profile)
	if err != nil {
		// Not found, return nil without error
		return nil, nil
	}
	return &profile, nil
}

// SaveUserProfile saves the user profile to Charm KV
func (s *Storage) SaveUserProfile(profile *models.UserProfile) error {
	profile.LastUpdated = time.Now()
	key := charm.ProfileKey()
	return s.charm.SetJSON(key, profile)
}

// GetFactByKey retrieves a fact by its key (returns most recent if multiple exist)
func (s *Storage) GetFactByKey(key string) (*models.Fact, error) {
	// Try the bykey lookup first
	lookupKey := charm.FactByKeyKey(key)
	var factID string
	if err := s.charm.GetJSON(lookupKey, &factID); err == nil {
		var fact models.Fact
		if err := s.charm.GetJSON(charm.FactKey(factID), &fact); err == nil {
			return &fact, nil
		}
	}

	// Fallback: scan all facts
	keys, err := s.charm.ListKeys(charm.FactPrefix)
	if err != nil {
		return nil, fmt.Errorf("failed to list fact keys: %w", err)
	}

	var latestFact *models.Fact
	for _, k := range keys {
		if strings.Contains(k, ":bykey:") {
			continue
		}

		var fact models.Fact
		if err := s.charm.GetJSON(k, &fact); err != nil {
			continue
		}

		if fact.Key == key {
			if latestFact == nil || fact.CreatedAt.After(latestFact.CreatedAt) {
				latestFact = &fact
			}
		}
	}

	return latestFact, nil
}

// DeleteFactByKey deletes all facts with the given key
func (s *Storage) DeleteFactByKey(key string) (int64, error) {
	keys, err := s.charm.ListKeys(charm.FactPrefix)
	if err != nil {
		return 0, fmt.Errorf("failed to list fact keys: %w", err)
	}

	var count int64
	for _, k := range keys {
		if strings.Contains(k, ":bykey:") {
			continue
		}

		var fact models.Fact
		if err := s.charm.GetJSON(k, &fact); err != nil {
			continue
		}

		if fact.Key == key {
			if err := s.charm.Delete(k); err == nil {
				count++
			}
		}
	}

	// Delete the bykey lookup
	if err := s.charm.Delete(charm.FactByKeyKey(key)); err != nil {
		log.Printf("[Storage] failed to delete fact bykey lookup for %s: %v", key, err)
	}

	return count, nil
}

// DeleteFactByID deletes a specific fact by its ID
func (s *Storage) DeleteFactByID(factID string) error {
	// Get the fact first to find its key for lookup cleanup
	key := charm.FactKey(factID)
	var fact models.Fact
	if err := s.charm.GetJSON(key, &fact); err == nil {
		// Clean up bykey lookup
		if err := s.charm.Delete(charm.FactByKeyKey(fact.Key)); err != nil {
			log.Printf("[Storage] failed to delete fact bykey lookup for %s: %v", fact.Key, err)
		}
	}

	return s.charm.Delete(key)
}

// SaveFact saves a single fact to Charm KV
func (s *Storage) SaveFact(fact *models.Fact) error {
	// Save the fact itself
	key := charm.FactKey(fact.FactID)
	if err := s.charm.SetJSON(key, fact); err != nil {
		return fmt.Errorf("failed to save fact: %w", err)
	}

	// Create bykey lookup for fast retrieval
	lookupKey := charm.FactByKeyKey(fact.Key)
	if err := s.charm.SetJSON(lookupKey, fact.FactID); err != nil {
		// Non-fatal, just skip the lookup index
		log.Printf("[Storage] failed to create fact lookup index: %v", err)
	}

	return nil
}

// DeleteBridgeBlock deletes a bridge block and all its associated data
func (s *Storage) DeleteBridgeBlock(blockID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Delete associated facts first (cascade delete)
	facts, err := s.GetFactsForBlock(blockID)
	if err == nil {
		for _, fact := range facts {
			if err := s.DeleteFactByID(fact.FactID); err != nil {
				log.Printf("[Storage] failed to delete fact %s during block deletion: %v", fact.FactID, err)
			}
		}
	}

	// Delete associated embeddings
	keys, err := s.charm.ListKeys(charm.EmbeddingPrefix)
	if err == nil {
		for _, key := range keys {
			var emb models.Embedding
			if s.charm.GetJSON(key, &emb) == nil && emb.BlockID == blockID {
				if err := s.charm.Delete(key); err != nil {
					log.Printf("[Storage] failed to delete embedding %s during block deletion: %v", key, err)
				}
			}
		}
	}

	// Delete the block itself
	return s.charm.Delete(charm.BlockKey(blockID))
}

// Sync manually triggers a sync with the Charm server
func (s *Storage) Sync() error {
	return s.charm.Sync()
}

// GetCharmID returns the user's Charm ID
func (s *Storage) GetCharmID() (string, error) {
	return s.charm.ID()
}

// RepairActiveBlockInvariant fixes the case where multiple blocks are ACTIVE
// This can happen with distributed sync when multiple devices create blocks concurrently
// Returns true if repair was needed, false if invariant was already satisfied
func (s *Storage) RepairActiveBlockInvariant() (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	activeBlocks, err := s.getBridgeBlocksByStatusUnlocked(models.StatusActive)
	if err != nil {
		return false, fmt.Errorf("failed to get active blocks: %w", err)
	}

	if len(activeBlocks) <= 1 {
		return false, nil // Invariant satisfied
	}

	// Find the most recently updated block - keep it active, pause others
	var newestBlock *models.BridgeBlock
	for i := range activeBlocks {
		block := &activeBlocks[i]
		if newestBlock == nil || block.UpdatedAt.After(newestBlock.UpdatedAt) {
			newestBlock = block
		}
	}

	// Pause all blocks except the newest
	for _, block := range activeBlocks {
		if block.BlockID != newestBlock.BlockID {
			block.Status = models.StatusPaused
			block.UpdatedAt = time.Now()
			if err := s.saveBridgeBlock(&block); err != nil {
				return false, fmt.Errorf("failed to pause block %s: %w", block.BlockID, err)
			}
		}
	}

	return true, nil
}
