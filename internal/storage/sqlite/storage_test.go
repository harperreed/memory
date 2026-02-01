// ABOUTME: Tests for unified Storage wrapper
// ABOUTME: Verifies high-level storage operations match old interface
package sqlite

import (
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
