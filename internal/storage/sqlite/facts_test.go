// ABOUTME: Tests for fact storage operations
// ABOUTME: Verifies CRUD operations and queries for facts
package sqlite

import (
	"testing"
	"time"

	"github.com/harper/remember-standalone/internal/models"
)

func TestFactCRUD(t *testing.T) {
	db, err := OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory() error = %v", err)
	}
	defer func() { _ = db.Close() }()

	// Create referenced block first
	blockStore := NewBlockStore(db)
	block := &models.BridgeBlock{
		BlockID:   "block_1",
		DayID:     "2026-01-31",
		Status:    models.StatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := blockStore.Save(block); err != nil {
		t.Fatalf("Save block error = %v", err)
	}

	store := NewFactStore(db)

	// Test save fact
	fact := &models.Fact{
		FactID:     "fact_abc123",
		BlockID:    "block_1",
		TurnID:     "turn_1",
		Key:        "user_name",
		Value:      "Harper",
		Confidence: 1.0,
		CreatedAt:  time.Now(),
	}

	err = store.Save(fact)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Test get by ID
	retrieved, err := store.GetByID(fact.FactID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if retrieved == nil {
		t.Fatal("GetByID() returned nil")
	}

	if retrieved.Key != "user_name" {
		t.Errorf("Key = %v, want user_name", retrieved.Key)
	}
	if retrieved.Value != "Harper" {
		t.Errorf("Value = %v, want Harper", retrieved.Value)
	}
	if retrieved.Confidence != 1.0 {
		t.Errorf("Confidence = %v, want 1.0", retrieved.Confidence)
	}

	// Test get by key
	byKey, err := store.GetByKey("user_name")
	if err != nil {
		t.Fatalf("GetByKey() error = %v", err)
	}
	if byKey == nil {
		t.Fatal("GetByKey() returned nil")
	}
	if byKey.Value != "Harper" {
		t.Errorf("Value = %v, want Harper", byKey.Value)
	}

	// Test delete by ID
	err = store.DeleteByID(fact.FactID)
	if err != nil {
		t.Fatalf("DeleteByID() error = %v", err)
	}

	deleted, err := store.GetByID(fact.FactID)
	if err != nil {
		t.Fatalf("GetByID() after delete error = %v", err)
	}
	if deleted != nil {
		t.Error("GetByID() should return nil after delete")
	}
}

func TestFactByKey(t *testing.T) {
	db, err := OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory() error = %v", err)
	}
	defer func() { _ = db.Close() }()

	store := NewFactStore(db)

	// Create multiple facts with the same key
	facts := []*models.Fact{
		{FactID: "fact_1", Key: "favorite_color", Value: "blue", Confidence: 0.8, CreatedAt: time.Now().Add(-2 * time.Hour)},
		{FactID: "fact_2", Key: "favorite_color", Value: "green", Confidence: 0.9, CreatedAt: time.Now().Add(-1 * time.Hour)},
		{FactID: "fact_3", Key: "favorite_color", Value: "purple", Confidence: 0.95, CreatedAt: time.Now()},
	}

	for _, f := range facts {
		if err := store.Save(f); err != nil {
			t.Fatalf("Save() error = %v", err)
		}
	}

	// GetByKey should return the most recent
	latest, err := store.GetByKey("favorite_color")
	if err != nil {
		t.Fatalf("GetByKey() error = %v", err)
	}
	if latest.Value != "purple" {
		t.Errorf("Value = %v, want purple (most recent)", latest.Value)
	}

	// DeleteByKey should delete all
	count, err := store.DeleteByKey("favorite_color")
	if err != nil {
		t.Fatalf("DeleteByKey() error = %v", err)
	}
	if count != 3 {
		t.Errorf("Deleted count = %v, want 3", count)
	}

	// Verify all deleted
	remaining, err := store.GetByKey("favorite_color")
	if err != nil {
		t.Fatalf("GetByKey() after delete error = %v", err)
	}
	if remaining != nil {
		t.Error("GetByKey() should return nil after DeleteByKey()")
	}
}

