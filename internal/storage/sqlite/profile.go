// ABOUTME: User profile storage operations for SQLite
// ABOUTME: Implements singleton profile pattern with JSON array serialization
package sqlite

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/harper/remember-standalone/internal/models"
)

// ProfileStore handles user profile persistence
type ProfileStore struct {
	db *DB
}

// NewProfileStore creates a new ProfileStore
func NewProfileStore(db *DB) *ProfileStore {
	return &ProfileStore{db: db}
}

// Get retrieves the user profile, returning nil if not found
func (s *ProfileStore) Get() (*models.UserProfile, error) {
	var (
		name           sql.NullString
		prefsJSON      sql.NullString
		topicsJSON     sql.NullString
		updatedAt      time.Time
	)

	err := s.db.QueryRow(`
		SELECT name, preferences, topics_of_interest, updated_at
		FROM user_profile
		WHERE id = 1
	`).Scan(&name, &prefsJSON, &topicsJSON, &updatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	profile := &models.UserProfile{
		LastUpdated: updatedAt,
	}

	if name.Valid {
		profile.Name = name.String
	}

	if prefsJSON.Valid && prefsJSON.String != "" {
		if err := json.Unmarshal([]byte(prefsJSON.String), &profile.Preferences); err != nil {
			profile.Preferences = []string{}
		}
	} else {
		profile.Preferences = []string{}
	}

	if topicsJSON.Valid && topicsJSON.String != "" {
		if err := json.Unmarshal([]byte(topicsJSON.String), &profile.TopicsOfInterest); err != nil {
			profile.TopicsOfInterest = []string{}
		}
	} else {
		profile.TopicsOfInterest = []string{}
	}

	return profile, nil
}

// Save saves or updates the user profile (upsert)
func (s *ProfileStore) Save(profile *models.UserProfile) error {
	var prefsJSON, topicsJSON []byte
	var err error

	if profile.Preferences != nil {
		prefsJSON, err = json.Marshal(profile.Preferences)
		if err != nil {
			return err
		}
	} else {
		prefsJSON = []byte("[]")
	}

	if profile.TopicsOfInterest != nil {
		topicsJSON, err = json.Marshal(profile.TopicsOfInterest)
		if err != nil {
			return err
		}
	} else {
		topicsJSON = []byte("[]")
	}

	updatedAt := time.Now()
	if !profile.LastUpdated.IsZero() {
		updatedAt = profile.LastUpdated
	}

	_, err = s.db.Exec(`
		INSERT INTO user_profile (id, name, preferences, topics_of_interest, updated_at)
		VALUES (1, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			preferences = excluded.preferences,
			topics_of_interest = excluded.topics_of_interest,
			updated_at = excluded.updated_at
	`, profile.Name, string(prefsJSON), string(topicsJSON), updatedAt)

	return err
}
