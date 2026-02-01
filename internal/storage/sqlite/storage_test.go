// ABOUTME: Tests for unified Storage wrapper
// ABOUTME: Verifies high-level storage operations match old interface
package sqlite

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/harper/remember-standalone/internal/models"
)

func TestStorageInMemory(t *testing.T) {
	store, err := NewStorageInMemory()
	if err != nil {
		t.Fatalf("NewStorageInMemory() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	// Verify we can use the storage
	profile, err := store.GetUserProfile()
	if err != nil {
		t.Fatalf("GetUserProfile() error = %v", err)
	}
	if profile != nil {
		t.Error("Expected nil profile initially")
	}
}

func TestStoreTurn(t *testing.T) {
	store, err := NewStorageInMemory()
	if err != nil {
		t.Fatalf("NewStorageInMemory() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	turn := &models.Turn{
		TurnID:      "turn_test_001",
		Timestamp:   time.Now(),
		UserMessage: "Hello, how are you?",
		AIResponse:  "I'm doing well, thank you!",
		Keywords:    []string{"greeting", "hello"},
		Topics:      []string{"casual conversation"},
	}

	blockID, err := store.StoreTurn(turn)
	if err != nil {
		t.Fatalf("StoreTurn() error = %v", err)
	}
	if blockID == "" {
		t.Error("StoreTurn() returned empty blockID")
	}

	// Verify block was created
	block, err := store.GetBridgeBlock(blockID)
	if err != nil {
		t.Fatalf("GetBridgeBlock() error = %v", err)
	}
	if block == nil {
		t.Fatal("GetBridgeBlock() returned nil")
	}

	if block.Status != models.StatusActive {
		t.Errorf("Block status = %v, want ACTIVE", block.Status)
	}
	if len(block.Turns) != 1 {
		t.Errorf("Block turns = %v, want 1", len(block.Turns))
	}
}

func TestAppendTurnToBlock(t *testing.T) {
	store, err := NewStorageInMemory()
	if err != nil {
		t.Fatalf("NewStorageInMemory() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create initial block
	turn1 := &models.Turn{
		TurnID:      "turn_1",
		Timestamp:   time.Now(),
		UserMessage: "First message",
		AIResponse:  "First response",
		Keywords:    []string{"first"},
		Topics:      []string{"topic1"},
	}

	blockID, err := store.StoreTurn(turn1)
	if err != nil {
		t.Fatalf("StoreTurn() error = %v", err)
	}

	// Append another turn
	turn2 := &models.Turn{
		TurnID:      "turn_2",
		Timestamp:   time.Now(),
		UserMessage: "Second message",
		AIResponse:  "Second response",
		Keywords:    []string{"second"},
		Topics:      []string{"topic1"},
	}

	err = store.AppendTurnToBlock(blockID, turn2)
	if err != nil {
		t.Fatalf("AppendTurnToBlock() error = %v", err)
	}

	// Verify
	block, err := store.GetBridgeBlock(blockID)
	if err != nil {
		t.Fatalf("GetBridgeBlock() error = %v", err)
	}
	if len(block.Turns) != 2 {
		t.Errorf("Block turns = %v, want 2", len(block.Turns))
	}
}

func TestActiveBlockInvariant(t *testing.T) {
	store, err := NewStorageInMemory()
	if err != nil {
		t.Fatalf("NewStorageInMemory() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create first block
	turn1 := &models.Turn{
		TurnID:      "turn_1",
		Timestamp:   time.Now(),
		UserMessage: "Message 1",
		Keywords:    []string{"test"},
		Topics:      []string{"topic1"},
	}
	blockID1, err := store.StoreTurn(turn1)
	if err != nil {
		t.Fatalf("StoreTurn() error = %v", err)
	}

	// Create second block - first should be paused
	turn2 := &models.Turn{
		TurnID:      "turn_2",
		Timestamp:   time.Now(),
		UserMessage: "Message 2",
		Keywords:    []string{"test"},
		Topics:      []string{"topic2"},
	}
	blockID2, err := store.StoreTurn(turn2)
	if err != nil {
		t.Fatalf("StoreTurn() error = %v", err)
	}

	// Check invariant
	active, err := store.GetActiveBridgeBlocks()
	if err != nil {
		t.Fatalf("GetActiveBridgeBlocks() error = %v", err)
	}
	if len(active) != 1 {
		t.Errorf("Active blocks = %v, want 1 (invariant violated)", len(active))
	}
	if active[0].BlockID != blockID2 {
		t.Errorf("Active block = %v, want %v (newest)", active[0].BlockID, blockID2)
	}

	// First block should be paused
	block1, err := store.GetBridgeBlock(blockID1)
	if err != nil {
		t.Fatalf("GetBridgeBlock() error = %v", err)
	}
	if block1.Status != models.StatusPaused {
		t.Errorf("First block status = %v, want PAUSED", block1.Status)
	}
}

func TestFactOperations(t *testing.T) {
	store, err := NewStorageInMemory()
	if err != nil {
		t.Fatalf("NewStorageInMemory() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	// Save fact
	fact := &models.Fact{
		FactID:     "fact_test",
		Key:        "user_name",
		Value:      "Harper",
		Confidence: 1.0,
		CreatedAt:  time.Now(),
	}
	err = store.SaveFact(fact)
	if err != nil {
		t.Fatalf("SaveFact() error = %v", err)
	}

	// Get by key
	retrieved, err := store.GetFactByKey("user_name")
	if err != nil {
		t.Fatalf("GetFactByKey() error = %v", err)
	}
	if retrieved == nil {
		t.Fatal("GetFactByKey() returned nil")
	}
	if retrieved.Value != "Harper" {
		t.Errorf("Value = %v, want Harper", retrieved.Value)
	}

	// Delete
	count, err := store.DeleteFactByKey("user_name")
	if err != nil {
		t.Fatalf("DeleteFactByKey() error = %v", err)
	}
	if count != 1 {
		t.Errorf("Deleted count = %v, want 1", count)
	}
}

func TestProfileOperations(t *testing.T) {
	store, err := NewStorageInMemory()
	if err != nil {
		t.Fatalf("NewStorageInMemory() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	// Initially nil
	profile, err := store.GetUserProfile()
	if err != nil {
		t.Fatalf("GetUserProfile() error = %v", err)
	}
	if profile != nil {
		t.Error("Expected nil profile initially")
	}

	// Save profile
	newProfile := &models.UserProfile{
		Name:             "Doctor Biz",
		Preferences:      []string{"TDD", "dark mode"},
		TopicsOfInterest: []string{"Go", "distributed systems"},
	}
	err = store.SaveUserProfile(newProfile)
	if err != nil {
		t.Fatalf("SaveUserProfile() error = %v", err)
	}

	// Retrieve
	retrieved, err := store.GetUserProfile()
	if err != nil {
		t.Fatalf("GetUserProfile() error = %v", err)
	}
	if retrieved == nil {
		t.Fatal("GetUserProfile() returned nil after save")
	}
	if retrieved.Name != "Doctor Biz" {
		t.Errorf("Name = %v, want Doctor Biz", retrieved.Name)
	}
}

func TestDeleteBridgeBlock(t *testing.T) {
	store, err := NewStorageInMemory()
	if err != nil {
		t.Fatalf("NewStorageInMemory() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	turn := &models.Turn{
		TurnID:      "turn_delete",
		Timestamp:   time.Now(),
		UserMessage: "Test message",
		Topics:      []string{"test"},
	}

	blockID, err := store.StoreTurn(turn)
	if err != nil {
		t.Fatalf("StoreTurn() error = %v", err)
	}

	// Delete
	err = store.DeleteBridgeBlock(blockID)
	if err != nil {
		t.Fatalf("DeleteBridgeBlock() error = %v", err)
	}

	// Verify deleted
	block, err := store.GetBridgeBlock(blockID)
	if err != nil {
		t.Fatalf("GetBridgeBlock() error = %v", err)
	}
	if block != nil {
		t.Error("Block should be nil after delete")
	}
}

func TestRepairActiveBlockInvariant(t *testing.T) {
	store, err := NewStorageInMemory()
	if err != nil {
		t.Fatalf("NewStorageInMemory() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	// Manually create multiple active blocks (simulating corruption)
	block1 := &models.BridgeBlock{
		BlockID:   "block_1",
		DayID:     "2026-01-31",
		Status:    models.StatusActive,
		CreatedAt: time.Now().Add(-1 * time.Hour),
		UpdatedAt: time.Now().Add(-1 * time.Hour),
	}
	block2 := &models.BridgeBlock{
		BlockID:   "block_2",
		DayID:     "2026-01-31",
		Status:    models.StatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	_ = store.blocks.Save(block1)
	_ = store.blocks.Save(block2)

	// Verify we have 2 active (corrupted state)
	active, _ := store.GetActiveBridgeBlocks()
	if len(active) != 2 {
		t.Fatalf("Expected 2 active blocks, got %d", len(active))
	}

	// Repair
	repaired, err := store.RepairActiveBlockInvariant()
	if err != nil {
		t.Fatalf("RepairActiveBlockInvariant() error = %v", err)
	}
	if !repaired {
		t.Error("Expected repair to be needed")
	}

	// Verify invariant is now satisfied
	active, _ = store.GetActiveBridgeBlocks()
	if len(active) != 1 {
		t.Errorf("After repair, active blocks = %v, want 1", len(active))
	}
	if active[0].BlockID != "block_2" {
		t.Errorf("Active block = %v, want block_2 (newest)", active[0].BlockID)
	}
}

func TestRepairActiveBlockInvariant_NoRepairNeeded(t *testing.T) {
	store, err := NewStorageInMemory()
	if err != nil {
		t.Fatalf("NewStorageInMemory() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create only one active block
	block := &models.BridgeBlock{
		BlockID:   "block_single",
		DayID:     "2026-01-31",
		Status:    models.StatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	_ = store.blocks.Save(block)

	// Repair should not be needed
	repaired, err := store.RepairActiveBlockInvariant()
	if err != nil {
		t.Fatalf("RepairActiveBlockInvariant() error = %v", err)
	}
	if repaired {
		t.Error("Expected no repair needed with single active block")
	}
}

func TestSearchMemory(t *testing.T) {
	store, err := NewStorageInMemory()
	if err != nil {
		t.Fatalf("NewStorageInMemory() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create some blocks with keywords
	turn1 := &models.Turn{
		TurnID:      "turn_search_1",
		Timestamp:   time.Now(),
		UserMessage: "Let's talk about Go programming",
		Keywords:    []string{"go", "programming"},
		Topics:      []string{"programming"},
	}
	_, err = store.StoreTurn(turn1)
	if err != nil {
		t.Fatalf("StoreTurn() error = %v", err)
	}

	turn2 := &models.Turn{
		TurnID:      "turn_search_2",
		Timestamp:   time.Now(),
		UserMessage: "What about cooking recipes?",
		Keywords:    []string{"cooking", "recipes"},
		Topics:      []string{"cooking"},
	}
	_, err = store.StoreTurn(turn2)
	if err != nil {
		t.Fatalf("StoreTurn() error = %v", err)
	}

	// Search for programming
	results, err := store.SearchMemory("go", 10)
	if err != nil {
		t.Fatalf("SearchMemory() error = %v", err)
	}

	if len(results) == 0 {
		t.Error("Expected at least one result for 'go' query")
	}

	// Verify programming result is found
	found := false
	for _, r := range results {
		if r.TopicLabel == "programming" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected programming topic in results")
	}
}

func TestGetPausedBridgeBlocks(t *testing.T) {
	store, err := NewStorageInMemory()
	if err != nil {
		t.Fatalf("NewStorageInMemory() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create first turn (becomes active then paused)
	turn1 := &models.Turn{
		TurnID:      "turn_paused_1",
		Timestamp:   time.Now(),
		UserMessage: "First topic",
		Topics:      []string{"first"},
	}
	_, err = store.StoreTurn(turn1)
	if err != nil {
		t.Fatalf("StoreTurn() error = %v", err)
	}

	// Create second turn (causes first to pause)
	turn2 := &models.Turn{
		TurnID:      "turn_paused_2",
		Timestamp:   time.Now(),
		UserMessage: "Second topic",
		Topics:      []string{"second"},
	}
	_, err = store.StoreTurn(turn2)
	if err != nil {
		t.Fatalf("StoreTurn() error = %v", err)
	}

	// Should have one paused block
	paused, err := store.GetPausedBridgeBlocks()
	if err != nil {
		t.Fatalf("GetPausedBridgeBlocks() error = %v", err)
	}

	if len(paused) != 1 {
		t.Errorf("Paused blocks = %d, want 1", len(paused))
	}
}

func TestUpdateBridgeBlockStatus(t *testing.T) {
	store, err := NewStorageInMemory()
	if err != nil {
		t.Fatalf("NewStorageInMemory() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	turn := &models.Turn{
		TurnID:      "turn_status",
		Timestamp:   time.Now(),
		UserMessage: "Status test",
		Topics:      []string{"test"},
	}
	blockID, err := store.StoreTurn(turn)
	if err != nil {
		t.Fatalf("StoreTurn() error = %v", err)
	}

	// Update status to archived
	err = store.UpdateBridgeBlockStatus(blockID, models.StatusArchived)
	if err != nil {
		t.Fatalf("UpdateBridgeBlockStatus() error = %v", err)
	}

	// Verify
	block, err := store.GetBridgeBlock(blockID)
	if err != nil {
		t.Fatalf("GetBridgeBlock() error = %v", err)
	}
	if block.Status != models.StatusArchived {
		t.Errorf("Status = %v, want ARCHIVED", block.Status)
	}
}

func TestSaveFacts(t *testing.T) {
	store, err := NewStorageInMemory()
	if err != nil {
		t.Fatalf("NewStorageInMemory() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	facts := []models.Fact{
		{FactID: "fact_1", Key: "name", Value: "Alice", Confidence: 1.0, CreatedAt: time.Now()},
		{FactID: "fact_2", Key: "color", Value: "blue", Confidence: 0.9, CreatedAt: time.Now()},
	}

	err = store.SaveFacts(facts)
	if err != nil {
		t.Fatalf("SaveFacts() error = %v", err)
	}

	// Verify facts were saved
	retrieved, err := store.GetFactByKey("name")
	if err != nil {
		t.Fatalf("GetFactByKey() error = %v", err)
	}
	if retrieved == nil || retrieved.Value != "Alice" {
		t.Error("Expected to retrieve saved fact")
	}
}

func TestSearchFacts(t *testing.T) {
	store, err := NewStorageInMemory()
	if err != nil {
		t.Fatalf("NewStorageInMemory() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	// Save some facts
	facts := []models.Fact{
		{FactID: "fact_search_1", Key: "user_name", Value: "Doctor Biz", Confidence: 1.0, CreatedAt: time.Now()},
		{FactID: "fact_search_2", Key: "user_preference", Value: "TDD", Confidence: 0.9, CreatedAt: time.Now()},
	}
	_ = store.SaveFacts(facts)

	// Search for facts
	results, err := store.SearchFacts("user", 10)
	if err != nil {
		t.Fatalf("SearchFacts() error = %v", err)
	}

	if len(results) != 2 {
		t.Errorf("SearchFacts() returned %d results, want 2", len(results))
	}
}

func TestDeleteFactByID(t *testing.T) {
	store, err := NewStorageInMemory()
	if err != nil {
		t.Fatalf("NewStorageInMemory() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	fact := &models.Fact{
		FactID:     "fact_to_delete",
		Key:        "temp_key",
		Value:      "temp_value",
		Confidence: 1.0,
		CreatedAt:  time.Now(),
	}
	_ = store.SaveFact(fact)

	// Delete by ID
	err = store.DeleteFactByID("fact_to_delete")
	if err != nil {
		t.Fatalf("DeleteFactByID() error = %v", err)
	}

	// Verify deleted
	retrieved, err := store.GetFactByKey("temp_key")
	if err != nil {
		t.Fatalf("GetFactByKey() error = %v", err)
	}
	if retrieved != nil {
		t.Error("Fact should be deleted")
	}
}

func TestGetVectorStorage(t *testing.T) {
	store, err := NewStorageInMemory()
	if err != nil {
		t.Fatalf("NewStorageInMemory() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	vectorStore := store.GetVectorStorage()
	if vectorStore == nil {
		t.Error("GetVectorStorage() returned nil")
	}
}

func TestSetOpenAIClient(t *testing.T) {
	store, err := NewStorageInMemory()
	if err != nil {
		t.Fatalf("NewStorageInMemory() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	// Test setting client (can't fully test without mock)
	// Just verify the method doesn't panic
	store.SetOpenAIClient(nil)
}

func TestSetChunkEngine(t *testing.T) {
	store, err := NewStorageInMemory()
	if err != nil {
		t.Fatalf("NewStorageInMemory() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	// Test setting engine (can't fully test without mock)
	// Just verify the method doesn't panic
	store.SetChunkEngine(nil)
}

func TestStoreTurn_NoTopics(t *testing.T) {
	store, err := NewStorageInMemory()
	if err != nil {
		t.Fatalf("NewStorageInMemory() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	// Turn with no topics should get "General Discussion" label
	turn := &models.Turn{
		TurnID:      "turn_no_topics",
		Timestamp:   time.Now(),
		UserMessage: "Hello",
		Keywords:    []string{},
		Topics:      []string{},
	}

	blockID, err := store.StoreTurn(turn)
	if err != nil {
		t.Fatalf("StoreTurn() error = %v", err)
	}

	block, err := store.GetBridgeBlock(blockID)
	if err != nil {
		t.Fatalf("GetBridgeBlock() error = %v", err)
	}

	if block.TopicLabel != "General Discussion" {
		t.Errorf("TopicLabel = %q, want 'General Discussion'", block.TopicLabel)
	}
}

func TestAppendTurnToBlock_MergesKeywords(t *testing.T) {
	store, err := NewStorageInMemory()
	if err != nil {
		t.Fatalf("NewStorageInMemory() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	turn1 := &models.Turn{
		TurnID:      "turn_kw_1",
		Timestamp:   time.Now(),
		UserMessage: "First",
		Keywords:    []string{"keyword1", "shared"},
		Topics:      []string{"topic"},
	}
	blockID, err := store.StoreTurn(turn1)
	if err != nil {
		t.Fatalf("StoreTurn() error = %v", err)
	}

	turn2 := &models.Turn{
		TurnID:      "turn_kw_2",
		Timestamp:   time.Now(),
		UserMessage: "Second",
		Keywords:    []string{"keyword2", "shared"}, // "shared" is duplicate
		Topics:      []string{"topic"},
	}
	err = store.AppendTurnToBlock(blockID, turn2)
	if err != nil {
		t.Fatalf("AppendTurnToBlock() error = %v", err)
	}

	block, err := store.GetBridgeBlock(blockID)
	if err != nil {
		t.Fatalf("GetBridgeBlock() error = %v", err)
	}

	// Should have 3 unique keywords: keyword1, shared, keyword2
	if len(block.Keywords) != 3 {
		t.Errorf("Keywords count = %d, want 3 (should merge without duplicates)", len(block.Keywords))
	}
}

func TestClose_NilDB(t *testing.T) {
	store := &Storage{db: nil}
	err := store.Close()
	if err != nil {
		t.Errorf("Close() with nil db should not error, got: %v", err)
	}
}

func TestInferTopicLabel(t *testing.T) {
	tests := []struct {
		name   string
		turn   *models.Turn
		expect string
	}{
		{
			name: "with topics",
			turn: &models.Turn{
				Topics: []string{"first_topic", "second_topic"},
			},
			expect: "first_topic",
		},
		{
			name: "empty topics",
			turn: &models.Turn{
				Topics: []string{},
			},
			expect: "General Discussion",
		},
		{
			name: "nil topics",
			turn: &models.Turn{
				Topics: nil,
			},
			expect: "General Discussion",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := inferTopicLabel(tt.turn)
			if result != tt.expect {
				t.Errorf("inferTopicLabel() = %q, want %q", result, tt.expect)
			}
		})
	}
}

func TestMatchesQuery(t *testing.T) {
	tests := []struct {
		name   string
		block  *models.BridgeBlock
		query  string
		expect bool
	}{
		{
			name: "matches keyword",
			block: &models.BridgeBlock{
				Keywords:   []string{"go", "programming"},
				TopicLabel: "development",
			},
			query:  "go",
			expect: true,
		},
		{
			name: "matches topic label",
			block: &models.BridgeBlock{
				Keywords:   []string{"test"},
				TopicLabel: "programming",
			},
			query:  "programming",
			expect: true,
		},
		{
			name: "case insensitive",
			block: &models.BridgeBlock{
				Keywords:   []string{"GO"},
				TopicLabel: "test",
			},
			query:  "go",
			expect: true,
		},
		{
			name: "no match",
			block: &models.BridgeBlock{
				Keywords:   []string{"python"},
				TopicLabel: "scripting",
			},
			query:  "java",
			expect: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchesQuery(tt.block, tt.query)
			if result != tt.expect {
				t.Errorf("matchesQuery() = %v, want %v", result, tt.expect)
			}
		})
	}
}

func TestNewStorageWithPath(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_storage.db")

	store, err := NewStorageWithPath(dbPath)
	if err != nil {
		t.Fatalf("NewStorageWithPath() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	// Verify we can use the storage
	_, err = store.GetUserProfile()
	if err != nil {
		t.Errorf("GetUserProfile() error = %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("Database file was not created")
	}
}

func TestGetFactsForBlock(t *testing.T) {
	store, err := NewStorageInMemory()
	if err != nil {
		t.Fatalf("NewStorageInMemory() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create a block
	turn := &models.Turn{
		TurnID:      "turn_fact_block",
		Timestamp:   time.Now(),
		UserMessage: "Test",
		Topics:      []string{"test"},
	}
	blockID, err := store.StoreTurn(turn)
	if err != nil {
		t.Fatalf("StoreTurn() error = %v", err)
	}

	// Save facts for this block
	facts := []models.Fact{
		{
			FactID:     "fact_block_1",
			Key:        "key1",
			Value:      "value1",
			BlockID:    blockID,
			TurnID:     "turn_fact_block",
			Confidence: 1.0,
			CreatedAt:  time.Now(),
		},
		{
			FactID:     "fact_block_2",
			Key:        "key2",
			Value:      "value2",
			BlockID:    blockID,
			TurnID:     "turn_fact_block",
			Confidence: 0.9,
			CreatedAt:  time.Now(),
		},
	}
	_ = store.SaveFacts(facts)

	// Get facts for block
	retrieved, err := store.GetFactsForBlock(blockID)
	if err != nil {
		t.Fatalf("GetFactsForBlock() error = %v", err)
	}

	if len(retrieved) != 2 {
		t.Errorf("GetFactsForBlock() returned %d facts, want 2", len(retrieved))
	}
}

func TestSearchMemory_EmptyResult(t *testing.T) {
	store, err := NewStorageInMemory()
	if err != nil {
		t.Fatalf("NewStorageInMemory() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	// Search with no data
	results, err := store.SearchMemory("nonexistent", 10)
	if err != nil {
		t.Fatalf("SearchMemory() error = %v", err)
	}

	if len(results) != 0 {
		t.Errorf("Expected 0 results for empty database, got %d", len(results))
	}
}

func TestSearchMemory_MaxResults(t *testing.T) {
	store, err := NewStorageInMemory()
	if err != nil {
		t.Fatalf("NewStorageInMemory() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create multiple blocks with matching keyword
	for i := 0; i < 10; i++ {
		turn := &models.Turn{
			TurnID:      fmt.Sprintf("turn_max_%d", i),
			Timestamp:   time.Now(),
			UserMessage: "Go programming topic",
			Keywords:    []string{"go", "programming"},
			Topics:      []string{"programming"},
		}
		_, _ = store.StoreTurn(turn)
	}

	// Search with limit
	results, err := store.SearchMemory("go", 3)
	if err != nil {
		t.Fatalf("SearchMemory() error = %v", err)
	}

	if len(results) > 3 {
		t.Errorf("SearchMemory() returned %d results, want max 3", len(results))
	}
}

func TestKeywordSearch(t *testing.T) {
	store, err := NewStorageInMemory()
	if err != nil {
		t.Fatalf("NewStorageInMemory() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create a block
	turn := &models.Turn{
		TurnID:      "turn_kw_search",
		Timestamp:   time.Now(),
		UserMessage: "Testing keywords",
		Keywords:    []string{"testing", "keywords"},
		Topics:      []string{"test"},
	}
	_, err = store.StoreTurn(turn)
	if err != nil {
		t.Fatalf("StoreTurn() error = %v", err)
	}

	// Test keyword search (internal function)
	results := store.keywordSearch("testing", 10)
	if len(results) == 0 {
		t.Error("keywordSearch() should find block with matching keyword")
	}
}

func TestAppendTurnToBlock_NonexistentBlock(t *testing.T) {
	store, err := NewStorageInMemory()
	if err != nil {
		t.Fatalf("NewStorageInMemory() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	turn := &models.Turn{
		TurnID:      "turn_nonexistent",
		UserMessage: "Test",
		Timestamp:   time.Now(),
	}

	err = store.AppendTurnToBlock("nonexistent_block", turn)
	if err == nil {
		t.Error("AppendTurnToBlock() should error for nonexistent block")
	}
}

func TestStoreTurn_AutoRepair(t *testing.T) {
	store, err := NewStorageInMemory()
	if err != nil {
		t.Fatalf("NewStorageInMemory() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	// Manually create multiple active blocks (corruption)
	block1 := &models.BridgeBlock{
		BlockID:   "block_repair_1",
		DayID:     "2026-02-01",
		Status:    models.StatusActive,
		CreatedAt: time.Now().Add(-2 * time.Hour),
		UpdatedAt: time.Now().Add(-2 * time.Hour),
	}
	block2 := &models.BridgeBlock{
		BlockID:   "block_repair_2",
		DayID:     "2026-02-01",
		Status:    models.StatusActive,
		CreatedAt: time.Now().Add(-1 * time.Hour),
		UpdatedAt: time.Now().Add(-1 * time.Hour),
	}
	_ = store.blocks.Save(block1)
	_ = store.blocks.Save(block2)

	// Verify corruption
	active, _ := store.GetActiveBridgeBlocks()
	if len(active) != 2 {
		t.Fatalf("Expected 2 active blocks (corruption), got %d", len(active))
	}

	// Store a new turn - should trigger auto-repair
	turn := &models.Turn{
		TurnID:      "turn_trigger_repair",
		Timestamp:   time.Now(),
		UserMessage: "This should trigger repair",
		Topics:      []string{"test"},
	}
	_, err = store.StoreTurn(turn)
	if err != nil {
		t.Fatalf("StoreTurn() error = %v", err)
	}

	// Verify only one active block now
	active, _ = store.GetActiveBridgeBlocks()
	if len(active) != 1 {
		t.Errorf("After auto-repair, expected 1 active block, got %d", len(active))
	}
}

func TestNewStorageWithPath_NestedDir(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "deep", "nested", "storage.db")

	store, err := NewStorageWithPath(dbPath)
	if err != nil {
		t.Fatalf("NewStorageWithPath() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("Database file in nested dir was not created")
	}
}