func TestFactsByBlock(t *testing.T) {
	db, err := OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory() error = %v", err)
	}
	defer func() { _ = db.Close() }()

	// Create a block first
	blockStore := NewBlockStore(db)
	block := &models.BridgeBlock{
		BlockID:   "block_facts",
		DayID:     "2026-01-31",
		Status:    models.StatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := blockStore.Save(block); err != nil {
		t.Fatalf("Save block error = %v", err)
	}

	factStore := NewFactStore(db)

	// Create facts for the block
	facts := []*models.Fact{
		{FactID: "fact_a", BlockID: "block_facts", Key: "key1", Value: "value1", Confidence: 1.0, CreatedAt: time.Now()},
		{FactID: "fact_b", BlockID: "block_facts", Key: "key2", Value: "value2", Confidence: 0.9, CreatedAt: time.Now()},
		{FactID: "fact_c", BlockID: "", Key: "key3", Value: "value3", Confidence: 0.8, CreatedAt: time.Now()}, // No block reference
	}

	for _, f := range facts {
		if err := factStore.Save(f); err != nil {
			t.Fatalf("Save() error = %v", err)
		}
	}

	// Get facts for block
	blockFacts, err := factStore.GetByBlock("block_facts")
	if err != nil {
		t.Fatalf("GetByBlock() error = %v", err)
	}
	if len(blockFacts) != 2 {
		t.Errorf("GetByBlock() count = %v, want 2", len(blockFacts))
	}
}

func TestFactSearch(t *testing.T) {
	db, err := OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory() error = %v", err)
	}
	defer func() { _ = db.Close() }()

	store := NewFactStore(db)

	// Create facts
	facts := []*models.Fact{
		{FactID: "f1", Key: "programming_language", Value: "Go", Confidence: 1.0, CreatedAt: time.Now()},
		{FactID: "f2", Key: "user_location", Value: "San Francisco", Confidence: 0.9, CreatedAt: time.Now()},
		{FactID: "f3", Key: "favorite_language", Value: "Golang", Confidence: 0.8, CreatedAt: time.Now()},
	}

	for _, f := range facts {
		if err := store.Save(f); err != nil {
			t.Fatalf("Save() error = %v", err)
		}
	}

	// Search by key substring
	results, err := store.Search("language", 10)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(results) != 2 {
		t.Errorf("Search('language') count = %v, want 2", len(results))
	}

	// Search by value substring
	results, err = store.Search("Go", 10)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(results) != 2 {
		t.Errorf("Search('Go') count = %v, want 2", len(results))
	}
}

func TestFactCascadeOnBlockDelete(t *testing.T) {
	db, err := OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory() error = %v", err)
	}
	defer func() { _ = db.Close() }()

	blockStore := NewBlockStore(db)
	factStore := NewFactStore(db)

	// Create block
	block := &models.BridgeBlock{
		BlockID:   "block_cascade",
		DayID:     "2026-01-31",
		Status:    models.StatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := blockStore.Save(block); err != nil {
		t.Fatalf("Save block error = %v", err)
	}

	// Create fact for block
	fact := &models.Fact{
		FactID:     "fact_cascade",
		BlockID:    "block_cascade",
		Key:        "test_key",
		Value:      "test_value",
		Confidence: 1.0,
		CreatedAt:  time.Now(),
	}
	if err := factStore.Save(fact); err != nil {
		t.Fatalf("Save fact error = %v", err)
	}

	// Delete block
	if err := blockStore.Delete(block.BlockID); err != nil {
		t.Fatalf("Delete block error = %v", err)
	}

	// Fact should still exist but with NULL block_id (ON DELETE SET NULL)
	retrieved, err := factStore.GetByID(fact.FactID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if retrieved == nil {
		t.Error("Fact should still exist after block deletion (SET NULL)")
	} else if retrieved.BlockID != "" {
		t.Errorf("BlockID should be empty after block deletion, got %v", retrieved.BlockID)
	}
}
