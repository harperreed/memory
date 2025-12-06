// ABOUTME: Turn represents a single conversation exchange between user and AI
// ABOUTME: Core data structure for HMLR memory system
package models

import "time"

// Turn represents a single conversation turn
type Turn struct {
	TurnID      string    `json:"turn_id"`
	Timestamp   time.Time `json:"timestamp"`
	UserMessage string    `json:"user_message"`
	AIResponse  string    `json:"ai_response"`
	Keywords    []string  `json:"keywords,omitempty"`
	Topics      []string  `json:"topics,omitempty"`
}
