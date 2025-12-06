// ABOUTME: BridgeBlock represents a topic-based conversation thread in HMLR
// ABOUTME: Organizes turns by topic with routing metadata
package models

import "time"

// BridgeBlockStatus represents the current state of a Bridge Block
type BridgeBlockStatus string

const (
	StatusActive BridgeBlockStatus = "ACTIVE"
	StatusPaused BridgeBlockStatus = "PAUSED"
	StatusClosed BridgeBlockStatus = "CLOSED"
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
