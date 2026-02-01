// ABOUTME: Tests for Turn model creation and validation
// ABOUTME: Verifies NewTurn constructor and field handling
package models

import (
	"strings"
	"testing"
)

func TestNewTurn(t *testing.T) {
	tests := []struct {
		name        string
		userMessage string
		aiResponse  string
		keywords    []string
		topics      []string
		wantErr     bool
		errMsg      string
	}{
		{
			name:        "valid turn with all fields",
			userMessage: "What is Go?",
			aiResponse:  "Go is a programming language.",
			keywords:    []string{"go", "programming"},
			topics:      []string{"technology"},
			wantErr:     false,
		},
		{
			name:        "valid turn without keywords",
			userMessage: "Hello",
			aiResponse:  "Hi there!",
			keywords:    nil,
			topics:      nil,
			wantErr:     false,
		},
		{
			name:        "valid turn with empty keywords slice",
			userMessage: "Test message",
			aiResponse:  "Test response",
			keywords:    []string{},
			topics:      []string{},
			wantErr:     false,
		},
		{
			name:        "empty user message",
			userMessage: "",
			aiResponse:  "Response",
			keywords:    nil,
			topics:      nil,
			wantErr:     true,
			errMsg:      "user message cannot be empty",
		},
		{
			name:        "whitespace-only user message",
			userMessage: "   \t\n  ",
			aiResponse:  "Response",
			keywords:    nil,
			topics:      nil,
			wantErr:     true,
			errMsg:      "user message cannot be empty",
		},
		{
			name:        "valid turn with empty AI response",
			userMessage: "Question",
			aiResponse:  "",
			keywords:    nil,
			topics:      nil,
			wantErr:     false, // AI response can be empty
		},
		{
			name:        "valid turn with long message",
			userMessage: strings.Repeat("test ", 1000),
			aiResponse:  strings.Repeat("response ", 1000),
			keywords:    []string{"long", "message"},
			topics:      []string{"testing"},
			wantErr:     false,
		},
		{
			name:        "valid turn with special characters",
			userMessage: "Hello! How are you? :) #test @user",
			aiResponse:  "I'm doing well! Thanks for asking.",
			keywords:    []string{"greeting"},
			topics:      []string{"social"},
			wantErr:     false,
		},
		{
			name:        "valid turn with unicode",
			userMessage: "Hello \u4e16\u754c",
			aiResponse:  "\u3053\u3093\u306b\u3061\u306f",
			keywords:    []string{"international"},
			topics:      []string{"language"},
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			turn, err := NewTurn(tt.userMessage, tt.aiResponse, tt.keywords, tt.topics)

			if (err != nil) != tt.wantErr {
				t.Errorf("NewTurn() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil {
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("NewTurn() error = %q, want to contain %q", err.Error(), tt.errMsg)
				}
				return
			}

			// Verify turn fields
			if turn == nil {
				t.Fatal("NewTurn() returned nil turn without error")
			}
			if turn.UserMessage != tt.userMessage {
				t.Errorf("UserMessage = %q, want %q", turn.UserMessage, tt.userMessage)
			}
			if turn.AIResponse != tt.aiResponse {
				t.Errorf("AIResponse = %q, want %q", turn.AIResponse, tt.aiResponse)
			}

			// Check keywords
			if len(turn.Keywords) != len(tt.keywords) {
				t.Errorf("Keywords length = %d, want %d", len(turn.Keywords), len(tt.keywords))
			}
			for i, kw := range tt.keywords {
				if i < len(turn.Keywords) && turn.Keywords[i] != kw {
					t.Errorf("Keywords[%d] = %q, want %q", i, turn.Keywords[i], kw)
				}
			}

			// Check topics
			if len(turn.Topics) != len(tt.topics) {
				t.Errorf("Topics length = %d, want %d", len(turn.Topics), len(tt.topics))
			}
			for i, tp := range tt.topics {
				if i < len(turn.Topics) && turn.Topics[i] != tp {
					t.Errorf("Topics[%d] = %q, want %q", i, turn.Topics[i], tp)
				}
			}

			// Verify generated fields
			if turn.TurnID == "" {
				t.Error("TurnID should be generated")
			}
			if !strings.HasPrefix(turn.TurnID, "turn_") {
				t.Errorf("TurnID = %q, should start with 'turn_'", turn.TurnID)
			}
			if turn.Timestamp.IsZero() {
				t.Error("Timestamp should be set")
			}
		})
	}
}

func TestNewTurn_UniqueIDs(t *testing.T) {
	// Create multiple turns and verify they get unique IDs
	ids := make(map[string]bool)

	for i := 0; i < 10; i++ {
		turn, err := NewTurn("message", "response", nil, nil)
		if err != nil {
			t.Fatalf("NewTurn() error = %v", err)
		}

		if ids[turn.TurnID] {
			t.Errorf("Duplicate TurnID generated: %s", turn.TurnID)
		}
		ids[turn.TurnID] = true
	}
}

func TestTurn_Fields(t *testing.T) {
	// Test direct field assignment (for cases where Turn is created without NewTurn)
	turn := Turn{
		TurnID:      "turn_manual",
		UserMessage: "Manual message",
		AIResponse:  "Manual response",
		Keywords:    []string{"manual", "test"},
		Topics:      []string{"testing"},
	}

	if turn.TurnID != "turn_manual" {
		t.Errorf("TurnID = %q, want %q", turn.TurnID, "turn_manual")
	}
	if turn.UserMessage != "Manual message" {
		t.Errorf("UserMessage = %q, want %q", turn.UserMessage, "Manual message")
	}
	if turn.AIResponse != "Manual response" {
		t.Errorf("AIResponse = %q, want %q", turn.AIResponse, "Manual response")
	}
	if len(turn.Keywords) != 2 {
		t.Errorf("Keywords length = %d, want 2", len(turn.Keywords))
	}
	if len(turn.Topics) != 1 {
		t.Errorf("Topics length = %d, want 1", len(turn.Topics))
	}
}

func TestTurn_TurnIDFormat(t *testing.T) {
	turn, err := NewTurn("test", "response", nil, nil)
	if err != nil {
		t.Fatalf("NewTurn() error = %v", err)
	}

	// TurnID format should be: turn_YYYYMMDD_HHMMSS_<uuid>
	parts := strings.Split(turn.TurnID, "_")
	if len(parts) < 3 {
		t.Errorf("TurnID format unexpected: %s", turn.TurnID)
	}
	if parts[0] != "turn" {
		t.Errorf("TurnID should start with 'turn', got: %s", parts[0])
	}

	// Date part should be 8 digits
	if len(parts) >= 2 && len(parts[1]) != 8 {
		t.Errorf("TurnID date part should be 8 digits, got: %s", parts[1])
	}

	// Time part should be 6 digits
	if len(parts) >= 3 && len(parts[2]) != 6 {
		t.Errorf("TurnID time part should be 6 digits, got: %s", parts[2])
	}
}
