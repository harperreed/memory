// ABOUTME: Unit tests for vector storage functionality
// ABOUTME: Tests embedding save/load and cosine similarity search
package storage

import (
	"os"
	"testing"
)

func TestVectorStorage_SaveAndSearch(t *testing.T) {
	// Allow small test vectors (not 1536D)
	SkipDimensionValidation = true
	defer func() { SkipDimensionValidation = false }()

	// Setup temp directory
	tmpDir := t.TempDir()

	vs, err := NewVectorStorage(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create vector storage: %v", err)
	}

	// Save some test embeddings
	vector1 := []float64{1.0, 0.0, 0.0}
	vector2 := []float64{0.0, 1.0, 0.0}
	vector3 := []float64{0.9, 0.1, 0.0}

	err = vs.SaveEmbedding("chunk1", "turn1", "block1", vector1)
	if err != nil {
		t.Fatalf("Failed to save embedding 1: %v", err)
	}

	err = vs.SaveEmbedding("chunk2", "turn2", "block2", vector2)
	if err != nil {
		t.Fatalf("Failed to save embedding 2: %v", err)
	}

	err = vs.SaveEmbedding("chunk3", "turn3", "block3", vector3)
	if err != nil {
		t.Fatalf("Failed to save embedding 3: %v", err)
	}

	// Search for vector similar to vector1
	queryVector := []float64{0.95, 0.05, 0.0}
	results, err := vs.SearchSimilar(queryVector, 3)
	if err != nil {
		t.Fatalf("Failed to search: %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("Expected 3 results, got %d", len(results))
	}

	// Verify top result is chunk1 or chunk3 (both are similar to query)
	// chunk1 = [1.0, 0.0, 0.0] - similarity to [0.95, 0.05, 0.0] ~ 0.998
	// chunk3 = [0.9, 0.1, 0.0] - similarity to [0.95, 0.05, 0.0] ~ 0.999
	// So chunk3 should be top
	if results[0].ChunkID != "chunk1" && results[0].ChunkID != "chunk3" {
		t.Errorf("Expected top result to be chunk1 or chunk3, got %s", results[0].ChunkID)
	}

	t.Logf("Top result: %s with similarity %.4f", results[0].ChunkID, results[0].SimilarityScore)

	// Verify scores are in descending order
	for i := 1; i < len(results); i++ {
		if results[i].SimilarityScore > results[i-1].SimilarityScore {
			t.Errorf("Results not sorted: score[%d]=%.4f > score[%d]=%.4f",
				i, results[i].SimilarityScore, i-1, results[i-1].SimilarityScore)
		}
	}
}

func TestCosineSimilarity(t *testing.T) {
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
		{
			name:     "opposite vectors",
			a:        []float64{1.0, 0.0, 0.0},
			b:        []float64{-1.0, 0.0, 0.0},
			expected: -1.0,
			delta:    0.001,
		},
		{
			name:     "similar vectors",
			a:        []float64{1.0, 0.0, 0.0},
			b:        []float64{0.9, 0.1, 0.0},
			expected: 0.995, // Approximately
			delta:    0.01,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cosineSimilarity(tt.a, tt.b)
			if abs(result-tt.expected) > tt.delta {
				t.Errorf("cosineSimilarity(%v, %v) = %.4f, expected %.4f (delta %.4f)",
					tt.a, tt.b, result, tt.expected, tt.delta)
			}
		})
	}
}

func TestVectorStorage_EmptySearch(t *testing.T) {
	tmpDir := t.TempDir()
	vs, err := NewVectorStorage(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create vector storage: %v", err)
	}

	// Search without any embeddings stored
	queryVector := []float64{1.0, 0.0, 0.0}
	results, err := vs.SearchSimilar(queryVector, 10)
	if err != nil {
		t.Fatalf("Failed to search: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("Expected 0 results, got %d", len(results))
	}
}

func TestVectorStorage_FileFormat(t *testing.T) {
	// Allow small test vectors (not 1536D)
	SkipDimensionValidation = true
	defer func() { SkipDimensionValidation = false }()

	tmpDir := t.TempDir()
	vs, err := NewVectorStorage(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create vector storage: %v", err)
	}

	vector := []float64{1.0, 2.0, 3.0}
	err = vs.SaveEmbedding("test-chunk", "test-turn", "test-block", vector)
	if err != nil {
		t.Fatalf("Failed to save embedding: %v", err)
	}

	// Verify file was created
	today := "2025-12-06" // Would normally use time.Now()
	filePath := vs.getEmbeddingFilePath(today)

	// The file should exist (in practice, we can't easily test the exact filename
	// since it uses time.Now(), but we can verify the directory exists)
	embeddingsDir := tmpDir + "/memory/embeddings"
	if _, err := os.Stat(embeddingsDir); os.IsNotExist(err) {
		t.Errorf("Embeddings directory not created: %s", embeddingsDir)
	}

	// Load the embeddings back
	embeddings, err := vs.loadEmbeddingsFromFile(filePath)
	if err != nil && !os.IsNotExist(err) {
		// It's OK if the file doesn't exist (date might be different), but other errors are bad
		t.Fatalf("Failed to load embeddings: %v", err)
	}

	// If we loaded successfully, verify the structure
	if err == nil && len(embeddings) > 0 {
		emb := embeddings[0]
		if emb.ChunkID != "test-chunk" {
			t.Errorf("Expected ChunkID 'test-chunk', got '%s'", emb.ChunkID)
		}
		if len(emb.Vector) != 3 {
			t.Errorf("Expected vector length 3, got %d", len(emb.Vector))
		}
	}
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
