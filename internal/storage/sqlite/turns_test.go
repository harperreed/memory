// ABOUTME: Tests for turn storage operations
// ABOUTME: Verifies CRUD operations for conversation turns

package sqlite

import (
	"testing"
	"time"

	"github.com/harper/remember-standalone/internal/models"
)

func TestNewTurnStore(t *testing.T) {
	db, err := OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory() error = %v", err)
	}
	defer func() { _ = db.Close() }()

	store := NewTurnStore(db)
	if store == nil {
		t.Error("NewTurnStore() returned nil")
	}
}

func TestTurnStore_SaveAndGetByBlock(t *testing.T) {
	db, err := OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory() error = %v", err)
	}
	defer func() { _ = db.Close() }()

	// Create block first
	blockStore := NewBlockStore(db)
	block := &models.BridgeBlock{
		BlockID:   "block_turn_test",
		DayID:     "2026-02-01",
		Status:    models.StatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err = blockStore.Save(block)
	if err != nil {
		t.Fatalf("Save block error = %v", err)
	}

	turnStore := NewTurnStore(db)

	// Save turn
	turn := &models.Turn{
		TurnID:      "turn_001",
		UserMessage: "Hello",
		AIResponse:  "Hi there!",
		Keywords:    []string{"greeting", "hello"},
		Topics:      []string{"casual"},
		Timestamp:   time.Now(),
	}

	err = turnStore.Save(block.BlockID, turn)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Retrieve turns
	turns, err := turnStore.GetByBlock(block.BlockID)
	if err != nil {
		t.Fatalf("GetByBlock() error = %v", err)
	}

	if len(turns) != 1 {
		t.Fatalf("GetByBlock() returned %d turns, want 1", len(turns))
	}

	if turns[0].TurnID != "turn_001" {
		t.Errorf("TurnID = %q, want turn_001", turns[0].TurnID)
	}
	if turns[0].UserMessage != "Hello" {
		t.Errorf("UserMessage = %q, want Hello", turns[0].UserMessage)
	}
	if len(turns[0].Keywords) != 2 {
		t.Errorf("Keywords count = %d, want 2", len(turns[0].Keywords))
	}
	if len(turns[0].Topics) != 1 {
		t.Errorf("Topics count = %d, want 1", len(turns[0].Topics))
	}
}

func TestTurnStore_Delete(t *testing.T) {
	db, err := OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory() error = %v", err)
	}
	defer func() { _ = db.Close() }()

	// Create block
	blockStore := NewBlockStore(db)
	block := &models.BridgeBlock{
		BlockID:   "block_del_turn",
		DayID:     "2026-02-01",
		Status:    models.StatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	_ = blockStore.Save(block)

	turnStore := NewTurnStore(db)

	// Save turn
	turn := &models.Turn{
		TurnID:      "turn_to_delete",
		UserMessage: "Delete me",
		Timestamp:   time.Now(),
	}
	_ = turnStore.Save(block.BlockID, turn)

	// Delete turn
	err = turnStore.Delete(turn.TurnID)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify deleted
	turns, err := turnStore.GetByBlock(block.BlockID)
	if err != nil {
		t.Fatalf("GetByBlock() error = %v", err)
	}
	if len(turns) != 0 {
		t.Errorf("Expected 0 turns after delete, got %d", len(turns))
	}
}

func TestTurnStore_SaveUpdate(t *testing.T) {
	db, err := OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory() error = %v", err)
	}
	defer func() { _ = db.Close() }()

	// Create block
	blockStore := NewBlockStore(db)
	block := &models.BridgeBlock{
		BlockID:   "block_update_turn",
		DayID:     "2026-02-01",
		Status:    models.StatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	_ = blockStore.Save(block)

	turnStore := NewTurnStore(db)

	// Save initial turn
	turn := &models.Turn{
		TurnID:      "turn_update",
		UserMessage: "Original message",
		AIResponse:  "Original response",
		Timestamp:   time.Now(),
	}
	_ = turnStore.Save(block.BlockID, turn)

	// Update turn
	turn.UserMessage = "Updated message"
	turn.AIResponse = "Updated response"
	err = turnStore.Save(block.BlockID, turn)
	if err != nil {
		t.Fatalf("Save() update error = %v", err)
	}

	// Verify update
	turns, _ := turnStore.GetByBlock(block.BlockID)
	if len(turns) != 1 {
		t.Fatalf("Expected 1 turn after update, got %d", len(turns))
	}
	if turns[0].UserMessage != "Updated message" {
		t.Errorf("UserMessage = %q, want 'Updated message'", turns[0].UserMessage)
	}
}

func TestTurnStore_EmptyKeywordsAndTopics(t *testing.T) {
	db, err := OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory() error = %v", err)
	}
	defer func() { _ = db.Close() }()

	// Create block
	blockStore := NewBlockStore(db)
	block := &models.BridgeBlock{
		BlockID:   "block_empty",
		DayID:     "2026-02-01",
		Status:    models.StatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	_ = blockStore.Save(block)

	turnStore := NewTurnStore(db)

	// Save turn with nil keywords and topics
	turn := &models.Turn{
		TurnID:      "turn_empty",
		UserMessage: "Message",
		Keywords:    nil,
		Topics:      nil,
		Timestamp:   time.Now(),
	}
	err = turnStore.Save(block.BlockID, turn)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Retrieve
	turns, _ := turnStore.GetByBlock(block.BlockID)
	if len(turns) != 1 {
		t.Fatalf("Expected 1 turn, got %d", len(turns))
	}

	// Check that we got expected lengths (empty)
	if len(turns[0].Keywords) != 0 {
		t.Errorf("Keywords length = %d, want 0", len(turns[0].Keywords))
	}
	if len(turns[0].Topics) != 0 {
		t.Errorf("Topics length = %d, want 0", len(turns[0].Topics))
	}
}

func TestTurnStore_MultipleTurnsOrdering(t *testing.T) {
	db, err := OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory() error = %v", err)
	}
	defer func() { _ = db.Close() }()

	// Create block
	blockStore := NewBlockStore(db)
	block := &models.BridgeBlock{
		BlockID:   "block_order",
		DayID:     "2026-02-01",
		Status:    models.StatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	_ = blockStore.Save(block)

	turnStore := NewTurnStore(db)

	// Save multiple turns with specific timestamps
	now := time.Now()
	turn1 := &models.Turn{
		TurnID:      "turn_first",
		UserMessage: "First",
		Timestamp:   now.Add(-2 * time.Minute),
	}
	turn2 := &models.Turn{
		TurnID:      "turn_second",
		UserMessage: "Second",
		Timestamp:   now.Add(-1 * time.Minute),
	}
	turn3 := &models.Turn{
		TurnID:      "turn_third",
		UserMessage: "Third",
		Timestamp:   now,
	}

	// Save out of order
	_ = turnStore.Save(block.BlockID, turn3)
	_ = turnStore.Save(block.BlockID, turn1)
	_ = turnStore.Save(block.BlockID, turn2)

	// Retrieve - should be ordered by created_at ASC
	turns, _ := turnStore.GetByBlock(block.BlockID)
	if len(turns) != 3 {
		t.Fatalf("Expected 3 turns, got %d", len(turns))
	}

	if turns[0].TurnID != "turn_first" {
		t.Errorf("First turn = %q, want turn_first", turns[0].TurnID)
	}
	if turns[1].TurnID != "turn_second" {
		t.Errorf("Second turn = %q, want turn_second", turns[1].TurnID)
	}
	if turns[2].TurnID != "turn_third" {
		t.Errorf("Third turn = %q, want turn_third", turns[2].TurnID)
	}
}
