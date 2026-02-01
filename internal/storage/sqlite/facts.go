// ABOUTME: Fact storage operations for SQLite
// ABOUTME: Implements CRUD and search operations for key-value facts
package sqlite

import (
	"database/sql"
	"time"

	"github.com/harper/remember-standalone/internal/models"
)

// FactStore handles fact persistence
type FactStore struct {
	db *DB
}

// NewFactStore creates a new FactStore
func NewFactStore(db *DB) *FactStore {
	return &FactStore{db: db}
}

// Save saves a fact
func (s *FactStore) Save(fact *models.Fact) error {
	createdAt := fact.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now()
	}

	_, err := s.db.Exec(`
		INSERT INTO facts (id, block_id, turn_id, key, value, confidence, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			block_id = excluded.block_id,
			turn_id = excluded.turn_id,
			key = excluded.key,
			value = excluded.value,
			confidence = excluded.confidence
	`, fact.FactID, nullString(fact.BlockID), nullString(fact.TurnID),
		fact.Key, fact.Value, fact.Confidence, createdAt)

	return err
}

// GetByID retrieves a fact by its ID
func (s *FactStore) GetByID(factID string) (*models.Fact, error) {
	var (
		fact    models.Fact
		blockID sql.NullString
		turnID  sql.NullString
	)

	err := s.db.QueryRow(`
		SELECT id, block_id, turn_id, key, value, confidence, created_at
		FROM facts
		WHERE id = ?
	`, factID).Scan(&fact.FactID, &blockID, &turnID, &fact.Key, &fact.Value,
		&fact.Confidence, &fact.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if blockID.Valid {
		fact.BlockID = blockID.String
	}
	if turnID.Valid {
		fact.TurnID = turnID.String
	}

	return &fact, nil
}

// GetByKey retrieves the most recent fact with the given key
func (s *FactStore) GetByKey(key string) (*models.Fact, error) {
	var (
		fact    models.Fact
		blockID sql.NullString
		turnID  sql.NullString
	)

	err := s.db.QueryRow(`
		SELECT id, block_id, turn_id, key, value, confidence, created_at
		FROM facts
		WHERE key = ?
		ORDER BY created_at DESC
		LIMIT 1
	`, key).Scan(&fact.FactID, &blockID, &turnID, &fact.Key, &fact.Value,
		&fact.Confidence, &fact.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if blockID.Valid {
		fact.BlockID = blockID.String
	}
	if turnID.Valid {
		fact.TurnID = turnID.String
	}

	return &fact, nil
}

// GetByBlock retrieves all facts for a block
func (s *FactStore) GetByBlock(blockID string) ([]models.Fact, error) {
	rows, err := s.db.Query(`
		SELECT id, block_id, turn_id, key, value, confidence, created_at
		FROM facts
		WHERE block_id = ?
		ORDER BY created_at DESC
	`, blockID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	return s.scanFacts(rows)
}

// Search searches facts by key or value containing the query string
func (s *FactStore) Search(query string, maxResults int) ([]models.Fact, error) {
	likePattern := "%" + query + "%"
	rows, err := s.db.Query(`
		SELECT id, block_id, turn_id, key, value, confidence, created_at
		FROM facts
		WHERE key LIKE ? OR value LIKE ?
		ORDER BY confidence DESC, created_at DESC
		LIMIT ?
	`, likePattern, likePattern, maxResults)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	return s.scanFacts(rows)
}

// DeleteByID deletes a fact by its ID
func (s *FactStore) DeleteByID(factID string) error {
	_, err := s.db.Exec("DELETE FROM facts WHERE id = ?", factID)
	return err
}

// DeleteByKey deletes all facts with the given key
func (s *FactStore) DeleteByKey(key string) (int64, error) {
	result, err := s.db.Exec("DELETE FROM facts WHERE key = ?", key)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// scanFacts scans rows into a slice of Fact
func (s *FactStore) scanFacts(rows *sql.Rows) ([]models.Fact, error) {
	var facts []models.Fact

	for rows.Next() {
		var (
			fact    models.Fact
			blockID sql.NullString
			turnID  sql.NullString
		)

		err := rows.Scan(&fact.FactID, &blockID, &turnID, &fact.Key, &fact.Value,
			&fact.Confidence, &fact.CreatedAt)
		if err != nil {
			return nil, err
		}

		if blockID.Valid {
			fact.BlockID = blockID.String
		}
		if turnID.Valid {
			fact.TurnID = turnID.String
		}

		facts = append(facts, fact)
	}

	return facts, rows.Err()
}

// nullString converts an empty string to sql.NullString
func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}
