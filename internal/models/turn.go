// ABOUTME: Turn represents a single conversation exchange between user and AI
// ABOUTME: Core data structure for HMLR memory system
package models

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Turn represents a single conversation turn
type Turn struct {
	TurnID      string    `json:"turn_id"`
	Timestamp   time.Time `json:"timestamp"`
	UserMessage string    `json:"user_message"`
	AIResponse  string    `json:"ai_response"`
	Keywords    []string  `json:"keywords,omitempty"`
	Topics      []string  `json:"topics,omitempty"`
}

// NewTurn creates a new Turn with validation
func NewTurn(userMessage, aiResponse string, keywords, topics []string) (*Turn, error) {
	if strings.TrimSpace(userMessage) == "" {
		return nil, errors.New("user message cannot be empty")
	}
	return &Turn{
		TurnID:      generateTurnID(),
		Timestamp:   time.Now().UTC(),
		UserMessage: userMessage,
		AIResponse:  aiResponse,
		Keywords:    keywords,
		Topics:      topics,
	}, nil
}

// generateTurnID generates a unique turn identifier
func generateTurnID() string {
	return fmt.Sprintf("turn_%s_%s", time.Now().Format("20060102_150405"), uuid.New().String()[:8])
}
