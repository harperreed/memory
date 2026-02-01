// ABOUTME: Bridge block storage operations for SQLite
// ABOUTME: Implements CRUD and query operations for conversation threads
package sqlite

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/harper/remember-standalone/internal/models"
)

// BlockStore handles bridge block persistence
type BlockStore struct {
	db *DB
}

// NewBlockStore creates a new BlockStore
func NewBlockStore(db *DB) *BlockStore {
	return &BlockStore{db: db}
}

// Save saves or updates a bridge block (upsert)
func (s *BlockStore) Save(block *models.BridgeBlock) error {
	keywordsJSON, err := json.Marshal(block.Keywords)
	if err != nil {
		return err
	}

	_, err = s.db.Exec(`
		INSERT INTO bridge_blocks (id, day_id, topic_label, keywords, status, summary, turn_count, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			day_id = excluded.day_id,
			topic_label = excluded.topic_label,
			keywords = excluded.keywords,
			status = excluded.status,
			summary = excluded.summary,
			turn_count = excluded.turn_count,
			updated_at = excluded.updated_at
	`, block.BlockID, block.DayID, block.TopicLabel, string(keywordsJSON), string(block.Status),
		block.Summary, block.TurnCount, block.CreatedAt, block.UpdatedAt)

	return err
}

// Get retrieves a bridge block by ID (without turns)
func (s *BlockStore) Get(blockID string) (*models.BridgeBlock, error) {
	var (
		block        models.BridgeBlock
		keywordsJSON sql.NullString
		summary      sql.NullString
		status       string
	)

	err := s.db.QueryRow(`
		SELECT id, day_id, topic_label, keywords, status, summary, turn_count, created_at, updated_at
		FROM bridge_blocks
		WHERE id = ?
	`, blockID).Scan(&block.BlockID, &block.DayID, &block.TopicLabel, &keywordsJSON,
		&status, &summary, &block.TurnCount, &block.CreatedAt, &block.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	block.Status = models.BridgeBlockStatus(status)

	if keywordsJSON.Valid && keywordsJSON.String != "" {
		if err := json.Unmarshal([]byte(keywordsJSON.String), &block.Keywords); err != nil {
			block.Keywords = []string{}
		}
	} else {
		block.Keywords = []string{}
	}

	if summary.Valid {
		block.Summary = summary.String
	}

	return &block, nil
}

// GetWithTurns retrieves a bridge block with all its turns
func (s *BlockStore) GetWithTurns(blockID string) (*models.BridgeBlock, error) {
	block, err := s.Get(blockID)
	if err != nil || block == nil {
		return block, err
	}

	turnStore := NewTurnStore(s.db)
	turns, err := turnStore.GetByBlock(blockID)
	if err != nil {
		return nil, err
	}

	block.Turns = turns
	block.TurnCount = len(turns)

	return block, nil
}

// GetByStatus retrieves all blocks with a specific status
func (s *BlockStore) GetByStatus(status models.BridgeBlockStatus) ([]models.BridgeBlock, error) {
	rows, err := s.db.Query(`
		SELECT id, day_id, topic_label, keywords, status, summary, turn_count, created_at, updated_at
		FROM bridge_blocks
		WHERE status = ?
		ORDER BY updated_at DESC
	`, string(status))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	return s.scanBlocks(rows)
}

// ListAll retrieves all bridge blocks
func (s *BlockStore) ListAll() ([]models.BridgeBlock, error) {
	rows, err := s.db.Query(`
		SELECT id, day_id, topic_label, keywords, status, summary, turn_count, created_at, updated_at
		FROM bridge_blocks
		ORDER BY updated_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	return s.scanBlocks(rows)
}

// Delete removes a bridge block (turns will cascade delete)
func (s *BlockStore) Delete(blockID string) error {
	_, err := s.db.Exec("DELETE FROM bridge_blocks WHERE id = ?", blockID)
	return err
}

// UpdateStatus updates only the status of a block
func (s *BlockStore) UpdateStatus(blockID string, status models.BridgeBlockStatus) error {
	_, err := s.db.Exec(`
		UPDATE bridge_blocks
		SET status = ?, updated_at = ?
		WHERE id = ?
	`, string(status), time.Now(), blockID)
	return err
}

// scanBlocks scans rows into a slice of BridgeBlock
func (s *BlockStore) scanBlocks(rows *sql.Rows) ([]models.BridgeBlock, error) {
	var blocks []models.BridgeBlock

	for rows.Next() {
		var (
			block        models.BridgeBlock
			keywordsJSON sql.NullString
			summary      sql.NullString
			status       string
		)

		err := rows.Scan(&block.BlockID, &block.DayID, &block.TopicLabel, &keywordsJSON,
			&status, &summary, &block.TurnCount, &block.CreatedAt, &block.UpdatedAt)
		if err != nil {
			return nil, err
		}

		block.Status = models.BridgeBlockStatus(status)

		if keywordsJSON.Valid && keywordsJSON.String != "" {
			if err := json.Unmarshal([]byte(keywordsJSON.String), &block.Keywords); err != nil {
				block.Keywords = []string{}
			}
		} else {
			block.Keywords = []string{}
		}

		if summary.Valid {
			block.Summary = summary.String
		}

		blocks = append(blocks, block)
	}

	return blocks, rows.Err()
}
