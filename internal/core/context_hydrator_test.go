// ABOUTME: Integration tests for ContextHydrator
// ABOUTME: Tests prompt assembly structure and token limiting logic
package core

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/harper/remember-standalone/internal/models"
	"github.com/harper/remember-standalone/internal/storage"
)

func TestContextHydrator_BasicPromptAssembly(t *testing.T) {
	store, err := storage.NewStorageInMemory()
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer func() { _ = store.Close() }()

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

		// Approximate token count (4 chars = 1 token)
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
	store, err := storage.NewStorageInMemory()
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer func() { _ = store.Close() }()

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

func TestContextHydrator_FormatRelevantFacts(t *testing.T) {
	store, err := storage.NewStorageInMemory()
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer func() { _ = store.Close() }()

	hydrator := NewContextHydrator(store, nil)

	facts := []models.Fact{
		{Key: "name", Value: "Alice", Confidence: 1.0},
		{Key: "favorite_color", Value: "blue", Confidence: 0.9},
	}

	result := hydrator.formatRelevantFacts(facts)

	if !strings.Contains(result, "RELEVANT FACTS") {
		t.Error("Expected RELEVANT FACTS header")
	}
	if !strings.Contains(result, "name") {
		t.Error("Expected name key in output")
	}
	if !strings.Contains(result, "Alice") {
		t.Error("Expected Alice value in output")
	}
	if !strings.Contains(result, "1.00") {
		t.Error("Expected confidence in output")
	}
}

func TestContextHydrator_FormatRetrievedMemories(t *testing.T) {
	store, err := storage.NewStorageInMemory()
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer func() { _ = store.Close() }()

	hydrator := NewContextHydrator(store, nil)

	memories := []models.MemorySearchResult{
		{
			BlockID:        "block_1",
			TopicLabel:     "Programming",
			RelevanceScore: 0.95,
			Summary:        "Discussion about Go",
		},
		{
			BlockID:        "block_2",
			TopicLabel:     "Testing",
			RelevanceScore: 0.85,
			Turns: []models.Turn{
				{UserMessage: "How to test?", AIResponse: "Use table-driven tests"},
			},
		},
	}

	result := hydrator.formatRetrievedMemories(memories)

	if !strings.Contains(result, "RETRIEVED MEMORIES") {
		t.Error("Expected RETRIEVED MEMORIES header")
	}
	if !strings.Contains(result, "Programming") {
		t.Error("Expected topic label in output")
	}
	if !strings.Contains(result, "0.95") {
		t.Error("Expected relevance score in output")
	}
	if !strings.Contains(result, "Discussion about Go") {
		t.Error("Expected summary in output")
	}
	if !strings.Contains(result, "How to test?") {
		t.Error("Expected turn user message in output")
	}
}

func TestContextHydrator_LimitTokens_Basic(t *testing.T) {
	store, err := storage.NewStorageInMemory()
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer func() { _ = store.Close() }()

	hydrator := NewContextHydrator(store, nil)

	// Create a long prompt that needs truncation
	longPrompt := "SYSTEM:\nYou are helpful.\n\nCONVERSATION HISTORY:\n"
	for i := 0; i < 100; i++ {
		longPrompt += "Turn content that should be trimmed. "
	}
	longPrompt += "\n\nCURRENT USER MESSAGE:\nShort question\n"

	userMessage := "Short question"

	// Call limitTokens with a small limit
	result := hydrator.limitTokens(longPrompt, userMessage, 50)

	// Should contain system prompt
	if !strings.Contains(result, "SYSTEM:") {
		t.Error("Expected SYSTEM in truncated output")
	}

	// Should contain user message
	if !strings.Contains(result, userMessage) {
		t.Error("Expected user message in truncated output")
	}
}

func TestContextHydrator_LimitTokens_UnderLimit(t *testing.T) {
	store, err := storage.NewStorageInMemory()
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer func() { _ = store.Close() }()

	hydrator := NewContextHydrator(store, nil)

	shortPrompt := "SYSTEM:\nYou are helpful.\n\nCURRENT USER MESSAGE:\nHello\n"
	userMessage := "Hello"

	// With large limit, should return unchanged
	result := hydrator.limitTokens(shortPrompt, userMessage, 1000)

	if result != shortPrompt {
		t.Error("Expected unchanged prompt when under limit")
	}
}

func TestContextHydrator_WithFacts(t *testing.T) {
	store, err := storage.NewStorageInMemory()
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create a block
	turn := &models.Turn{
		TurnID:      "turn_facts_test",
		Timestamp:   time.Now(),
		UserMessage: "What's my name?",
		AIResponse:  "Your name is Alice.",
		Keywords:    []string{"name"},
		Topics:      []string{"personal"},
	}
	blockID, err := store.StoreTurn(turn)
	if err != nil {
		t.Fatalf("StoreTurn() error = %v", err)
	}

	// Save a fact
	fact := &models.Fact{
		FactID:     "fact_name",
		Key:        "name",
		Value:      "Alice",
		Confidence: 1.0,
		CreatedAt:  time.Now(),
	}
	_ = store.SaveFact(fact)

	hydrator := NewContextHydrator(store, nil)

	prompt, err := hydrator.HydrateBridgeBlock(blockID, "name", 4000)
	if err != nil {
		t.Fatalf("HydrateBridgeBlock() error = %v", err)
	}

	// Should include facts section
	if !strings.Contains(prompt, "RELEVANT FACTS") {
		t.Error("Expected RELEVANT FACTS section when facts match")
	}
}

func TestContextHydrator_LimitTokens_WithSections(t *testing.T) {
	store, err := storage.NewStorageInMemory()
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer func() { _ = store.Close() }()

	hydrator := NewContextHydrator(store, nil)

	tests := []struct {
		name        string
		fullPrompt  string
		userMessage string
		maxTokens   int
		wantParts   []string
	}{
		{
			name: "prompt with all sections under limit",
			fullPrompt: `SYSTEM:
You are a helpful AI assistant with access to conversation history and context.

USER PROFILE:
Name: Alice

CONVERSATION HISTORY:
Topic: Programming

Turn 1:
User: Hello
AI: Hi there!

CURRENT USER MESSAGE:
How are you?
`,
			userMessage: "How are you?",
			maxTokens:   500,
			wantParts:   []string{"SYSTEM:", "USER PROFILE:", "CONVERSATION HISTORY:", "CURRENT USER MESSAGE:"},
		},
		{
			name: "prompt with retrieved memories",
			fullPrompt: `SYSTEM:
You are a helpful AI assistant with access to conversation history and context.

CONVERSATION HISTORY:
Topic: Test

RETRIEVED MEMORIES (from other conversations):

Memory 1 (Relevance: 0.95):
Topic: Past Topic
Summary: Old discussion

CURRENT USER MESSAGE:
Test message
`,
			userMessage: "Test message",
			maxTokens:   400,
			wantParts:   []string{"SYSTEM:", "CONVERSATION HISTORY:", "CURRENT USER MESSAGE:"},
		},
		{
			name: "prompt with relevant facts",
			fullPrompt: `SYSTEM:
You are a helpful AI assistant with access to conversation history and context.

CONVERSATION HISTORY:
Topic: Facts Test

RELEVANT FACTS:
- name: Alice (confidence: 1.00)

CURRENT USER MESSAGE:
What's my name?
`,
			userMessage: "What's my name?",
			maxTokens:   400,
			wantParts:   []string{"SYSTEM:", "CONVERSATION HISTORY:", "CURRENT USER MESSAGE:"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hydrator.limitTokens(tt.fullPrompt, tt.userMessage, tt.maxTokens)

			for _, part := range tt.wantParts {
				if !strings.Contains(result, part) {
					t.Errorf("Expected result to contain %q", part)
				}
			}
		})
	}
}

func TestContextHydrator_LimitTokens_ExtremelySmall(t *testing.T) {
	store, err := storage.NewStorageInMemory()
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer func() { _ = store.Close() }()

	hydrator := NewContextHydrator(store, nil)

	longPrompt := `SYSTEM:
You are a helpful AI assistant with access to conversation history and context.

USER PROFILE:
Name: Test User

CONVERSATION HISTORY:
Topic: Test

Turn 1:
User: Very long message that should be truncated
AI: Response

CURRENT USER MESSAGE:
A very very very long message that exceeds the token limit significantly
`

	// Extremely small token limit
	result := hydrator.limitTokens(longPrompt, "A very very very long message that exceeds the token limit significantly", 20)

	// Should at minimum have SYSTEM section
	if !strings.Contains(result, "SYSTEM:") {
		t.Error("Expected SYSTEM section in truncated output")
	}
}

func TestContextHydrator_FormatUserProfile_AllFields(t *testing.T) {
	store, err := storage.NewStorageInMemory()
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer func() { _ = store.Close() }()

	hydrator := NewContextHydrator(store, nil)

	profile := &models.UserProfile{
		Name:             "Doctor Biz",
		Preferences:      []string{"TDD", "Go", "Simple code"},
		TopicsOfInterest: []string{"AI", "MCP", "CLI tools"},
	}

	result := hydrator.formatUserProfile(profile)

	if !strings.Contains(result, "USER PROFILE:") {
		t.Error("Expected USER PROFILE header")
	}
	if !strings.Contains(result, "Doctor Biz") {
		t.Error("Expected name in profile")
	}
	if !strings.Contains(result, "TDD") {
		t.Error("Expected preferences in profile")
	}
	if !strings.Contains(result, "AI") {
		t.Error("Expected topics in profile")
	}
}

func TestContextHydrator_FormatBlockHistory_EmptyTurns(t *testing.T) {
	store, err := storage.NewStorageInMemory()
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer func() { _ = store.Close() }()

	hydrator := NewContextHydrator(store, nil)

	block := &models.BridgeBlock{
		BlockID:    "block_empty",
		TopicLabel: "Empty Topic",
		Turns:      []models.Turn{},
	}

	result := hydrator.formatBlockHistory(block)

	if !strings.Contains(result, "CONVERSATION HISTORY:") {
		t.Error("Expected CONVERSATION HISTORY header")
	}
	if !strings.Contains(result, "Empty Topic") {
		t.Error("Expected topic label")
	}
}

func TestContextHydrator_FormatBlockHistory_MultipleTurns(t *testing.T) {
	store, err := storage.NewStorageInMemory()
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer func() { _ = store.Close() }()

	hydrator := NewContextHydrator(store, nil)

	block := &models.BridgeBlock{
		BlockID:    "block_multi",
		TopicLabel: "Multi Turn Topic",
		Turns: []models.Turn{
			{UserMessage: "First question", AIResponse: "First answer"},
			{UserMessage: "Second question", AIResponse: "Second answer"},
			{UserMessage: "Third question", AIResponse: "Third answer"},
		},
	}

	result := hydrator.formatBlockHistory(block)

	if !strings.Contains(result, "Turn 1:") {
		t.Error("Expected Turn 1")
	}
	if !strings.Contains(result, "Turn 2:") {
		t.Error("Expected Turn 2")
	}
	if !strings.Contains(result, "Turn 3:") {
		t.Error("Expected Turn 3")
	}
	if !strings.Contains(result, "First question") {
		t.Error("Expected first question")
	}
	if !strings.Contains(result, "Third answer") {
		t.Error("Expected third answer")
	}
}

func TestContextHydrator_FormatRetrievedMemories_WithSummary(t *testing.T) {
	store, err := storage.NewStorageInMemory()
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer func() { _ = store.Close() }()

	hydrator := NewContextHydrator(store, nil)

	memories := []models.MemorySearchResult{
		{
			BlockID:        "block_with_summary",
			TopicLabel:     "Summarized Topic",
			RelevanceScore: 0.88,
			Summary:        "This is a summary of the conversation",
		},
	}

	result := hydrator.formatRetrievedMemories(memories)

	if !strings.Contains(result, "RETRIEVED MEMORIES") {
		t.Error("Expected RETRIEVED MEMORIES header")
	}
	if !strings.Contains(result, "This is a summary") {
		t.Error("Expected summary in output")
	}
	if !strings.Contains(result, "0.88") {
		t.Error("Expected relevance score in output")
	}
}

func TestContextHydrator_FormatRelevantFacts_MultipleFacts(t *testing.T) {
	store, err := storage.NewStorageInMemory()
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer func() { _ = store.Close() }()

	hydrator := NewContextHydrator(store, nil)

	facts := []models.Fact{
		{Key: "api_key_location", Value: "1Password", Confidence: 1.0},
		{Key: "preferred_editor", Value: "vim", Confidence: 0.95},
		{Key: "favorite_language", Value: "Go", Confidence: 0.9},
	}

	result := hydrator.formatRelevantFacts(facts)

	if !strings.Contains(result, "RELEVANT FACTS") {
		t.Error("Expected RELEVANT FACTS header")
	}
	if !strings.Contains(result, "api_key_location") {
		t.Error("Expected first fact key")
	}
	if !strings.Contains(result, "1Password") {
		t.Error("Expected first fact value")
	}
	if !strings.Contains(result, "vim") {
		t.Error("Expected second fact value")
	}
	if !strings.Contains(result, "Go") {
		t.Error("Expected third fact value")
	}
}

func TestContextHydrator_LimitTokens_TruncatesMessage(t *testing.T) {
	store, err := storage.NewStorageInMemory()
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer func() { _ = store.Close() }()

	hydrator := NewContextHydrator(store, nil)

	// Create a prompt where even essentials exceed limit - should truncate message
	systemPrompt := "SYSTEM:\nYou are helpful.\n\n"
	veryLongMessage := strings.Repeat("word ", 200) // Very long message
	longPrompt := systemPrompt + "CURRENT USER MESSAGE:\n" + veryLongMessage + "\n"

	// Token limit that allows system but requires message truncation
	result := hydrator.limitTokens(longPrompt, veryLongMessage, 30)

	// Should contain SYSTEM
	if !strings.Contains(result, "SYSTEM:") {
		t.Error("Expected SYSTEM section in output")
	}

	// Result should be truncated
	if len(result) >= len(longPrompt) {
		t.Error("Expected truncated result to be shorter than original")
	}
}

func TestContextHydrator_LimitTokens_AllSections(t *testing.T) {
	store, err := storage.NewStorageInMemory()
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer func() { _ = store.Close() }()

	hydrator := NewContextHydrator(store, nil)

	// Create a prompt with all sections
	fullPrompt := `SYSTEM:
You are a helpful AI assistant with access to conversation history and context.

USER PROFILE:
Name: Alice
Preferences: TDD, Go

CONVERSATION HISTORY:
Topic: Testing

Turn 1:
User: How do I test?
AI: Use table-driven tests.

RETRIEVED MEMORIES (from other conversations):

Memory 1 (Relevance: 0.90):
Topic: Previous Discussion
Summary: We discussed testing before.

RELEVANT FACTS:
- name: Alice (confidence: 1.00)
- editor: vim (confidence: 0.95)

CURRENT USER MESSAGE:
Tell me about testing
`
	userMessage := "Tell me about testing"

	// Large enough limit to include everything
	result := hydrator.limitTokens(fullPrompt, userMessage, 1000)

	// Should be unchanged when under limit
	if result != fullPrompt {
		t.Error("Expected unchanged prompt when under limit")
	}

	// Now test with tight limit - should prioritize conversation history
	result = hydrator.limitTokens(fullPrompt, userMessage, 150)

	// Essential parts should still be present
	if !strings.Contains(result, "SYSTEM:") {
		t.Error("Expected SYSTEM section")
	}
	if !strings.Contains(result, userMessage) {
		t.Error("Expected user message")
	}
}

func TestContextHydrator_LimitTokens_OnlyRetrievedMemories(t *testing.T) {
	store, err := storage.NewStorageInMemory()
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer func() { _ = store.Close() }()

	hydrator := NewContextHydrator(store, nil)

	// Prompt with RETRIEVED MEMORIES but no RELEVANT FACTS
	fullPrompt := `SYSTEM:
You are helpful.

CONVERSATION HISTORY:
Topic: Test

Turn 1:
User: Hi
AI: Hello!

RETRIEVED MEMORIES (from other conversations):

Memory 1 (Relevance: 0.95):
Topic: Past
Summary: Past discussion

CURRENT USER MESSAGE:
Test
`
	userMessage := "Test"

	result := hydrator.limitTokens(fullPrompt, userMessage, 500)

	if !strings.Contains(result, "SYSTEM:") {
		t.Error("Expected SYSTEM section")
	}
	if !strings.Contains(result, "CONVERSATION HISTORY:") {
		t.Error("Expected CONVERSATION HISTORY section")
	}
}

func TestContextHydrator_LimitTokens_OnlyRelevantFacts(t *testing.T) {
	store, err := storage.NewStorageInMemory()
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer func() { _ = store.Close() }()

	hydrator := NewContextHydrator(store, nil)

	// Prompt with RELEVANT FACTS but no RETRIEVED MEMORIES
	fullPrompt := `SYSTEM:
You are helpful.

CONVERSATION HISTORY:
Topic: Test

Turn 1:
User: Hi
AI: Hello!

RELEVANT FACTS:
- name: Alice (confidence: 1.00)

CURRENT USER MESSAGE:
Test
`
	userMessage := "Test"

	result := hydrator.limitTokens(fullPrompt, userMessage, 500)

	if !strings.Contains(result, "SYSTEM:") {
		t.Error("Expected SYSTEM section")
	}
	if !strings.Contains(result, "RELEVANT FACTS:") {
		t.Error("Expected RELEVANT FACTS section")
	}
}

func TestContextHydrator_LimitTokens_UserProfileAtEnd(t *testing.T) {
	store, err := storage.NewStorageInMemory()
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer func() { _ = store.Close() }()

	hydrator := NewContextHydrator(store, nil)

	// Prompt with USER PROFILE (which has lower priority in truncation)
	fullPrompt := `SYSTEM:
You are helpful.

USER PROFILE:
Name: Alice

CONVERSATION HISTORY:
Topic: Test

CURRENT USER MESSAGE:
Test
`
	userMessage := "Test"

	result := hydrator.limitTokens(fullPrompt, userMessage, 500)

	if !strings.Contains(result, "SYSTEM:") {
		t.Error("Expected SYSTEM section")
	}
	if !strings.Contains(result, "CONVERSATION HISTORY:") {
		t.Error("Expected CONVERSATION HISTORY section")
	}
}

func TestContextHydrator_FormatUserProfile_NoPreferences(t *testing.T) {
	store, err := storage.NewStorageInMemory()
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer func() { _ = store.Close() }()

	hydrator := NewContextHydrator(store, nil)

	profile := &models.UserProfile{
		Name:             "Bob",
		Preferences:      []string{},
		TopicsOfInterest: []string{},
	}

	result := hydrator.formatUserProfile(profile)

	if !strings.Contains(result, "USER PROFILE:") {
		t.Error("Expected USER PROFILE header")
	}
	if !strings.Contains(result, "Bob") {
		t.Error("Expected name in profile")
	}
	// Should not contain Preferences line when empty
	if strings.Contains(result, "Preferences:") {
		t.Error("Should not show Preferences when empty")
	}
}

func TestContextHydrator_FormatUserProfile_OnlyTopics(t *testing.T) {
	store, err := storage.NewStorageInMemory()
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer func() { _ = store.Close() }()

	hydrator := NewContextHydrator(store, nil)

	profile := &models.UserProfile{
		Name:             "Charlie",
		Preferences:      nil,
		TopicsOfInterest: []string{"AI", "ML"},
	}

	result := hydrator.formatUserProfile(profile)

	if !strings.Contains(result, "USER PROFILE:") {
		t.Error("Expected USER PROFILE header")
	}
	if !strings.Contains(result, "Charlie") {
		t.Error("Expected name in profile")
	}
	if !strings.Contains(result, "Topics of Interest:") {
		t.Error("Expected Topics of Interest")
	}
	if !strings.Contains(result, "AI") {
		t.Error("Expected AI topic")
	}
}

func TestContextHydrator_FormatRetrievedMemories_NoSummaryFallback(t *testing.T) {
	store, err := storage.NewStorageInMemory()
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer func() { _ = store.Close() }()

	hydrator := NewContextHydrator(store, nil)

	// Memory with no summary but has turns - should use first turn
	memories := []models.MemorySearchResult{
		{
			BlockID:        "block_no_summary",
			TopicLabel:     "Topic Without Summary",
			RelevanceScore: 0.75,
			Summary:        "", // No summary
			Turns: []models.Turn{
				{UserMessage: "First user message", AIResponse: "First AI response"},
				{UserMessage: "Second user message", AIResponse: "Second AI response"},
			},
		},
	}

	result := hydrator.formatRetrievedMemories(memories)

	if !strings.Contains(result, "RETRIEVED MEMORIES") {
		t.Error("Expected RETRIEVED MEMORIES header")
	}
	// Should include the first turn since no summary
	if !strings.Contains(result, "First user message") {
		t.Error("Expected first turn user message")
	}
	if !strings.Contains(result, "First AI response") {
		t.Error("Expected first turn AI response")
	}
	// Should not include second turn
	if strings.Contains(result, "Second user message") {
		t.Error("Should only include first turn, not second")
	}
}

func TestContextHydrator_FormatRetrievedMemories_NoSummaryNoTurns(t *testing.T) {
	store, err := storage.NewStorageInMemory()
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer func() { _ = store.Close() }()

	hydrator := NewContextHydrator(store, nil)

	// Memory with neither summary nor turns
	memories := []models.MemorySearchResult{
		{
			BlockID:        "block_empty",
			TopicLabel:     "Empty Topic",
			RelevanceScore: 0.60,
			Summary:        "",
			Turns:          []models.Turn{},
		},
	}

	result := hydrator.formatRetrievedMemories(memories)

	if !strings.Contains(result, "RETRIEVED MEMORIES") {
		t.Error("Expected RETRIEVED MEMORIES header")
	}
	if !strings.Contains(result, "Empty Topic") {
		t.Error("Expected topic label")
	}
	if !strings.Contains(result, "0.60") {
		t.Error("Expected relevance score")
	}
}

func TestContextHydrator_LimitTokens_MessageTruncation(t *testing.T) {
	store, err := storage.NewStorageInMemory()
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer func() { _ = store.Close() }()

	hydrator := NewContextHydrator(store, nil)

	// systemPrompt is: "SYSTEM:\nYou are a helpful AI assistant with access to conversation history and context.\n\n"
	// That's about 90 chars, so 23 tokens
	// For truncation to happen, we need:
	// 1. essentialChars > maxChars (system + message is too large)
	// 2. remaining > 100 (so truncated message can be added)
	// remaining = maxChars - len(systemPrompt) = maxChars - 90
	// Need remaining > 100, so maxChars > 190, meaning maxTokens > 47

	// Message long enough to exceed limit
	userMessage := strings.Repeat("This is a very long message that will exceed the token limit significantly. ", 20)
	longPrompt := "SYSTEM:\nYou are a helpful AI assistant with access to conversation history and context.\n\n"
	longPrompt += "CURRENT USER MESSAGE:\n" + userMessage + "\n"

	// maxTokens = 60 => maxChars = 240
	// systemPrompt = ~90 chars
	// remaining = 240 - 90 = 150 > 100, so truncation path is taken
	result := hydrator.limitTokens(longPrompt, userMessage, 60)

	// Should have system
	if !strings.Contains(result, "SYSTEM:") {
		t.Error("Expected SYSTEM section")
	}

	// Should have truncated indicator
	if !strings.Contains(result, "[truncated]") {
		t.Errorf("Expected [truncated] indicator in output, got: %s", result)
	}
}

func TestContextHydrator_LimitTokens_SystemPromptOnly(t *testing.T) {
	store, err := storage.NewStorageInMemory()
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer func() { _ = store.Close() }()

	hydrator := NewContextHydrator(store, nil)

	// Token limit so tiny that not even system + minimal message fits
	userMessage := strings.Repeat("word ", 1000)
	longPrompt := "SYSTEM:\nYou are a helpful AI assistant with access to conversation history and context.\n\n"
	longPrompt += "CURRENT USER MESSAGE:\n" + userMessage + "\n"

	// Extremely small limit - only system prompt should fit
	result := hydrator.limitTokens(longPrompt, userMessage, 15) // ~60 chars

	// Should just have SYSTEM section
	if !strings.Contains(result, "SYSTEM:") {
		t.Error("Expected SYSTEM section")
	}
}

func TestContextHydrator_LimitTokens_SectionPriority(t *testing.T) {
	store, err := storage.NewStorageInMemory()
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer func() { _ = store.Close() }()

	hydrator := NewContextHydrator(store, nil)

	// Build a prompt with all optional sections to test priority dropping
	fullPrompt := `SYSTEM:
You are a helpful AI assistant with access to conversation history and context.

USER PROFILE:
Name: TestUser
Preferences: A, B, C

CONVERSATION HISTORY:
Topic: Testing Topic

Turn 1:
User: Test question
AI: Test answer

RETRIEVED MEMORIES (from other conversations):

Memory 1 (Relevance: 0.90):
Topic: Old Topic
Summary: Old conversation summary

RELEVANT FACTS:
- key1: value1 (confidence: 1.00)

CURRENT USER MESSAGE:
Current test message
`
	userMessage := "Current test message"

	// Limit that forces section dropping
	result := hydrator.limitTokens(fullPrompt, userMessage, 100)

	// Essential always present
	if !strings.Contains(result, "SYSTEM:") {
		t.Error("Expected SYSTEM section")
	}
	if !strings.Contains(result, userMessage) {
		t.Error("Expected user message")
	}
}

func TestContextHydrator_LimitTokens_ConversationHistoryOnly(t *testing.T) {
	store, err := storage.NewStorageInMemory()
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer func() { _ = store.Close() }()

	hydrator := NewContextHydrator(store, nil)

	// Prompt with CONVERSATION HISTORY going directly to CURRENT USER MESSAGE
	fullPrompt := `SYSTEM:
You are a helpful AI assistant with access to conversation history and context.

CONVERSATION HISTORY:
Topic: Test

Turn 1:
User: Hello
AI: Hi!

CURRENT USER MESSAGE:
Test message
`
	userMessage := "Test message"

	result := hydrator.limitTokens(fullPrompt, userMessage, 500)

	if !strings.Contains(result, "CONVERSATION HISTORY:") {
		t.Error("Expected CONVERSATION HISTORY section")
	}
}

func TestContextHydrator_LimitTokens_RetrievedMemoriesToCurrentMessage(t *testing.T) {
	store, err := storage.NewStorageInMemory()
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer func() { _ = store.Close() }()

	hydrator := NewContextHydrator(store, nil)

	// Prompt with RETRIEVED MEMORIES going directly to CURRENT USER MESSAGE (no RELEVANT FACTS)
	fullPrompt := `SYSTEM:
You are a helpful AI assistant with access to conversation history and context.

CONVERSATION HISTORY:
Topic: Test

Turn 1:
User: Hello
AI: Hi!

RETRIEVED MEMORIES (from other conversations):

Memory 1 (Relevance: 0.90):
Topic: Past
Summary: Past discussion

CURRENT USER MESSAGE:
Test message
`
	userMessage := "Test message"

	result := hydrator.limitTokens(fullPrompt, userMessage, 500)

	if !strings.Contains(result, "RETRIEVED MEMORIES") {
		t.Error("Expected RETRIEVED MEMORIES section")
	}
}
