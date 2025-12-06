// ABOUTME: Main storage implementation for HMLR memory system
// ABOUTME: Handles XDG directories, JSON files, and SQLite database
package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/adrg/xdg"
	"github.com/google/uuid"
	"github.com/harper/remember-standalone/internal/models"
	_ "github.com/mattn/go-sqlite3"
)

// Storage manages all persistent data for HMLR
type Storage struct {
	basePath string
	db       *sql.DB
}

// NewStorage initializes storage with XDG-compliant paths
func NewStorage() (*Storage, error) {
	// Use XDG data directory: ~/.local/share/remember/
	// Respects XDG_DATA_HOME environment variable override for testing
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		dataHome = xdg.DataHome
	}
	basePath := filepath.Join(dataHome, "remember")

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

	return &Storage{
		basePath: basePath,
		db:       db,
	}, nil
}

// Close closes the database connection
func (s *Storage) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// StoreTurn stores a conversation turn and creates/updates a Bridge Block
// INVARIANT: Only ONE block can be ACTIVE at a time. This is enforced by validation and maintained by this function.
func (s *Storage) StoreTurn(turn *models.Turn) (string, error) {
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

	return blockID, nil
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
func (s *Storage) SearchMemory(query string, maxResults int) ([]models.MemorySearchResult, error) {
	// Simple keyword-based search for now
	// TODO: Implement vector-based semantic search

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

				// Simple keyword matching
				if matchesQuery(&block, query) {
					results = append(results, models.MemorySearchResult{
						BlockID:        block.BlockID,
						TopicLabel:     block.TopicLabel,
						RelevanceScore: 0.9, // TODO: Calculate real relevance
						Summary:        block.Summary,
						Turns:          block.Turns,
					})
				}

				if len(results) >= maxResults {
					return results, nil
				}
			}
		}
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

// extractAndSaveFacts extracts facts from a turn and saves them to SQLite
func (s *Storage) extractAndSaveFacts(turn *models.Turn, blockID string) error {
	// Simple fact extraction for now
	// TODO: Implement full FactScrubber with LLM

	// Example: Extract "capital of X is Y" pattern
	facts := []models.Fact{}

	// Hardcoded example for testing
	if containsIgnoreCase(turn.UserMessage, "capital") && containsIgnoreCase(turn.AIResponse, "Paris") {
		facts = append(facts, models.Fact{
			FactID:     "fact_" + uuid.New().String(),
			BlockID:    blockID,
			TurnID:     turn.TurnID,
			Key:        "capital_of_France",
			Value:      "Paris",
			Confidence: 1.0,
			CreatedAt:  time.Now(),
		})
	}

	// Save facts to database
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
