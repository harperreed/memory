// ABOUTME: Main storage implementation for HMLR memory system
// ABOUTME: Handles XDG directories, JSON files, and SQLite database
package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/adrg/xdg"
	"github.com/google/uuid"
	"github.com/harper/remember-standalone/internal/models"
	_ "github.com/mattn/go-sqlite3"
)

// Storage manages all persistent data for HMLR
type Storage struct {
	basePath      string
	db            *sql.DB
	vectorStorage *VectorStorage
	openaiClient  interface {
		GenerateEmbedding(text string) ([]float64, error)
	}
	chunkEngine interface {
		ChunkTurn(text string, turnID string) ([]models.Chunk, error)
	}
	mu sync.Mutex // Protects concurrent access to StoreTurn and block operations
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

// NewStorage initializes storage with XDG-compliant paths
func NewStorage() (*Storage, error) {
	// Use XDG data directory: ~/.local/share/memory/
	// Respects XDG_DATA_HOME environment variable override for testing
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		dataHome = xdg.DataHome
	}
	basePath := filepath.Join(dataHome, "memory")

	// Create directory structure
	dirs := []string{
		filepath.Join(basePath, "bridge_blocks"),
		filepath.Join(basePath, "embeddings"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Initialize SQLite database
	dbPath := filepath.Join(basePath, "facts.db")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Create facts table
	schema := `
	CREATE TABLE IF NOT EXISTS facts (
		fact_id TEXT PRIMARY KEY,
		block_id TEXT NOT NULL,
		turn_id TEXT NOT NULL,
		key TEXT NOT NULL,
		value TEXT NOT NULL,
		confidence REAL DEFAULT 1.0,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_facts_block ON facts(block_id);
	CREATE INDEX IF NOT EXISTS idx_facts_key ON facts(key);
	`

	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create schema: %w", err)
	}

	// Initialize vector storage
	vectorStorage, err := NewVectorStorage(dataHome)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create vector storage: %w", err)
	}

	return &Storage{
		basePath:      basePath,
		db:            db,
		vectorStorage: vectorStorage,
	}, nil
}

