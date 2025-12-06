// ABOUTME: Fact represents a key-value pair extracted from conversations
// ABOUTME: Stored in SQLite for efficient lookup
package models

import "time"

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
