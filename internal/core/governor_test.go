// ABOUTME: Tests for Governor smart routing logic
// ABOUTME: Verifies routing scenarios: continuation, resumption, new topic, topic shift

package core

import (
	"testing"
	"time"

	"github.com/harper/remember-standalone/internal/models"
	"github.com/harper/remember-standalone/internal/storage"
)

func TestNewGovernor(t *testing.T) {
	store, err := storage.NewStorageInMemory()
	if err != nil {
		t.Fatalf("NewStorageInMemory() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	gov := NewGovernor(store)
	if gov == nil {
		t.Fatal("NewGovernor() returned nil")
	}

	// Default threshold should be 0.3
	if gov.topicMatchThreshold != 0.3 {
		t.Errorf("topicMatchThreshold = %v, want 0.3", gov.topicMatchThreshold)
	}
}

func TestGovernor_Scenario3_NewTopicFirst(t *testing.T) {
	store, err := storage.NewStorageInMemory()
	if err != nil {
		t.Fatalf("NewStorageInMemory() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	gov := NewGovernor(store)

	// No blocks exist yet
	turn := &models.Turn{
		TurnID:      "turn_001",
		UserMessage: "Hello world",
		Keywords:    []string{"greeting"},
		Topics:      []string{"casual"},
	}

	decision, err := gov.Route(turn)
	if err != nil {
		t.Fatalf("Route() error = %v", err)
	}

	if decision.Scenario != models.NewTopicFirst {
		t.Errorf("Scenario = %v, want NewTopicFirst", decision.Scenario)
	}

	if decision.MatchedBlockID != "" {
		t.Errorf("MatchedBlockID should be empty, got %q", decision.MatchedBlockID)
	}

	if decision.ActiveBlockID != "" {
		t.Errorf("ActiveBlockID should be empty, got %q", decision.ActiveBlockID)
	}
}

func TestGovernor_Scenario1_TopicContinuation(t *testing.T) {
	store, err := storage.NewStorageInMemory()
	if err != nil {
		t.Fatalf("NewStorageInMemory() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create an active block
	existingTurn := &models.Turn{
		TurnID:      "turn_existing",
		Timestamp:   time.Now(),
		UserMessage: "Let's talk about Go programming",
		Keywords:    []string{"go", "programming"},
		Topics:      []string{"programming"},
	}
	blockID, err := store.StoreTurn(existingTurn)
	if err != nil {
		t.Fatalf("StoreTurn() error = %v", err)
	}

	gov := NewGovernor(store)

	// New turn with same topic
	turn := &models.Turn{
		TurnID:      "turn_new",
		UserMessage: "What about error handling in Go?",
		Keywords:    []string{"go", "errors"},
		Topics:      []string{"programming"},
	}

	decision, err := gov.Route(turn)
	if err != nil {
		t.Fatalf("Route() error = %v", err)
	}

	if decision.Scenario != models.TopicContinuation {
		t.Errorf("Scenario = %v, want TopicContinuation", decision.Scenario)
	}

	if decision.MatchedBlockID != blockID {
		t.Errorf("MatchedBlockID = %q, want %q", decision.MatchedBlockID, blockID)
	}
}

func TestGovernor_Scenario4_TopicShift(t *testing.T) {
	store, err := storage.NewStorageInMemory()
	if err != nil {
		t.Fatalf("NewStorageInMemory() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create an active block about programming
	existingTurn := &models.Turn{
		TurnID:      "turn_existing",
		Timestamp:   time.Now(),
		UserMessage: "Let's talk about Go programming",
		Keywords:    []string{"go", "programming"},
		Topics:      []string{"programming"},
	}
	blockID, err := store.StoreTurn(existingTurn)
	if err != nil {
		t.Fatalf("StoreTurn() error = %v", err)
	}

	gov := NewGovernor(store)

	// New turn with completely different topic
	turn := &models.Turn{
		TurnID:      "turn_cooking",
		UserMessage: "What's a good recipe for pasta?",
		Keywords:    []string{"cooking", "pasta", "recipe"},
		Topics:      []string{"cooking"},
	}

	decision, err := gov.Route(turn)
	if err != nil {
		t.Fatalf("Route() error = %v", err)
	}

	if decision.Scenario != models.TopicShift {
		t.Errorf("Scenario = %v, want TopicShift", decision.Scenario)
	}

	if decision.MatchedBlockID != "" {
		t.Errorf("MatchedBlockID should be empty for shift, got %q", decision.MatchedBlockID)
	}

	if decision.ActiveBlockID != blockID {
		t.Errorf("ActiveBlockID = %q, want %q", decision.ActiveBlockID, blockID)
	}
}

func TestGovernor_Scenario2_TopicResumption(t *testing.T) {
	store, err := storage.NewStorageInMemory()
	if err != nil {
		t.Fatalf("NewStorageInMemory() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create first block (will become paused)
	turn1 := &models.Turn{
		TurnID:      "turn_go",
		Timestamp:   time.Now().Add(-1 * time.Hour),
		UserMessage: "Let's talk about Go programming",
		Keywords:    []string{"go", "programming"},
		Topics:      []string{"go-programming"},
	}
	block1ID, err := store.StoreTurn(turn1)
	if err != nil {
		t.Fatalf("StoreTurn() error = %v", err)
	}

	// Create second block (will be active, pausing the first)
	turn2 := &models.Turn{
		TurnID:      "turn_cooking",
		Timestamp:   time.Now(),
		UserMessage: "What about cooking?",
		Keywords:    []string{"cooking", "food"},
		Topics:      []string{"cooking"},
	}
	_, err = store.StoreTurn(turn2)
	if err != nil {
		t.Fatalf("StoreTurn() error = %v", err)
	}

	// Verify first block is paused
	block1, err := store.GetBridgeBlock(block1ID)
	if err != nil {
		t.Fatalf("GetBridgeBlock() error = %v", err)
	}
	if block1.Status != models.StatusPaused {
		t.Errorf("Block1 status = %v, want PAUSED", block1.Status)
	}

	gov := NewGovernor(store)

	// Resume Go programming topic
	turn3 := &models.Turn{
		TurnID:      "turn_resume_go",
		UserMessage: "Let's continue with Go programming",
		Keywords:    []string{"go", "programming"},
		Topics:      []string{"go-programming"},
	}

	decision, err := gov.Route(turn3)
	if err != nil {
		t.Fatalf("Route() error = %v", err)
	}

	if decision.Scenario != models.TopicResumption {
		t.Errorf("Scenario = %v, want TopicResumption", decision.Scenario)
	}

	if decision.MatchedBlockID != block1ID {
		t.Errorf("MatchedBlockID = %q, want %q", decision.MatchedBlockID, block1ID)
	}
}

func TestGovernor_KeywordMatch(t *testing.T) {
	store, err := storage.NewStorageInMemory()
	if err != nil {
		t.Fatalf("NewStorageInMemory() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	gov := NewGovernor(store)

	tests := []struct {
		name  string
		k1    string
		k2    string
		match bool
	}{
		{"exact match", "golang", "golang", true},
		{"case insensitive", "Golang", "golang", true},
		{"case insensitive reverse", "golang", "GOLANG", true},
		{"different words", "golang", "python", false},
		{"partial match", "go", "golang", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gov.keywordMatch(tt.k1, tt.k2)
			if result != tt.match {
				t.Errorf("keywordMatch(%q, %q) = %v, want %v", tt.k1, tt.k2, result, tt.match)
			}
		})
	}
}

func TestGovernor_MatchesTopic_ByTopicLabel(t *testing.T) {
	store, err := storage.NewStorageInMemory()
	if err != nil {
		t.Fatalf("NewStorageInMemory() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	gov := NewGovernor(store)

	block := &models.BridgeBlock{
		BlockID:    "block_001",
		TopicLabel: "programming",
		Keywords:   []string{"code", "development"},
	}

	turn := &models.Turn{
		TurnID:   "turn_001",
		Keywords: []string{"different", "keywords"},
		Topics:   []string{"programming"}, // Matches block's TopicLabel
	}

	if !gov.matchesTopic(turn, block) {
		t.Error("matchesTopic() should match by topic label")
	}
}

func TestGovernor_MatchesTopic_ByKeywordOverlap(t *testing.T) {
	store, err := storage.NewStorageInMemory()
	if err != nil {
		t.Fatalf("NewStorageInMemory() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	gov := NewGovernor(store)

	block := &models.BridgeBlock{
		BlockID:    "block_001",
		TopicLabel: "other-topic",
		Keywords:   []string{"go", "programming", "backend"},
	}

	tests := []struct {
		name     string
		keywords []string
		match    bool
	}{
		{
			name:     "high overlap (2 of 3 = 66%)",
			keywords: []string{"go", "programming", "unrelated"},
			match:    true,
		},
		{
			name:     "threshold overlap (1 of 3 = 33%)",
			keywords: []string{"go", "unrelated", "other"},
			match:    true,
		},
		{
			name:     "low overlap (1 of 4 = 25%)",
			keywords: []string{"go", "unrelated", "other", "stuff"},
			match:    false,
		},
		{
			name:     "no overlap",
			keywords: []string{"cooking", "food", "recipe"},
			match:    false,
		},
		{
			name:     "empty turn keywords",
			keywords: []string{},
			match:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			turn := &models.Turn{
				TurnID:   "turn_test",
				Keywords: tt.keywords,
				Topics:   []string{"different"},
			}

			result := gov.matchesTopic(turn, block)
			if result != tt.match {
				t.Errorf("matchesTopic() = %v, want %v", result, tt.match)
			}
		})
	}
}

func TestGovernor_MatchesTopic_EmptyBlockKeywords(t *testing.T) {
	store, err := storage.NewStorageInMemory()
	if err != nil {
		t.Fatalf("NewStorageInMemory() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	gov := NewGovernor(store)

	block := &models.BridgeBlock{
		BlockID:    "block_001",
		TopicLabel: "other-topic",
		Keywords:   []string{}, // Empty
	}

	turn := &models.Turn{
		TurnID:   "turn_001",
		Keywords: []string{"go", "programming"},
		Topics:   []string{"different"},
	}

	// Should not match (no keyword overlap possible)
	if gov.matchesTopic(turn, block) {
		t.Error("matchesTopic() should return false when block has no keywords")
	}
}
