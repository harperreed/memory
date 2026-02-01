// ABOUTME: Tests for main storage wrapper functionality
// ABOUTME: Verifies storage factory functions and helper utilities

package storage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/harper/remember-standalone/internal/models"
)

func TestNewStorageInMemory(t *testing.T) {
	store, err := NewStorageInMemory()
	if err != nil {
		t.Fatalf("NewStorageInMemory() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	// Should be usable
	profile, err := store.GetUserProfile()
	if err != nil {
		t.Fatalf("GetUserProfile() error = %v", err)
	}
	if profile != nil {
		t.Error("Expected nil profile initially")
	}
}

func TestNewStorageWithPath(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_memory.db")

	store, err := NewStorageWithPath(dbPath)
	if err != nil {
		t.Fatalf("NewStorageWithPath() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	// Database file should be created
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("Database file was not created")
	}
}

func TestNewStorageWithPath_NestedDir(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "nested", "deep", "memory.db")

	store, err := NewStorageWithPath(dbPath)
	if err != nil {
		t.Fatalf("NewStorageWithPath() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	// Database file should be created (with parent dirs)
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("Database file was not created")
	}
}

func TestGetTurnsFromBlock(t *testing.T) {
	block := &models.BridgeBlock{
		BlockID: "block_test",
		Turns: []models.Turn{
			{TurnID: "turn_1", UserMessage: "First"},
			{TurnID: "turn_2", UserMessage: "Second"},
		},
	}

	turns := GetTurnsFromBlock(block)

	if len(turns) != 2 {
		t.Errorf("GetTurnsFromBlock() returned %d turns, want 2", len(turns))
	}

	if turns[0].TurnID != "turn_1" {
		t.Errorf("First turn ID = %q, want turn_1", turns[0].TurnID)
	}
}

func TestGetTurnsFromBlock_EmptyBlock(t *testing.T) {
	block := &models.BridgeBlock{
		BlockID: "block_empty",
		Turns:   []models.Turn{},
	}

	turns := GetTurnsFromBlock(block)

	if len(turns) != 0 {
		t.Errorf("GetTurnsFromBlock() returned %d turns, want 0", len(turns))
	}
}

func TestGetTurnsFromBlock_NilTurns(t *testing.T) {
	block := &models.BridgeBlock{
		BlockID: "block_nil",
		Turns:   nil,
	}

	turns := GetTurnsFromBlock(block)

	if len(turns) != 0 {
		t.Errorf("GetTurnsFromBlock() returned %d turns, want 0", len(turns))
	}
}

func TestDefaultDBPath(t *testing.T) {
	path := DefaultDBPath()

	if path == "" {
		t.Error("DefaultDBPath() returned empty string")
	}

	// Should end with memory.db
	if filepath.Base(path) != "memory.db" {
		t.Errorf("DefaultDBPath() = %q, should end with memory.db", path)
	}
}

func TestExpectedEmbeddingDimension(t *testing.T) {
	// OpenAI text-embedding-3-small uses 1536 dimensions
	if ExpectedEmbeddingDimension != 1536 {
		t.Errorf("ExpectedEmbeddingDimension = %d, want 1536", ExpectedEmbeddingDimension)
	}
}

func TestSkipDimensionValidation_Default(t *testing.T) {
	// Default should be false
	if SkipDimensionValidation {
		t.Error("SkipDimensionValidation default should be false")
	}
}

func TestSaveEmbedding_WithValidation(t *testing.T) {
	store, err := NewStorageInMemory()
	if err != nil {
		t.Fatalf("NewStorageInMemory() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	// First create a block and turn to satisfy foreign key constraints
	turn := &models.Turn{
		TurnID:      "turn_001",
		UserMessage: "Test message",
	}
	blockID, err := store.StoreTurn(turn)
	if err != nil {
		t.Fatalf("StoreTurn() error = %v", err)
	}

	vectorStore := store.GetVectorStorage()

	// Create a 1536-dimension vector
	vector := make([]float64, 1536)
	for i := range vector {
		vector[i] = float64(i) / 1536.0
	}

	// Should succeed with correct dimension
	err = SaveEmbedding(vectorStore, "chunk_001", "turn_001", blockID, vector)
	if err != nil {
		t.Errorf("SaveEmbedding() with valid dimension error = %v", err)
	}
}

func TestSaveEmbedding_SkipValidation(t *testing.T) {
	store, err := NewStorageInMemory()
	if err != nil {
		t.Fatalf("NewStorageInMemory() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	// First create a block and turn to satisfy foreign key constraints
	turn := &models.Turn{
		TurnID:      "turn_002",
		UserMessage: "Test message",
	}
	blockID, err := store.StoreTurn(turn)
	if err != nil {
		t.Fatalf("StoreTurn() error = %v", err)
	}

	vectorStore := store.GetVectorStorage()

	// Small test vector (not 1536)
	vector := []float64{0.1, 0.2, 0.3, 0.4, 0.5}

	// Enable skip validation
	originalSkip := SkipDimensionValidation
	SkipDimensionValidation = true
	defer func() { SkipDimensionValidation = originalSkip }()

	// Should succeed with skip validation
	err = SaveEmbedding(vectorStore, "chunk_002", "turn_002", blockID, vector)
	if err != nil {
		t.Errorf("SaveEmbedding() with skip validation error = %v", err)
	}
}

func TestSearchSimilarVectors(t *testing.T) {
	store, err := NewStorageInMemory()
	if err != nil {
		t.Fatalf("NewStorageInMemory() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	// First create a block and turn to satisfy foreign key constraints
	turn := &models.Turn{
		TurnID:      "turn_search",
		UserMessage: "Test message for search",
	}
	blockID, err := store.StoreTurn(turn)
	if err != nil {
		t.Fatalf("StoreTurn() error = %v", err)
	}

	vectorStore := store.GetVectorStorage()

	// Enable skip validation for testing with small vectors
	originalSkip := SkipDimensionValidation
	SkipDimensionValidation = true
	defer func() { SkipDimensionValidation = originalSkip }()

	// Save some embeddings
	vectors := [][]float64{
		{1.0, 0.0, 0.0},
		{0.9, 0.1, 0.0},
		{0.0, 1.0, 0.0},
	}

	for i, vec := range vectors {
		chunkID := "chunk_" + string(rune('a'+i))
		err := SaveEmbedding(vectorStore, chunkID, "turn_search", blockID, vec)
		if err != nil {
			t.Fatalf("SaveEmbedding() error = %v", err)
		}
	}

	// Search with query similar to first vector
	queryVector := []float64{1.0, 0.0, 0.0}
	results, err := SearchSimilarVectors(vectorStore, queryVector, 3)
	if err != nil {
		t.Fatalf("SearchSimilarVectors() error = %v", err)
	}

	if len(results) == 0 {
		t.Error("SearchSimilarVectors() returned no results")
	}

	// First result should be most similar
	if len(results) > 0 && results[0].SimilarityScore < 0.9 {
		t.Errorf("First result similarity = %v, expected > 0.9", results[0].SimilarityScore)
	}
}

func TestCosineSimilarity_Wrapper(t *testing.T) {
	// Test the wrapper function
	tests := []struct {
		name     string
		a        []float64
		b        []float64
		expected float64
		delta    float64
	}{
		{
			name:     "identical vectors",
			a:        []float64{1.0, 0.0, 0.0},
			b:        []float64{1.0, 0.0, 0.0},
			expected: 1.0,
			delta:    0.001,
		},
		{
			name:     "orthogonal vectors",
			a:        []float64{1.0, 0.0, 0.0},
			b:        []float64{0.0, 1.0, 0.0},
			expected: 0.0,
			delta:    0.001,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CosineSimilarity(tt.a, tt.b)
			if absFloat(result-tt.expected) > tt.delta {
				t.Errorf("CosineSimilarity() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// absFloat helper renamed to avoid redeclaration with vector_storage_test.go
func absFloat(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
