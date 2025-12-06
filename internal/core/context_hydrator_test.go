// ABOUTME: Unit tests for ContextHydrator without requiring OpenAI API
// ABOUTME: Tests prompt assembly structure and token limiting logic
package core

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/harper/remember-standalone/internal/models"
	"github.com/harper/remember-standalone/internal/storage"
)

func TestContextHydrator_BasicPromptAssembly(t *testing.T) {
	// Setup temporary storage
	tmpDir := t.TempDir()
	os.Setenv("XDG_DATA_HOME", tmpDir)
	defer os.Unsetenv("XDG_DATA_HOME")

	store, err := storage.NewStorage()
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer store.Close()

	// Create a simple bridge block with conversation history
	turn1 := &models.Turn{
		TurnID:      "turn_" + uuid.New().String(),
		Timestamp:   time.Now(),
		UserMessage: "What is Python?",
		AIResponse:  "Python is a high-level programming language.",
		Keywords:    []string{"python", "programming"},
		Topics:      []string{"programming"},
	}

	blockID, err := store.StoreTurn(turn1)
	if err != nil {
		t.Fatalf("Failed to store turn: %v", err)
	}

	// Create ContextHydrator without vector storage (nil for embedding client)
	hydrator := NewContextHydrator(store, nil)

	t.Run("assembles basic prompt without vector search", func(t *testing.T) {
		userMessage := "Tell me more about Python frameworks"
		prompt, err := hydrator.HydrateBridgeBlock(blockID, userMessage, 4000)
		if err != nil {
			t.Fatalf("Failed to hydrate: %v", err)
		}

		// Verify system prompt
		if !strings.Contains(prompt, "SYSTEM:") {
			t.Error("Expected SYSTEM section")
		}

		// Verify conversation history
		if !strings.Contains(prompt, "CONVERSATION HISTORY:") {
			t.Error("Expected CONVERSATION HISTORY section")
		}

		if !strings.Contains(prompt, "What is Python?") {
			t.Error("Expected user message from history")
		}

		// Verify current message
		if !strings.Contains(prompt, "CURRENT USER MESSAGE:") {
			t.Error("Expected CURRENT USER MESSAGE section")
		}

		if !strings.Contains(prompt, userMessage) {
			t.Error("Expected current user message in prompt")
		}

		t.Logf("Generated prompt:\n%s", prompt)
	})

	t.Run("includes user profile when available", func(t *testing.T) {
		profile := &models.UserProfile{
			Name:             "TestUser",
			Preferences:      []string{"Python", "Go"},
			TopicsOfInterest: []string{"backend development"},
			LastUpdated:      time.Now(),
		}

		if err := store.SaveUserProfile(profile); err != nil {
			t.Fatalf("Failed to save profile: %v", err)
		}

		userMessage := "What should I learn?"
		prompt, err := hydrator.HydrateBridgeBlock(blockID, userMessage, 4000)
		if err != nil {
			t.Fatalf("Failed to hydrate: %v", err)
		}

		if !strings.Contains(prompt, "USER PROFILE:") {
			t.Error("Expected USER PROFILE section")
		}

		if !strings.Contains(prompt, "TestUser") {
			t.Error("Expected user name in profile")
		}

		if !strings.Contains(prompt, "Python") {
			t.Error("Expected preferences in profile")
		}
	})

	t.Run("token limiting truncates properly", func(t *testing.T) {
		// Very tight token limit
		userMessage := "Short question"
		prompt, err := hydrator.HydrateBridgeBlock(blockID, userMessage, 100)
		if err != nil {
			t.Fatalf("Failed to hydrate: %v", err)
		}

		// Approximate token count (4 chars ≈ 1 token)
		approxTokens := len(prompt) / 4

		t.Logf("Token limit: 100, Actual: ~%d, Chars: %d", approxTokens, len(prompt))

		// Should be under limit (with 20% tolerance)
		if approxTokens > 120 {
			t.Errorf("Token limit exceeded: ~%d tokens (limit: 100)", approxTokens)
		}

		// Essential parts should still be present
		if !strings.Contains(prompt, "SYSTEM:") {
			t.Error("System prompt should always be included")
		}

		if !strings.Contains(prompt, userMessage) {
			t.Error("Current message should always be included")
		}
	})
}

func TestContextHydrator_MultiTurnHistory(t *testing.T) {
	// Setup temporary storage
	tmpDir := t.TempDir()
	os.Setenv("XDG_DATA_HOME", tmpDir)
	defer os.Unsetenv("XDG_DATA_HOME")

	store, err := storage.NewStorage()
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer store.Close()

	// Create a block with multiple turns
	turn1 := &models.Turn{
		TurnID:      "turn_" + uuid.New().String(),
		Timestamp:   time.Now(),
		UserMessage: "First message",
		AIResponse:  "First response",
		Keywords:    []string{"test"},
		Topics:      []string{"test"},
	}

	blockID, err := store.StoreTurn(turn1)
	if err != nil {
		t.Fatalf("Failed to store turn1: %v", err)
	}

	turn2 := &models.Turn{
		TurnID:      "turn_" + uuid.New().String(),
		Timestamp:   time.Now().Add(1 * time.Minute),
		UserMessage: "Second message",
		AIResponse:  "Second response",
		Keywords:    []string{"test"},
		Topics:      []string{"test"},
	}

	if err := store.AppendTurnToBlock(blockID, turn2); err != nil {
		t.Fatalf("Failed to append turn2: %v", err)
	}

	hydrator := NewContextHydrator(store, nil)

	prompt, err := hydrator.HydrateBridgeBlock(blockID, "Third message", 4000)
	if err != nil {
		t.Fatalf("Failed to hydrate: %v", err)
	}

	// Verify both turns are in history
	if !strings.Contains(prompt, "First message") {
		t.Error("Expected first turn in history")
	}

	if !strings.Contains(prompt, "Second message") {
		t.Error("Expected second turn in history")
	}

	// Verify turn ordering
	firstIdx := strings.Index(prompt, "First message")
	secondIdx := strings.Index(prompt, "Second message")

	if firstIdx > secondIdx {
		t.Error("Expected turns in chronological order")
	}
}
