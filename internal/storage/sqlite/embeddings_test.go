// ABOUTME: Tests for embedding storage operations
// ABOUTME: Verifies vector storage and similarity search
package sqlite

import (
	"math"
	"testing"
	"time"

	"github.com/harper/remember-standalone/internal/models"
)

func TestEmbeddingCRUD(t *testing.T) {
	db, err := OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory() error = %v", err)
	}
	defer func() { _ = db.Close() }()

	// Create referenced block first
	blockStore := NewBlockStore(db)
	block := &models.BridgeBlock{
		BlockID:   "block_emb",
		DayID:     "2026-01-31",
		Status:    models.StatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := blockStore.Save(block); err != nil {
		t.Fatalf("Save block error = %v", err)
	}

	store := NewEmbeddingStore(db)

	// Test save embedding
	vector := make([]float64, 1536)
	for i := range vector {
		vector[i] = float64(i) / 1536.0
	}

	err = store.Save("chunk_1", "turn_1", "block_emb", vector)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Test get by chunk ID
	retrieved, err := store.GetByChunkID("chunk_1")
	if err != nil {
		t.Fatalf("GetByChunkID() error = %v", err)
	}
	if retrieved == nil {
		t.Fatal("GetByChunkID() returned nil")
	}

	if retrieved.ChunkID != "chunk_1" {
		t.Errorf("ChunkID = %v, want chunk_1", retrieved.ChunkID)
	}
	if retrieved.TurnID != "turn_1" {
		t.Errorf("TurnID = %v, want turn_1", retrieved.TurnID)
	}
	if retrieved.BlockID != "block_emb" {
		t.Errorf("BlockID = %v, want block_emb", retrieved.BlockID)
	}
	if len(retrieved.Vector) != 1536 {
		t.Errorf("Vector length = %v, want 1536", len(retrieved.Vector))
	}

	// Verify vector values
	for i, v := range retrieved.Vector {
		expected := float64(i) / 1536.0
		if math.Abs(v-expected) > 1e-10 {
			t.Errorf("Vector[%d] = %v, want %v", i, v, expected)
			break
		}
	}

	// Test delete
	err = store.Delete("chunk_1")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	deleted, err := store.GetByChunkID("chunk_1")
	if err != nil {
		t.Fatalf("GetByChunkID() after delete error = %v", err)
	}
	if deleted != nil {
		t.Error("GetByChunkID() should return nil after delete")
	}
}

func TestEmbeddingSearch(t *testing.T) {
	db, err := OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory() error = %v", err)
	}
	defer func() { _ = db.Close() }()

	// Create block
	blockStore := NewBlockStore(db)
	block := &models.BridgeBlock{
		BlockID:   "block_search",
		DayID:     "2026-01-31",
		Status:    models.StatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := blockStore.Save(block); err != nil {
		t.Fatalf("Save block error = %v", err)
	}

	store := NewEmbeddingStore(db)

	// Create test vectors with known similarity
	dim := 4 // Use small dimension for testing

	// Vector 1: [1, 0, 0, 0] - similar to query
	v1 := []float64{1, 0, 0, 0}
	// Vector 2: [0, 1, 0, 0] - orthogonal to query
	v2 := []float64{0, 1, 0, 0}
	// Vector 3: [0.9, 0.1, 0, 0] - most similar to query
	v3 := []float64{0.9, 0.1, 0, 0}

	if err := store.SaveWithDimension("chunk_1", "turn_1", "block_search", v1, dim); err != nil {
		t.Fatalf("Save v1 error = %v", err)
	}
	if err := store.SaveWithDimension("chunk_2", "turn_2", "block_search", v2, dim); err != nil {
		t.Fatalf("Save v2 error = %v", err)
	}
	if err := store.SaveWithDimension("chunk_3", "turn_3", "block_search", v3, dim); err != nil {
		t.Fatalf("Save v3 error = %v", err)
	}

	// Query vector: [1, 0, 0, 0]
	query := []float64{1, 0, 0, 0}

	results, err := store.SearchSimilar(query, 3)
	if err != nil {
		t.Fatalf("SearchSimilar() error = %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("Results count = %v, want 3", len(results))
	}

	// chunk_1 (identical) should have score 1.0
	// chunk_3 (0.9, 0.1, 0, 0) should be second
	// chunk_2 (orthogonal) should have score 0.0

	if results[0].ChunkID != "chunk_1" {
		t.Errorf("First result ChunkID = %v, want chunk_1 (most similar)", results[0].ChunkID)
	}
	if math.Abs(results[0].SimilarityScore-1.0) > 0.01 {
		t.Errorf("First result score = %v, want ~1.0", results[0].SimilarityScore)
	}

	if results[1].ChunkID != "chunk_3" {
		t.Errorf("Second result ChunkID = %v, want chunk_3", results[1].ChunkID)
	}

	if results[2].ChunkID != "chunk_2" {
		t.Errorf("Third result ChunkID = %v, want chunk_2 (orthogonal)", results[2].ChunkID)
	}
	if math.Abs(results[2].SimilarityScore-0.0) > 0.01 {
		t.Errorf("Third result score = %v, want ~0.0", results[2].SimilarityScore)
	}
}

func TestEmbeddingsByBlock(t *testing.T) {
	db, err := OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory() error = %v", err)
	}
	defer func() { _ = db.Close() }()

	blockStore := NewBlockStore(db)
	block1 := &models.BridgeBlock{BlockID: "block_1", DayID: "2026-01-31", Status: models.StatusActive, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	block2 := &models.BridgeBlock{BlockID: "block_2", DayID: "2026-01-31", Status: models.StatusActive, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := blockStore.Save(block1); err != nil {
		t.Fatalf("Save block1 error = %v", err)
	}
	if err := blockStore.Save(block2); err != nil {
		t.Fatalf("Save block2 error = %v", err)
	}

	store := NewEmbeddingStore(db)

	v := []float64{1, 0, 0, 0}
	if err := store.SaveWithDimension("chunk_a", "turn_a", "block_1", v, 4); err != nil {
		t.Fatalf("Save error = %v", err)
	}
	if err := store.SaveWithDimension("chunk_b", "turn_b", "block_1", v, 4); err != nil {
		t.Fatalf("Save error = %v", err)
	}
	if err := store.SaveWithDimension("chunk_c", "turn_c", "block_2", v, 4); err != nil {
		t.Fatalf("Save error = %v", err)
	}

	embeddings, err := store.GetByBlock("block_1")
	if err != nil {
		t.Fatalf("GetByBlock() error = %v", err)
	}
	if len(embeddings) != 2 {
		t.Errorf("GetByBlock() count = %v, want 2", len(embeddings))
	}
}

func TestEmbeddingCascadeOnBlockDelete(t *testing.T) {
	db, err := OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory() error = %v", err)
	}
	defer func() { _ = db.Close() }()

	blockStore := NewBlockStore(db)
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

	embStore := NewEmbeddingStore(db)
	v := []float64{1, 0, 0, 0}
	if err := embStore.SaveWithDimension("chunk_cascade", "turn_1", "block_cascade", v, 4); err != nil {
		t.Fatalf("Save embedding error = %v", err)
	}

	// Delete block
	if err := blockStore.Delete(block.BlockID); err != nil {
		t.Fatalf("Delete block error = %v", err)
	}

	// Embedding should be cascade deleted (ON DELETE CASCADE)
	emb, err := embStore.GetByChunkID("chunk_cascade")
	if err != nil {
		t.Fatalf("GetByChunkID() error = %v", err)
	}
	if emb != nil {
		t.Error("Embedding should be deleted after block deletion (CASCADE)")
	}
}
