// ABOUTME: Unified Storage layer that wraps all SQLite stores
// ABOUTME: Provides the same interface as the old Charm-based Storage
package sqlite

import (
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/harper/remember-standalone/internal/models"
)

// Storage manages all persistent data for HMLR using SQLite
type Storage struct {
	db            *DB
	blocks        *BlockStore
	turns         *TurnStore
	facts         *FactStore
	embeddings    *EmbeddingStore
	profile       *ProfileStore
	openaiClient  interface {
		GenerateEmbedding(text string) ([]float64, error)
	}
	chunkEngine interface {
		ChunkTurn(text string, turnID string) ([]models.Chunk, error)
	}
	mu sync.RWMutex
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

// NewStorage initializes storage with SQLite backend
func NewStorage() (*Storage, error) {
	return NewStorageWithPath(DefaultDBPath())
}

// NewStorageWithPath initializes storage with a custom database path
func NewStorageWithPath(dbPath string) (*Storage, error) {
	db, err := Open(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	return &Storage{
		db:         db,
		blocks:     NewBlockStore(db),
		turns:      NewTurnStore(db),
		facts:      NewFactStore(db),
		embeddings: NewEmbeddingStore(db),
		profile:    NewProfileStore(db),
	}, nil
}

// NewStorageInMemory creates an in-memory storage (for testing)
func NewStorageInMemory() (*Storage, error) {
	db, err := OpenInMemory()
	if err != nil {
		return nil, fmt.Errorf("failed to open in-memory database: %w", err)
	}

	return &Storage{
		db:         db,
		blocks:     NewBlockStore(db),
		turns:      NewTurnStore(db),
		facts:      NewFactStore(db),
		embeddings: NewEmbeddingStore(db),
		profile:    NewProfileStore(db),
	}, nil
}

// Close closes the database connection
func (s *Storage) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
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
func (s *Storage) StoreTurn(turn *models.Turn) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Get active blocks and auto-repair if invariant violated
	activeBlocks, err := s.blocks.GetByStatus(models.StatusActive)
	if err != nil {
		return "", fmt.Errorf("failed to check active blocks: %w", err)
	}

	// Auto-repair: if multiple active blocks exist, pause all but newest
	if len(activeBlocks) > 1 {
		var newestBlock *models.BridgeBlock
		for i := range activeBlocks {
			block := &activeBlocks[i]
			if newestBlock == nil || block.UpdatedAt.After(newestBlock.UpdatedAt) {
				newestBlock = block
			}
		}
		for i := range activeBlocks {
			if activeBlocks[i].BlockID != newestBlock.BlockID {
				if err := s.blocks.UpdateStatus(activeBlocks[i].BlockID, models.StatusPaused); err != nil {
					return "", fmt.Errorf("failed to auto-repair block %s: %w", activeBlocks[i].BlockID, err)
				}
			}
		}
		activeBlocks, _ = s.blocks.GetByStatus(models.StatusActive)
	}

	// Pause any existing active block
	if len(activeBlocks) == 1 {
		if err := s.blocks.UpdateStatus(activeBlocks[0].BlockID, models.StatusPaused); err != nil {
			return "", fmt.Errorf("failed to pause existing active block: %w", err)
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
		TurnCount:  1,
	}

	// Save Bridge Block
	if err := s.blocks.Save(block); err != nil {
		return "", fmt.Errorf("failed to save bridge block: %w", err)
	}

	// Save the turn
	if err := s.turns.Save(blockID, turn); err != nil {
		return "", fmt.Errorf("failed to save turn: %w", err)
	}

	// Generate and save embeddings if clients are configured
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

		if err := s.embeddings.Save(chunk.ChunkID, turn.TurnID, blockID, embedding); err != nil {
			return fmt.Errorf("failed to save embedding for chunk %s: %w", chunk.ChunkID, err)
		}
	}

	return nil
}

// GetBridgeBlock retrieves a Bridge Block
func (s *Storage) GetBridgeBlock(blockID string) (*models.BridgeBlock, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.blocks.GetWithTurns(blockID)
}

// GetActiveBridgeBlocks retrieves all Bridge Blocks with ACTIVE status
func (s *Storage) GetActiveBridgeBlocks() ([]models.BridgeBlock, error) {
	return s.blocks.GetByStatus(models.StatusActive)
}

// GetPausedBridgeBlocks retrieves all Bridge Blocks with PAUSED status
func (s *Storage) GetPausedBridgeBlocks() ([]models.BridgeBlock, error) {
	return s.blocks.GetByStatus(models.StatusPaused)
}

// UpdateBridgeBlockStatus updates the status of a Bridge Block
func (s *Storage) UpdateBridgeBlockStatus(blockID string, status models.BridgeBlockStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.blocks.UpdateStatus(blockID, status)
}

// AppendTurnToBlock appends a turn to an existing Bridge Block
func (s *Storage) AppendTurnToBlock(blockID string, turn *models.Turn) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	block, err := s.blocks.Get(blockID)
	if err != nil || block == nil {
		return fmt.Errorf("failed to get block: %w", err)
	}

	// Save the turn
	if err := s.turns.Save(blockID, turn); err != nil {
		return fmt.Errorf("failed to save turn: %w", err)
	}

	// Update block metadata
	block.TurnCount++
	block.UpdatedAt = time.Now()

	// Merge keywords
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

	return s.blocks.Save(block)
}

