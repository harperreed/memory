// ABOUTME: Tests for bridge block storage operations
// ABOUTME: Verifies CRUD operations for bridge blocks and turns
package sqlite

import (
	"testing"
	"time"

	"github.com/harper/remember-standalone/internal/models"
)

func TestBlockCRUD(t *testing.T) {
	db, err := OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory() error = %v", err)
	}
	defer func() { _ = db.Close() }()

	store := NewBlockStore(db)

	// Test create block
	block := &models.BridgeBlock{
		BlockID:    "block_20260131_abc12345",
		DayID:      "2026-01-31",
		TopicLabel: "Testing SQLite",
		Keywords:   []string{"testing", "sqlite", "go"},
		Status:     models.StatusActive,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		TurnCount:  0,
	}

	err = store.Save(block)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Test get block
	retrieved, err := store.Get(block.BlockID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if retrieved == nil {
		t.Fatal("Get() returned nil")
	}

	if retrieved.BlockID != block.BlockID {
		t.Errorf("BlockID = %v, want %v", retrieved.BlockID, block.BlockID)
	}
	if retrieved.TopicLabel != "Testing SQLite" {
		t.Errorf("TopicLabel = %v, want Testing SQLite", retrieved.TopicLabel)
	}
	if retrieved.Status != models.StatusActive {
		t.Errorf("Status = %v, want ACTIVE", retrieved.Status)
	}
	if len(retrieved.Keywords) != 3 {
		t.Errorf("Keywords length = %v, want 3", len(retrieved.Keywords))
	}

	// Test update block
	retrieved.Status = models.StatusPaused
	retrieved.Summary = "Updated summary"
	err = store.Save(retrieved)
	if err != nil {
		t.Fatalf("Save() update error = %v", err)
	}

	updated, err := store.Get(block.BlockID)
	if err != nil {
		t.Fatalf("Get() after update error = %v", err)
	}
	if updated.Status != models.StatusPaused {
		t.Errorf("Status = %v, want PAUSED", updated.Status)
	}
	if updated.Summary != "Updated summary" {
		t.Errorf("Summary = %v, want Updated summary", updated.Summary)
	}

	// Test delete block
	err = store.Delete(block.BlockID)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	deleted, err := store.Get(block.BlockID)
	if err != nil {
		t.Fatalf("Get() after delete error = %v", err)
	}
	if deleted != nil {
		t.Error("Get() should return nil after delete")
	}
}

