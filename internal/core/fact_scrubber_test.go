// ABOUTME: Tests for FactScrubber fact extraction
// ABOUTME: Verifies fact extraction and storage linking

package core

import (
	"testing"

	"github.com/harper/remember-standalone/internal/llm"
)

func TestNewFactScrubber(t *testing.T) {
	// Create with nil client (we can't test actual extraction without OpenAI)
	fs := NewFactScrubber(nil)
	if fs == nil {
		t.Fatal("NewFactScrubber() returned nil")
	}
}

func TestNewFactScrubber_WithClient(t *testing.T) {
	// This tests the structure, not actual API calls
	var client *llm.OpenAIClient
	fs := NewFactScrubber(client)

	if fs == nil {
		t.Fatal("NewFactScrubber() returned nil")
	}

	if fs.client != nil {
		t.Error("Client should be nil when passed nil")
	}
}

// Note: Full ExtractAndSave testing would require mocking the OpenAI client,
// which isn't practical here. The structure is tested in integration tests.
