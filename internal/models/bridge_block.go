// ABOUTME: BridgeBlock represents a topic-based conversation thread in HMLR
// ABOUTME: Organizes turns by topic with routing metadata
package models

import (
	"errors"
	"time"
)

// BridgeBlockStatus represents the current state of a Bridge Block
type BridgeBlockStatus string

const (
	StatusActive   BridgeBlockStatus = "ACTIVE"
	StatusPaused   BridgeBlockStatus = "PAUSED"
	StatusClosed   BridgeBlockStatus = "CLOSED"
	StatusArchived BridgeBlockStatus = "ARCHIVED"
)

// BridgeBlock represents a topic-based conversation thread
type BridgeBlock struct {
	BlockID     string            `json:"block_id"`
	DayID       string            `json:"day_id"`
	TopicLabel  string            `json:"topic_label"`
	Keywords    []string          `json:"keywords"`
	Status      BridgeBlockStatus `json:"status"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	Turns       []Turn            `json:"turns"`
	Summary     string            `json:"summary,omitempty"`
	TurnCount   int               `json:"turn_count"`
}

// Validate checks if the BridgeBlock has valid data
func (b *BridgeBlock) Validate() error {
	if b.BlockID == "" {
		return errors.New("block ID cannot be empty")
	}
	if b.TopicLabel == "" {
		return errors.New("topic label cannot be empty")
	}
	if b.Status != StatusActive && b.Status != StatusPaused &&
	   b.Status != StatusClosed && b.Status != StatusArchived {
		return errors.New("invalid status")
	}
	return nil
}

// AddTurn appends a turn to the bridge block and updates metadata
func (b *BridgeBlock) AddTurn(turn Turn) {
	b.Turns = append(b.Turns, turn)
	b.TurnCount = len(b.Turns)
	b.UpdatedAt = time.Now().UTC()
}