// Close closes the database connection
func (s *Storage) Close() error {
	if s.db != nil {
		return s.db.Close()
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
// INVARIANT: Only ONE block can be ACTIVE at a time. This is enforced by validation and maintained by this function.
func (s *Storage) StoreTurn(turn *models.Turn) (string, error) {
	// Lock to prevent concurrent modifications to active blocks (race condition fix)
	s.mu.Lock()
	defer s.mu.Unlock()

	// For now, create a new Bridge Block for each turn (simplified routing)
	// TODO: Implement full Governor routing logic (4 scenarios)

	// Validate active block cardinality - only one block should be ACTIVE at a time
	activeBlocks, err := s.GetActiveBridgeBlocks()
	if err != nil {
		return "", fmt.Errorf("failed to check active blocks: %w", err)
	}
	if len(activeBlocks) > 1 {
		return "", fmt.Errorf("invariant violation: found %d active blocks, expected at most 1", len(activeBlocks))
	}

	// Before creating a new active block, pause any existing active block
	// This maintains the invariant that only ONE block can be ACTIVE at a time
	if len(activeBlocks) == 1 {
		if err := s.UpdateBridgeBlockStatus(activeBlocks[0].BlockID, models.StatusPaused); err != nil {
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
		Turns:      []models.Turn{*turn},
		TurnCount:  1,
	}

	// Save Bridge Block to disk
	if err := s.saveBridgeBlock(block); err != nil {
		return "", fmt.Errorf("failed to save bridge block: %w", err)
	}

	// Extract and save facts
	if err := s.extractAndSaveFacts(turn, blockID); err != nil {
		return "", fmt.Errorf("failed to save facts: %w", err)
	}

	// Generate and save embeddings if OpenAI client and ChunkEngine are configured
	if s.openaiClient != nil && s.chunkEngine != nil {
		if err := s.generateAndSaveEmbeddings(turn, blockID); err != nil {
			// Log error but don't fail the entire operation
			// Embeddings are optional enhancement, not critical
			fmt.Fprintf(os.Stderr, "Warning: failed to generate embeddings: %v\n", err)
		}
	}

	return blockID, nil
}

// generateAndSaveEmbeddings generates and saves embeddings for a turn
func (s *Storage) generateAndSaveEmbeddings(turn *models.Turn, blockID string) error {
	// Combine user message and AI response for embedding
	fullText := turn.UserMessage + " " + turn.AIResponse

	// Chunk the text
	chunks, err := s.chunkEngine.ChunkTurn(fullText, turn.TurnID)
	if err != nil {
		return fmt.Errorf("failed to chunk turn: %w", err)
	}

	// Generate and save embeddings for each chunk
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

// saveBridgeBlock writes a Bridge Block to disk as JSON
func (s *Storage) saveBridgeBlock(block *models.BridgeBlock) error {
	dayDir := filepath.Join(s.basePath, "bridge_blocks", block.DayID)
	if err := os.MkdirAll(dayDir, 0755); err != nil {
		return fmt.Errorf("failed to create day directory: %w", err)
	}

	blockPath := filepath.Join(dayDir, block.BlockID+".json")
	data, err := json.MarshalIndent(block, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal block: %w", err)
	}

	if err := os.WriteFile(blockPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write block file: %w", err)
	}

	return nil
}

// GetBridgeBlock retrieves a Bridge Block from disk
func (s *Storage) GetBridgeBlock(blockID string) (*models.BridgeBlock, error) {
	// Search for block in recent days (last 30 days)
	for i := 0; i < 30; i++ {
		day := time.Now().AddDate(0, 0, -i).Format("2006-01-02")
		blockPath := filepath.Join(s.basePath, "bridge_blocks", day, blockID+".json")

		data, err := os.ReadFile(blockPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("failed to read block file: %w", err)
		}

		var block models.BridgeBlock
		if err := json.Unmarshal(data, &block); err != nil {
			return nil, fmt.Errorf("failed to unmarshal block: %w", err)
		}

		return &block, nil
	}

	return nil, fmt.Errorf("block not found: %s", blockID)
}

// SearchMemory searches for relevant blocks based on query
// Uses both keyword matching and semantic search (if OpenAI client is configured)
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
			// Log error but continue with keyword results
			fmt.Fprintf(os.Stderr, "Warning: semantic search failed: %v\n", err)
		} else {
			// Merge semantic results with keyword results
			for _, result := range semanticResults {
				if existingScore, exists := blockScores[result.BlockID]; exists {
					// Combine scores (weighted average)
					blockScores[result.BlockID] = (existingScore + result.RelevanceScore) / 2
				} else {
					blockScores[result.BlockID] = result.RelevanceScore
					allResults = append(allResults, result)
				}
			}
		}
	}

	// Update scores and sort by relevance
	for i := range allResults {
		if score, exists := blockScores[allResults[i].BlockID]; exists {
			allResults[i].RelevanceScore = score
		}
	}

	// Sort by relevance score (descending)
	sort.Slice(allResults, func(i, j int) bool {
		return allResults[i].RelevanceScore > allResults[j].RelevanceScore
	})

	// Deduplicate by BlockID
	seenBlocks := make(map[string]bool)
	var uniqueResults []models.MemorySearchResult
	for _, result := range allResults {
		if !seenBlocks[result.BlockID] {
			seenBlocks[result.BlockID] = true
			uniqueResults = append(uniqueResults, result)
		}
	}

	// Limit to maxResults
	if len(uniqueResults) > maxResults {
		uniqueResults = uniqueResults[:maxResults]
	}

	return uniqueResults, nil
}

// keywordSearch performs keyword-based search
func (s *Storage) keywordSearch(query string, maxResults int) []models.MemorySearchResult {
	results := []models.MemorySearchResult{}

	// Search last 30 days
	for i := 0; i < 30; i++ {
		day := time.Now().AddDate(0, 0, -i).Format("2006-01-02")
		dayDir := filepath.Join(s.basePath, "bridge_blocks", day)

		entries, err := os.ReadDir(dayDir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			continue
		}

		for _, entry := range entries {
			if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
				blockPath := filepath.Join(dayDir, entry.Name())
				data, err := os.ReadFile(blockPath)
				if err != nil {
					continue
				}

				var block models.BridgeBlock
				if err := json.Unmarshal(data, &block); err != nil {
					continue
				}

				// Simple keyword matching
				if matchesQuery(&block, query) {
					results = append(results, models.MemorySearchResult{
						BlockID:        block.BlockID,
						TopicLabel:     block.TopicLabel,
						RelevanceScore: 0.5, // Keyword match score
						Summary:        block.Summary,
						Turns:          block.Turns,
					})
				}

				if len(results) >= maxResults*2 {
					return results
				}
			}
		}
	}

	return results
}

