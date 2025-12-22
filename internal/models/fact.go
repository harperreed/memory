// ABOUTME: Fact represents a key-value pair extracted from conversations
// ABOUTME: Stored in SQLite for efficient lookup
package models

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Fact represents an extracted key-value fact
type Fact struct {
	FactID     string    `json:"fact_id"`
	BlockID    string    `json:"block_id"`
	TurnID     string    `json:"turn_id"`
	Key        string    `json:"key"`
	Value      string    `json:"value"`
	Confidence float64   `json:"confidence"`
	CreatedAt  time.Time `json:"created_at"`
}

// NewFact creates a new Fact with validation
func NewFact(blockID, turnID, key, value string, confidence float64) (*Fact, error) {
	if blockID == "" {
		return nil, errors.New("fact blockID cannot be empty")
	}
	if key == "" || value == "" {
		return nil, errors.New("fact key and value cannot be empty")
	}
	if confidence < 0.0 || confidence > 1.0 {
		return nil, fmt.Errorf("confidence must be 0.0-1.0, got %f", confidence)
	}
	return &Fact{
		FactID:     generateFactID(),
		BlockID:    blockID,
		TurnID:     turnID,
		Key:        key,
		Value:      value,
		Confidence: confidence,
		CreatedAt:  time.Now().UTC(),
	}, nil
}

// generateFactID generates a unique fact identifier
func generateFactID() string {
	return fmt.Sprintf("fact_%s", uuid.New().String()[:8])
}
