// ABOUTME: Turn storage operations for SQLite
// ABOUTME: Implements CRUD operations for conversation turns
package sqlite

import (
	"database/sql"
	"encoding/json"

	"github.com/harper/remember-standalone/internal/models"
)

// TurnStore handles turn persistence
type TurnStore struct {
	db *DB
}

// NewTurnStore creates a new TurnStore
func NewTurnStore(db *DB) *TurnStore {
	return &TurnStore{db: db}
}

// Save saves a turn for a block
func (s *TurnStore) Save(blockID string, turn *models.Turn) error {
	keywordsJSON, err := json.Marshal(turn.Keywords)
	if err != nil {
		return err
	}

	topicsJSON, err := json.Marshal(turn.Topics)
	if err != nil {
		return err
	}

	_, err = s.db.Exec(`
		INSERT INTO turns (id, block_id, user_message, ai_response, keywords, topics, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			user_message = excluded.user_message,
			ai_response = excluded.ai_response,
			keywords = excluded.keywords,
			topics = excluded.topics
	`, turn.TurnID, blockID, turn.UserMessage, turn.AIResponse,
		string(keywordsJSON), string(topicsJSON), turn.Timestamp)

	return err
}

// GetByBlock retrieves all turns for a block
func (s *TurnStore) GetByBlock(blockID string) ([]models.Turn, error) {
	rows, err := s.db.Query(`
		SELECT id, user_message, ai_response, keywords, topics, created_at
		FROM turns
		WHERE block_id = ?
		ORDER BY created_at ASC
	`, blockID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var turns []models.Turn
	for rows.Next() {
		var (
			turn         models.Turn
			keywordsJSON sql.NullString
			topicsJSON   sql.NullString
		)

		err := rows.Scan(&turn.TurnID, &turn.UserMessage, &turn.AIResponse,
			&keywordsJSON, &topicsJSON, &turn.Timestamp)
		if err != nil {
			return nil, err
		}

		if keywordsJSON.Valid && keywordsJSON.String != "" {
			if err := json.Unmarshal([]byte(keywordsJSON.String), &turn.Keywords); err != nil {
				turn.Keywords = []string{}
			}
		} else {
			turn.Keywords = []string{}
		}

		if topicsJSON.Valid && topicsJSON.String != "" {
			if err := json.Unmarshal([]byte(topicsJSON.String), &turn.Topics); err != nil {
				turn.Topics = []string{}
			}
		} else {
			turn.Topics = []string{}
		}

		turns = append(turns, turn)
	}

	return turns, rows.Err()
}

// Delete removes a specific turn
func (s *TurnStore) Delete(turnID string) error {
	_, err := s.db.Exec("DELETE FROM turns WHERE id = ?", turnID)
	return err
}
