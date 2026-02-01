// ABOUTME: Tests for BridgeBlock model
// ABOUTME: Verifies validation and turn management

package models

import (
	"testing"
	"time"
)

func TestBridgeBlock_Validate(t *testing.T) {
	tests := []struct {
		name    string
		block   BridgeBlock
		wantErr bool
	}{
		{
			name: "valid block",
			block: BridgeBlock{
				BlockID:    "block_001",
				TopicLabel: "programming",
				Status:     StatusActive,
			},
			wantErr: false,
		},
		{
			name: "empty block ID",
			block: BridgeBlock{
				BlockID:    "",
				TopicLabel: "programming",
				Status:     StatusActive,
			},
			wantErr: true,
		},
		{
			name: "empty topic label",
			block: BridgeBlock{
				BlockID:    "block_001",
				TopicLabel: "",
				Status:     StatusActive,
			},
			wantErr: true,
		},
		{
			name: "invalid status",
			block: BridgeBlock{
				BlockID:    "block_001",
				TopicLabel: "programming",
				Status:     BridgeBlockStatus("INVALID"),
			},
			wantErr: true,
		},
		{
			name: "status PAUSED",
			block: BridgeBlock{
				BlockID:    "block_001",
				TopicLabel: "programming",
				Status:     StatusPaused,
			},
			wantErr: false,
		},
		{
			name: "status CLOSED",
			block: BridgeBlock{
				BlockID:    "block_001",
				TopicLabel: "programming",
				Status:     StatusClosed,
			},
			wantErr: false,
		},
		{
			name: "status ARCHIVED",
			block: BridgeBlock{
				BlockID:    "block_001",
				TopicLabel: "programming",
				Status:     StatusArchived,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.block.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBridgeBlock_AddTurn(t *testing.T) {
	block := &BridgeBlock{
		BlockID:    "block_001",
		TopicLabel: "programming",
		Status:     StatusActive,
		Turns:      []Turn{},
		UpdatedAt:  time.Now().Add(-1 * time.Hour),
	}
	oldUpdatedAt := block.UpdatedAt

	turn := Turn{
		TurnID:      "turn_001",
		UserMessage: "Hello",
		AIResponse:  "Hi there!",
		Timestamp:   time.Now(),
	}

	time.Sleep(1 * time.Millisecond) // Ensure time difference

	block.AddTurn(turn)

	if len(block.Turns) != 1 {
		t.Errorf("Turns length = %d, want 1", len(block.Turns))
	}
	if block.TurnCount != 1 {
		t.Errorf("TurnCount = %d, want 1", block.TurnCount)
	}
	if !block.UpdatedAt.After(oldUpdatedAt) {
		t.Error("UpdatedAt should be updated after AddTurn")
	}
	if block.Turns[0].TurnID != "turn_001" {
		t.Errorf("Turn ID = %q, want turn_001", block.Turns[0].TurnID)
	}
}

func TestBridgeBlock_AddTurn_Multiple(t *testing.T) {
	block := &BridgeBlock{
		BlockID:    "block_001",
		TopicLabel: "programming",
		Status:     StatusActive,
		Turns:      []Turn{},
	}

	for i := 0; i < 5; i++ {
		turn := Turn{
			TurnID:      "turn_" + string(rune('a'+i)),
			UserMessage: "Message",
		}
		block.AddTurn(turn)
	}

	if len(block.Turns) != 5 {
		t.Errorf("Turns length = %d, want 5", len(block.Turns))
	}
	if block.TurnCount != 5 {
		t.Errorf("TurnCount = %d, want 5", block.TurnCount)
	}
}

func TestBridgeBlockStatus_Constants(t *testing.T) {
	// Verify the string values of status constants
	if StatusActive != "ACTIVE" {
		t.Errorf("StatusActive = %q, want %q", StatusActive, "ACTIVE")
	}
	if StatusPaused != "PAUSED" {
		t.Errorf("StatusPaused = %q, want %q", StatusPaused, "PAUSED")
	}
	if StatusClosed != "CLOSED" {
		t.Errorf("StatusClosed = %q, want %q", StatusClosed, "CLOSED")
	}
	if StatusArchived != "ARCHIVED" {
		t.Errorf("StatusArchived = %q, want %q", StatusArchived, "ARCHIVED")
	}
}

func TestBridgeBlock_Fields(t *testing.T) {
	now := time.Now()
	block := BridgeBlock{
		BlockID:    "block_001",
		DayID:      "2026-02-01",
		TopicLabel: "programming",
		Keywords:   []string{"go", "testing"},
		Status:     StatusActive,
		CreatedAt:  now,
		UpdatedAt:  now,
		Summary:    "A discussion about Go",
		TurnCount:  3,
	}

	if block.BlockID != "block_001" {
		t.Errorf("BlockID = %q, want %q", block.BlockID, "block_001")
	}
	if block.DayID != "2026-02-01" {
		t.Errorf("DayID = %q, want %q", block.DayID, "2026-02-01")
	}
	if block.TopicLabel != "programming" {
		t.Errorf("TopicLabel = %q, want %q", block.TopicLabel, "programming")
	}
	if len(block.Keywords) != 2 {
		t.Errorf("Keywords length = %d, want 2", len(block.Keywords))
	}
	if block.Summary != "A discussion about Go" {
		t.Errorf("Summary = %q, want %q", block.Summary, "A discussion about Go")
	}
}
