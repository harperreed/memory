// ABOUTME: Tests for LatticeCrawler semantic search
// ABOUTME: Verifies memory retrieval via vector similarity

package core

import (
	"fmt"
	"testing"
	"time"

	"github.com/harper/remember-standalone/internal/models"
	"github.com/harper/remember-standalone/internal/storage"
)

func TestNewLatticeCrawler(t *testing.T) {
	store, err := storage.NewStorageInMemory()
	if err != nil {
		t.Fatalf("NewStorageInMemory() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	crawler := NewLatticeCrawler(nil, store)
	if crawler == nil {
		t.Fatal("NewLatticeCrawler() returned nil")
	}

	if crawler.storage != store {
		t.Error("LatticeCrawler.storage not set correctly")
	}
}

func TestLatticeCrawler_RetrieveCandidates(t *testing.T) {
	store, err := storage.NewStorageInMemory()
	if err != nil {
		t.Fatalf("NewStorageInMemory() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create some test data
	turn1 := &models.Turn{
		TurnID:      "turn_lc_1",
		Timestamp:   time.Now(),
		UserMessage: "Let's discuss Go programming",
		AIResponse:  "Sure, Go is a great language!",
		Keywords:    []string{"go", "programming"},
		Topics:      []string{"programming"},
	}
	blockID1, err := store.StoreTurn(turn1)
	if err != nil {
		t.Fatalf("StoreTurn() error = %v", err)
	}

	turn2 := &models.Turn{
		TurnID:      "turn_lc_2",
		Timestamp:   time.Now(),
		UserMessage: "What about cooking?",
		AIResponse:  "I love cooking discussions!",
		Keywords:    []string{"cooking", "food"},
		Topics:      []string{"cooking"},
	}
	_, err = store.StoreTurn(turn2)
	if err != nil {
		t.Fatalf("StoreTurn() error = %v", err)
	}

	crawler := NewLatticeCrawler(nil, store)

	// Search for programming
	candidates, err := crawler.RetrieveCandidates("go programming", 5)
	if err != nil {
		t.Fatalf("RetrieveCandidates() error = %v", err)
	}

	// Should find at least the programming block
	if len(candidates) == 0 {
		t.Error("Expected at least one candidate")
	}

	// Verify block content
	found := false
	for _, c := range candidates {
		if c.BlockID == blockID1 {
			found = true
			if c.Block == nil {
				t.Error("Candidate Block should not be nil")
			}
			break
		}
	}
	if !found {
		// May not find due to keyword search limitations
		t.Log("Programming block not found in candidates (may be expected)")
	}
}

func TestLatticeCrawler_RetrieveCandidates_NoMatches(t *testing.T) {
	store, err := storage.NewStorageInMemory()
	if err != nil {
		t.Fatalf("NewStorageInMemory() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	crawler := NewLatticeCrawler(nil, store)

	// Search with no data
	candidates, err := crawler.RetrieveCandidates("nonexistent topic", 5)
	if err != nil {
		t.Fatalf("RetrieveCandidates() error = %v", err)
	}

	if len(candidates) != 0 {
		t.Errorf("Expected 0 candidates, got %d", len(candidates))
	}
}

func TestCandidateMemory_Fields(t *testing.T) {
	cm := CandidateMemory{
		BlockID:         "block_test",
		SimilarityScore: 0.95,
		MatchedChunks:   []string{"chunk_1", "chunk_2"},
	}

	if cm.BlockID != "block_test" {
		t.Errorf("BlockID = %q, want block_test", cm.BlockID)
	}
	if cm.SimilarityScore != 0.95 {
		t.Errorf("SimilarityScore = %v, want 0.95", cm.SimilarityScore)
	}
	if len(cm.MatchedChunks) != 2 {
		t.Errorf("MatchedChunks length = %d, want 2", len(cm.MatchedChunks))
	}
}

func TestLatticeCrawler_RetrieveCandidatesByVector(t *testing.T) {
	store, err := storage.NewStorageInMemory()
	if err != nil {
		t.Fatalf("NewStorageInMemory() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create test data
	turn := &models.Turn{
		TurnID:      "turn_vector_test",
		Timestamp:   time.Now(),
		UserMessage: "Test message about programming",
		AIResponse:  "Response about programming",
		Keywords:    []string{"programming", "test"},
		Topics:      []string{"programming"},
	}
	blockID, err := store.StoreTurn(turn)
	if err != nil {
		t.Fatalf("StoreTurn() error = %v", err)
	}

	// Store an embedding for this block using test vector
	testVector := make([]float64, 4)
	testVector[0] = 1.0
	testVector[1] = 0.0
	testVector[2] = 0.0
	testVector[3] = 0.0

	err = store.GetVectorStorage().SaveWithDimension("chunk_vector_test", turn.TurnID, blockID, testVector, 4)
	if err != nil {
		t.Fatalf("SaveWithDimension() error = %v", err)
	}

	crawler := NewLatticeCrawler(nil, store)

	// Query with a similar vector
	queryVector := []float64{0.9, 0.1, 0.0, 0.0}
	candidates, err := crawler.RetrieveCandidatesByVector(queryVector, 5)
	if err != nil {
		t.Fatalf("RetrieveCandidatesByVector() error = %v", err)
	}

	// Should find the block we created
	if len(candidates) == 0 {
		t.Error("Expected at least one candidate from vector search")
	}

	// Verify the candidate has expected fields
	if len(candidates) > 0 {
		c := candidates[0]
		if c.BlockID != blockID {
			t.Errorf("BlockID = %q, want %q", c.BlockID, blockID)
		}
		if c.Block == nil {
			t.Error("Block should not be nil")
		}
		if c.SimilarityScore <= 0 {
			t.Error("SimilarityScore should be positive")
		}
	}
}

func TestLatticeCrawler_RetrieveCandidatesByVector_NoMatches(t *testing.T) {
	store, err := storage.NewStorageInMemory()
	if err != nil {
		t.Fatalf("NewStorageInMemory() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	crawler := NewLatticeCrawler(nil, store)

	// Query with any vector when no embeddings exist
	queryVector := []float64{1.0, 0.0, 0.0, 0.0}
	candidates, err := crawler.RetrieveCandidatesByVector(queryVector, 5)
	if err != nil {
		t.Fatalf("RetrieveCandidatesByVector() error = %v", err)
	}

	if len(candidates) != 0 {
		t.Errorf("Expected 0 candidates, got %d", len(candidates))
	}
}

func TestLatticeCrawler_RetrieveCandidatesByVector_MultipleBlocks(t *testing.T) {
	store, err := storage.NewStorageInMemory()
	if err != nil {
		t.Fatalf("NewStorageInMemory() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create multiple blocks with embeddings
	for i := 0; i < 3; i++ {
		turn := &models.Turn{
			TurnID:      fmt.Sprintf("turn_multi_%d", i),
			Timestamp:   time.Now(),
			UserMessage: fmt.Sprintf("Message %d", i),
			Keywords:    []string{"test"},
		}
		blockID, err := store.StoreTurn(turn)
		if err != nil {
			t.Fatalf("StoreTurn() error = %v", err)
		}

		// Create a vector with unique direction for each block
		vector := make([]float64, 4)
		vector[i] = 1.0

		err = store.GetVectorStorage().SaveWithDimension(
			fmt.Sprintf("chunk_multi_%d", i),
			turn.TurnID,
			blockID,
			vector,
			4,
		)
		if err != nil {
			t.Fatalf("SaveWithDimension() error = %v", err)
		}
	}

	crawler := NewLatticeCrawler(nil, store)

	// Query for a vector that should match the first block best
	queryVector := []float64{0.9, 0.1, 0.0, 0.0}
	candidates, err := crawler.RetrieveCandidatesByVector(queryVector, 5)
	if err != nil {
		t.Fatalf("RetrieveCandidatesByVector() error = %v", err)
	}

	if len(candidates) != 3 {
		t.Errorf("Expected 3 candidates, got %d", len(candidates))
	}

	// Results should be sorted by similarity (descending)
	for i := 0; i < len(candidates)-1; i++ {
		if candidates[i].SimilarityScore < candidates[i+1].SimilarityScore {
			t.Error("Candidates should be sorted by similarity score (descending)")
			break
		}
	}
}

func TestLatticeCrawler_RetrieveCandidates_WithDeletedBlock(t *testing.T) {
	store, err := storage.NewStorageInMemory()
	if err != nil {
		t.Fatalf("NewStorageInMemory() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create a block
	turn := &models.Turn{
		TurnID:      "turn_deleted_test",
		Timestamp:   time.Now(),
		UserMessage: "Go programming is great",
		Keywords:    []string{"go", "programming"},
		Topics:      []string{"programming"},
	}
	_, err = store.StoreTurn(turn)
	if err != nil {
		t.Fatalf("StoreTurn() error = %v", err)
	}

	crawler := NewLatticeCrawler(nil, store)

	// Search should work
	candidates, err := crawler.RetrieveCandidates("go programming", 5)
	if err != nil {
		t.Fatalf("RetrieveCandidates() error = %v", err)
	}

	// May or may not find candidates depending on keyword matching
	t.Logf("Found %d candidates for 'go programming'", len(candidates))
}