// DeleteBridgeBlock deletes a bridge block (cascade deletes turns and embeddings)
func (s *Storage) DeleteBridgeBlock(blockID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.blocks.Delete(blockID)
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
	if s.openaiClient != nil {
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
	var results []models.MemorySearchResult

	blocks, err := s.blocks.ListAll()
	if err != nil {
		return results
	}

	for _, block := range blocks {
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

	vectorResults, err := s.embeddings.SearchSimilar(queryEmbedding, maxResults*3)
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
		if err != nil || block == nil {
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

// --- Fact operations ---

// SaveFact saves a single fact
func (s *Storage) SaveFact(fact *models.Fact) error {
	return s.facts.Save(fact)
}

// SaveFacts saves a slice of facts
func (s *Storage) SaveFacts(facts []models.Fact) error {
	for _, fact := range facts {
		if err := s.facts.Save(&fact); err != nil {
			return err
		}
	}
	return nil
}

// GetFactByKey retrieves a fact by its key (returns most recent if multiple exist)
func (s *Storage) GetFactByKey(key string) (*models.Fact, error) {
	return s.facts.GetByKey(key)
}

// GetFactsForBlock retrieves all facts for a specific block
func (s *Storage) GetFactsForBlock(blockID string) ([]models.Fact, error) {
	return s.facts.GetByBlock(blockID)
}

// SearchFacts searches for facts relevant to a query string
func (s *Storage) SearchFacts(query string, maxResults int) ([]models.Fact, error) {
	return s.facts.Search(query, maxResults)
}

// DeleteFactByKey deletes all facts with the given key
func (s *Storage) DeleteFactByKey(key string) (int64, error) {
	return s.facts.DeleteByKey(key)
}

// DeleteFactByID deletes a specific fact by its ID
func (s *Storage) DeleteFactByID(factID string) error {
	return s.facts.DeleteByID(factID)
}

// --- Profile operations ---

// GetUserProfile loads the user profile
func (s *Storage) GetUserProfile() (*models.UserProfile, error) {
	return s.profile.Get()
}

// SaveUserProfile saves the user profile
func (s *Storage) SaveUserProfile(profile *models.UserProfile) error {
	profile.LastUpdated = time.Now()
	return s.profile.Save(profile)
}

// --- Embedding operations ---

// GetVectorStorage returns the underlying embedding store (for compatibility)
func (s *Storage) GetVectorStorage() *EmbeddingStore {
	return s.embeddings
}

// RepairActiveBlockInvariant fixes multiple ACTIVE blocks by keeping newest
func (s *Storage) RepairActiveBlockInvariant() (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	activeBlocks, err := s.blocks.GetByStatus(models.StatusActive)
	if err != nil {
		return false, fmt.Errorf("failed to get active blocks: %w", err)
	}

	if len(activeBlocks) <= 1 {
		return false, nil
	}

	var newestBlock *models.BridgeBlock
	for i := range activeBlocks {
		block := &activeBlocks[i]
		if newestBlock == nil || block.UpdatedAt.After(newestBlock.UpdatedAt) {
			newestBlock = block
		}
	}

	for _, block := range activeBlocks {
		if block.BlockID != newestBlock.BlockID {
			if err := s.blocks.UpdateStatus(block.BlockID, models.StatusPaused); err != nil {
				return false, fmt.Errorf("failed to pause block %s: %w", block.BlockID, err)
			}
		}
	}

	return true, nil
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
	return strings.Contains(strings.ToLower(block.TopicLabel), queryLower)
}