// semanticSearch performs vector-based semantic search
func (s *Storage) semanticSearch(query string, maxResults int) ([]models.MemorySearchResult, error) {
	// Generate embedding for query
	queryEmbedding, err := s.openaiClient.GenerateEmbedding(query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	// Search for similar chunks
	vectorResults, err := s.vectorStorage.SearchSimilar(queryEmbedding, maxResults*3)
	if err != nil {
		return nil, fmt.Errorf("failed to search similar: %w", err)
	}

	// Map chunk results to block results
	blockScores := make(map[string]float64)
	for _, vr := range vectorResults {
		// Take the maximum similarity score for each block
		if existingScore, exists := blockScores[vr.BlockID]; !exists || vr.SimilarityScore > existingScore {
			blockScores[vr.BlockID] = vr.SimilarityScore
		}
	}

	// Create MemorySearchResult for each block
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
	rows, err := s.db.Query(`
		SELECT fact_id, block_id, turn_id, key, value, confidence, created_at
		FROM facts
		WHERE block_id = ?
		ORDER BY created_at DESC
	`, blockID)
	if err != nil {
		return nil, fmt.Errorf("failed to query facts: %w", err)
	}
	defer rows.Close()

	facts := []models.Fact{}
	for rows.Next() {
		var fact models.Fact
		if err := rows.Scan(
			&fact.FactID,
			&fact.BlockID,
			&fact.TurnID,
			&fact.Key,
			&fact.Value,
			&fact.Confidence,
			&fact.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan fact: %w", err)
		}
		facts = append(facts, fact)
	}

	return facts, nil
}

// SearchFacts searches for facts relevant to a query string
// Searches both keys and values for keyword matches
func (s *Storage) SearchFacts(query string, maxResults int) ([]models.Fact, error) {
	// Simple keyword search in both key and value columns
	searchPattern := "%" + query + "%"

	rows, err := s.db.Query(`
		SELECT fact_id, block_id, turn_id, key, value, confidence, created_at
		FROM facts
		WHERE key LIKE ? OR value LIKE ?
		ORDER BY confidence DESC, created_at DESC
		LIMIT ?
	`, searchPattern, searchPattern, maxResults)
	if err != nil {
		return nil, fmt.Errorf("failed to query facts: %w", err)
	}
	defer rows.Close()

	facts := []models.Fact{}
	for rows.Next() {
		var fact models.Fact
		if err := rows.Scan(
			&fact.FactID,
			&fact.BlockID,
			&fact.TurnID,
			&fact.Key,
			&fact.Value,
			&fact.Confidence,
			&fact.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan fact: %w", err)
		}
		facts = append(facts, fact)
	}

	return facts, nil
}

// extractAndSaveFacts is a placeholder that will be replaced by FactScrubber
// This method is called by StoreTurn but currently does nothing
// The actual fact extraction should be done by calling FactScrubber.ExtractAndSave explicitly
func (s *Storage) extractAndSaveFacts(turn *models.Turn, blockID string) error {
	// Fact extraction is now handled by FactScrubber
	// This method is kept for backward compatibility but does nothing
	return nil
}

// SaveFacts saves a slice of facts to the database
// Called by FactScrubber after extracting facts
func (s *Storage) SaveFacts(facts []models.Fact) error {
	for _, fact := range facts {
		_, err := s.db.Exec(`
			INSERT INTO facts (fact_id, block_id, turn_id, key, value, confidence, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, fact.FactID, fact.BlockID, fact.TurnID, fact.Key, fact.Value, fact.Confidence, fact.CreatedAt)
		if err != nil {
			return fmt.Errorf("failed to insert fact: %w", err)
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
	// Simple keyword matching
	for _, keyword := range block.Keywords {
		if containsIgnoreCase(query, keyword) {
			return true
		}
	}
	return false
}

func containsIgnoreCase(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
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
	blocks := []models.BridgeBlock{}

	// Search last 30 days
	for i := 0; i < 30; i++ {
		day := time.Now().AddDate(0, 0, -i).Format("2006-01-02")
		dayDir := filepath.Join(s.basePath, "bridge_blocks", day)

		entries, err := os.ReadDir(dayDir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("failed to read day directory: %w", err)
		}

		for _, entry := range entries {
			if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
				blockPath := filepath.Join(dayDir, entry.Name())
				data, err := os.ReadFile(blockPath)
				if err != nil {
					continue
				}

				var block models.BridgeBlock
				if err := json.Unmarshal(data, &block); err != nil {
					continue
				}

				if block.Status == status {
					blocks = append(blocks, block)
				}
			}
		}
	}

	return blocks, nil
}

// UpdateBridgeBlockStatus updates the status of a Bridge Block
func (s *Storage) UpdateBridgeBlockStatus(blockID string, status models.BridgeBlockStatus) error {
	block, err := s.GetBridgeBlock(blockID)
	if err != nil {
		return fmt.Errorf("failed to get block: %w", err)
	}

	block.Status = status
	block.UpdatedAt = time.Now()

	return s.saveBridgeBlock(block)
}

// AppendTurnToBlock appends a turn to an existing Bridge Block
func (s *Storage) AppendTurnToBlock(blockID string, turn *models.Turn) error {
	block, err := s.GetBridgeBlock(blockID)
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

// GetUserProfile loads the user profile from disk
func (s *Storage) GetUserProfile() (*models.UserProfile, error) {
	return models.LoadUserProfile()
}

// SaveUserProfile saves the user profile to disk
func (s *Storage) SaveUserProfile(profile *models.UserProfile) error {
	return profile.Save()
}

// GetFactByKey retrieves a fact by its key (returns most recent if multiple exist)
func (s *Storage) GetFactByKey(key string) (*models.Fact, error) {
	row := s.db.QueryRow(`
		SELECT fact_id, block_id, turn_id, key, value, confidence, created_at
		FROM facts
		WHERE key = ?
		ORDER BY created_at DESC
		LIMIT 1
	`, key)

	var fact models.Fact
	err := row.Scan(
		&fact.FactID,
		&fact.BlockID,
		&fact.TurnID,
		&fact.Key,
		&fact.Value,
		&fact.Confidence,
		&fact.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Not found, return nil without error
		}
		return nil, fmt.Errorf("failed to get fact: %w", err)
	}

	return &fact, nil
}

// DeleteFactByKey deletes all facts with the given key
func (s *Storage) DeleteFactByKey(key string) (int64, error) {
	result, err := s.db.Exec(`DELETE FROM facts WHERE key = ?`, key)
	if err != nil {
		return 0, fmt.Errorf("failed to delete fact: %w", err)
	}
	return result.RowsAffected()
}

// DeleteFactByID deletes a specific fact by its ID
func (s *Storage) DeleteFactByID(factID string) error {
	_, err := s.db.Exec(`DELETE FROM facts WHERE fact_id = ?`, factID)
	if err != nil {
		return fmt.Errorf("failed to delete fact: %w", err)
	}
	return nil
}

// SaveFact saves a single fact to the database (for direct fact insertion)
func (s *Storage) SaveFact(fact *models.Fact) error {
	_, err := s.db.Exec(`
		INSERT INTO facts (fact_id, block_id, turn_id, key, value, confidence, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, fact.FactID, fact.BlockID, fact.TurnID, fact.Key, fact.Value, fact.Confidence, fact.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to insert fact: %w", err)
	}
	return nil
}

// DeleteBridgeBlock deletes a bridge block and all its associated data
func (s *Storage) DeleteBridgeBlock(blockID string) error {
	// Delete associated facts first
	_, err := s.db.Exec(`DELETE FROM facts WHERE block_id = ?`, blockID)
	if err != nil {
		return fmt.Errorf("failed to delete block facts: %w", err)
	}

	// Delete associated embeddings
	_, err = s.db.Exec(`DELETE FROM embeddings WHERE block_id = ?`, blockID)
	if err != nil {
		return fmt.Errorf("failed to delete block embeddings: %w", err)
	}

	// Get block to find file path
	block, err := s.GetBridgeBlock(blockID)
	if err != nil {
		return fmt.Errorf("failed to get block for deletion: %w", err)
	}

	// Delete the JSON file
	blockFile := filepath.Join(s.basePath, "bridge_blocks", block.CreatedAt.Format("2006-01-02"), blockID+".json")
	if err := os.Remove(blockFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete block file: %w", err)
	}

	return nil
}
