// ABOUTME: Tests for Fact model creation and validation
// ABOUTME: Verifies NewFact constructor and error conditions
package models

import (
	"strings"
	"testing"
)

func TestNewFact(t *testing.T) {
	tests := []struct {
		name       string
		blockID    string
		turnID     string
		key        string
		value      string
		confidence float64
		wantErr    bool
		errMsg     string
	}{
		{
			name:       "valid fact with all fields",
			blockID:    "block_001",
			turnID:     "turn_001",
			key:        "user_name",
			value:      "Alice",
			confidence: 1.0,
			wantErr:    false,
		},
		{
			name:       "valid fact without turn ID",
			blockID:    "block_001",
			turnID:     "",
			key:        "preference",
			value:      "dark mode",
			confidence: 0.9,
			wantErr:    false,
		},
		{
			name:       "empty block ID",
			blockID:    "",
			turnID:     "turn_001",
			key:        "key",
			value:      "value",
			confidence: 0.8,
			wantErr:    true,
			errMsg:     "blockID cannot be empty",
		},
		{
			name:       "empty key",
			blockID:    "block_001",
			turnID:     "turn_001",
			key:        "",
			value:      "value",
			confidence: 0.8,
			wantErr:    true,
			errMsg:     "key and value cannot be empty",
		},
		{
			name:       "empty value",
			blockID:    "block_001",
			turnID:     "turn_001",
			key:        "key",
			value:      "",
			confidence: 0.8,
			wantErr:    true,
			errMsg:     "key and value cannot be empty",
		},
		{
			name:       "both key and value empty",
			blockID:    "block_001",
			turnID:     "turn_001",
			key:        "",
			value:      "",
			confidence: 0.8,
			wantErr:    true,
			errMsg:     "key and value cannot be empty",
		},
		{
			name:       "confidence too low (negative)",
			blockID:    "block_001",
			turnID:     "turn_001",
			key:        "key",
			value:      "value",
			confidence: -0.1,
			wantErr:    true,
			errMsg:     "confidence must be 0.0-1.0",
		},
		{
			name:       "confidence too high",
			blockID:    "block_001",
			turnID:     "turn_001",
			key:        "key",
			value:      "value",
			confidence: 1.5,
			wantErr:    true,
			errMsg:     "confidence must be 0.0-1.0",
		},
		{
			name:       "confidence at lower boundary",
			blockID:    "block_001",
			turnID:     "turn_001",
			key:        "key",
			value:      "value",
			confidence: 0.0,
			wantErr:    false,
		},
		{
			name:       "confidence at upper boundary",
			blockID:    "block_001",
			turnID:     "turn_001",
			key:        "key",
			value:      "value",
			confidence: 1.0,
			wantErr:    false,
		},
		{
			name:       "confidence in middle",
			blockID:    "block_001",
			turnID:     "turn_001",
			key:        "key",
			value:      "value",
			confidence: 0.5,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fact, err := NewFact(tt.blockID, tt.turnID, tt.key, tt.value, tt.confidence)

			if (err != nil) != tt.wantErr {
				t.Errorf("NewFact() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil {
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("NewFact() error = %q, want to contain %q", err.Error(), tt.errMsg)
				}
				return
			}

			// Verify fact fields
			if fact == nil {
				t.Fatal("NewFact() returned nil fact without error")
			}
			if fact.BlockID != tt.blockID {
				t.Errorf("BlockID = %q, want %q", fact.BlockID, tt.blockID)
			}
			if fact.TurnID != tt.turnID {
				t.Errorf("TurnID = %q, want %q", fact.TurnID, tt.turnID)
			}
			if fact.Key != tt.key {
				t.Errorf("Key = %q, want %q", fact.Key, tt.key)
			}
			if fact.Value != tt.value {
				t.Errorf("Value = %q, want %q", fact.Value, tt.value)
			}
			if fact.Confidence != tt.confidence {
				t.Errorf("Confidence = %v, want %v", fact.Confidence, tt.confidence)
			}

			// Verify generated fields
			if fact.FactID == "" {
				t.Error("FactID should be generated")
			}
			if !strings.HasPrefix(fact.FactID, "fact_") {
				t.Errorf("FactID = %q, should start with 'fact_'", fact.FactID)
			}
			if fact.CreatedAt.IsZero() {
				t.Error("CreatedAt should be set")
			}
		})
	}
}

func TestNewFact_UniqueIDs(t *testing.T) {
	// Create multiple facts and verify they get unique IDs
	facts := make([]*Fact, 10)
	ids := make(map[string]bool)

	for i := 0; i < 10; i++ {
		fact, err := NewFact("block_001", "turn_001", "key", "value", 0.9)
		if err != nil {
			t.Fatalf("NewFact() error = %v", err)
		}
		facts[i] = fact

		if ids[fact.FactID] {
			t.Errorf("Duplicate FactID generated: %s", fact.FactID)
		}
		ids[fact.FactID] = true
	}
}

func TestFact_Fields(t *testing.T) {
	// Test direct field assignment (for cases where Fact is created without NewFact)
	fact := Fact{
		FactID:     "fact_manual",
		BlockID:    "block_001",
		TurnID:     "turn_001",
		Key:        "manual_key",
		Value:      "manual_value",
		Confidence: 0.75,
	}

	if fact.FactID != "fact_manual" {
		t.Errorf("FactID = %q, want %q", fact.FactID, "fact_manual")
	}
	if fact.BlockID != "block_001" {
		t.Errorf("BlockID = %q, want %q", fact.BlockID, "block_001")
	}
	if fact.TurnID != "turn_001" {
		t.Errorf("TurnID = %q, want %q", fact.TurnID, "turn_001")
	}
	if fact.Key != "manual_key" {
		t.Errorf("Key = %q, want %q", fact.Key, "manual_key")
	}
	if fact.Value != "manual_value" {
		t.Errorf("Value = %q, want %q", fact.Value, "manual_value")
	}
	if fact.Confidence != 0.75 {
		t.Errorf("Confidence = %v, want %v", fact.Confidence, 0.75)
	}
}