func TestBlocksByStatus(t *testing.T) {
	db, err := OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory() error = %v", err)
	}
	defer func() { _ = db.Close() }()

	store := NewBlockStore(db)

	// Create blocks with different statuses
	blocks := []*models.BridgeBlock{
		{BlockID: "block_1", DayID: "2026-01-31", TopicLabel: "Active 1", Status: models.StatusActive, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{BlockID: "block_2", DayID: "2026-01-31", TopicLabel: "Paused 1", Status: models.StatusPaused, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{BlockID: "block_3", DayID: "2026-01-31", TopicLabel: "Active 2", Status: models.StatusActive, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{BlockID: "block_4", DayID: "2026-01-31", TopicLabel: "Archived", Status: models.StatusArchived, CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}

	for _, b := range blocks {
		if err := store.Save(b); err != nil {
			t.Fatalf("Save() error = %v", err)
		}
	}

	// Test GetByStatus - Active
	active, err := store.GetByStatus(models.StatusActive)
	if err != nil {
		t.Fatalf("GetByStatus(Active) error = %v", err)
	}
	if len(active) != 2 {
		t.Errorf("Active blocks count = %v, want 2", len(active))
	}

	// Test GetByStatus - Paused
	paused, err := store.GetByStatus(models.StatusPaused)
	if err != nil {
		t.Fatalf("GetByStatus(Paused) error = %v", err)
	}
	if len(paused) != 1 {
		t.Errorf("Paused blocks count = %v, want 1", len(paused))
	}

	// Test ListAll
	all, err := store.ListAll()
	if err != nil {
		t.Fatalf("ListAll() error = %v", err)
	}
	if len(all) != 4 {
		t.Errorf("All blocks count = %v, want 4", len(all))
	}
}

func TestBlockWithTurns(t *testing.T) {
	db, err := OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory() error = %v", err)
	}
	defer func() { _ = db.Close() }()

	blockStore := NewBlockStore(db)
	turnStore := NewTurnStore(db)

	// Create a block
	block := &models.BridgeBlock{
		BlockID:    "block_with_turns",
		DayID:      "2026-01-31",
		TopicLabel: "Block with turns",
		Status:     models.StatusActive,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	if err := blockStore.Save(block); err != nil {
		t.Fatalf("Save block error = %v", err)
	}

	// Add turns
	turn1 := &models.Turn{
		TurnID:      "turn_1",
		UserMessage: "Hello",
		AIResponse:  "Hi there!",
		Keywords:    []string{"greeting"},
		Topics:      []string{"casual"},
		Timestamp:   time.Now(),
	}
	if err := turnStore.Save(block.BlockID, turn1); err != nil {
		t.Fatalf("Save turn1 error = %v", err)
	}

	turn2 := &models.Turn{
		TurnID:      "turn_2",
		UserMessage: "How are you?",
		AIResponse:  "I'm doing well!",
		Keywords:    []string{"question"},
		Topics:      []string{"casual"},
		Timestamp:   time.Now(),
	}
	if err := turnStore.Save(block.BlockID, turn2); err != nil {
		t.Fatalf("Save turn2 error = %v", err)
	}

	// Get block with turns
	retrieved, err := blockStore.GetWithTurns(block.BlockID)
	if err != nil {
		t.Fatalf("GetWithTurns() error = %v", err)
	}

	if len(retrieved.Turns) != 2 {
		t.Errorf("Turns count = %v, want 2", len(retrieved.Turns))
	}
}

func TestTurnCascadeDelete(t *testing.T) {
	db, err := OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory() error = %v", err)
	}
	defer func() { _ = db.Close() }()

	blockStore := NewBlockStore(db)
	turnStore := NewTurnStore(db)

	// Create block with turn
	block := &models.BridgeBlock{
		BlockID:    "block_cascade",
		DayID:      "2026-01-31",
		TopicLabel: "Cascade test",
		Status:     models.StatusActive,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	if err := blockStore.Save(block); err != nil {
		t.Fatalf("Save block error = %v", err)
	}

	turn := &models.Turn{
		TurnID:      "turn_cascade",
		UserMessage: "Test message",
		AIResponse:  "Test response",
		Timestamp:   time.Now(),
	}
	if err := turnStore.Save(block.BlockID, turn); err != nil {
		t.Fatalf("Save turn error = %v", err)
	}

	// Verify turn exists
	turns, err := turnStore.GetByBlock(block.BlockID)
	if err != nil {
		t.Fatalf("GetByBlock() error = %v", err)
	}
	if len(turns) != 1 {
		t.Fatalf("Expected 1 turn, got %d", len(turns))
	}

	// Delete block
	if err := blockStore.Delete(block.BlockID); err != nil {
		t.Fatalf("Delete block error = %v", err)
	}

	// Verify turn was cascade deleted
	turns, err = turnStore.GetByBlock(block.BlockID)
	if err != nil {
		t.Fatalf("GetByBlock() after delete error = %v", err)
	}
	if len(turns) != 0 {
		t.Errorf("Expected 0 turns after cascade delete, got %d", len(turns))
	}
}

func TestUpdateBlockStatus(t *testing.T) {
	db, err := OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory() error = %v", err)
	}
	defer func() { _ = db.Close() }()

	store := NewBlockStore(db)

	block := &models.BridgeBlock{
		BlockID:    "block_status",
		DayID:      "2026-01-31",
		TopicLabel: "Status test",
		Status:     models.StatusActive,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	if err := store.Save(block); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Update status
	err = store.UpdateStatus(block.BlockID, models.StatusPaused)
	if err != nil {
		t.Fatalf("UpdateStatus() error = %v", err)
	}

	retrieved, err := store.Get(block.BlockID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if retrieved.Status != models.StatusPaused {
		t.Errorf("Status = %v, want PAUSED", retrieved.Status)
	}
}
