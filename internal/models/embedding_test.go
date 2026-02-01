// ABOUTME: Tests for Embedding model and dimension validation
// ABOUTME: Verifies vector dimension checking for embedding consistency
package models

import (
	"testing"
	"time"
)

func TestEmbedding_ValidateDimension(t *testing.T) {
	tests := []struct {
		name        string
		embedding   Embedding
		expectedDim int
		wantErr     bool
		errContains string
	}{
		{
			name: "valid dimension match",
			embedding: Embedding{
				ChunkID: "chunk_001",
				Vector:  []float64{0.1, 0.2, 0.3, 0.4},
			},
			expectedDim: 4,
			wantErr:     false,
		},
		{
			name: "empty vector",
			embedding: Embedding{
				ChunkID: "chunk_002",
				Vector:  []float64{},
			},
			expectedDim: 4,
			wantErr:     true,
			errContains: "cannot be empty",
		},
		{
			name: "nil vector",
			embedding: Embedding{
				ChunkID: "chunk_003",
				Vector:  nil,
			},
			expectedDim: 4,
			wantErr:     true,
			errContains: "cannot be empty",
		},
		{
			name: "dimension mismatch - too short",
			embedding: Embedding{
				ChunkID: "chunk_004",
				Vector:  []float64{0.1, 0.2},
			},
			expectedDim: 4,
			wantErr:     true,
			errContains: "dimension mismatch",
		},
		{
			name: "dimension mismatch - too long",
			embedding: Embedding{
				ChunkID: "chunk_005",
				Vector:  []float64{0.1, 0.2, 0.3, 0.4, 0.5, 0.6},
			},
			expectedDim: 4,
			wantErr:     true,
			errContains: "dimension mismatch",
		},
		{
			name: "large dimension match (like text-embedding-ada-002)",
			embedding: Embedding{
				ChunkID: "chunk_006",
				Vector:  make([]float64, 1536),
			},
			expectedDim: 1536,
			wantErr:     false,
		},
		{
			name: "small-3 dimension match",
			embedding: Embedding{
				ChunkID: "chunk_007",
				Vector:  make([]float64, 512),
			},
			expectedDim: 512,
			wantErr:     false,
		},
		{
			name: "zero expected dimension with non-empty vector",
			embedding: Embedding{
				ChunkID: "chunk_008",
				Vector:  []float64{0.1},
			},
			expectedDim: 0,
			wantErr:     true,
			errContains: "dimension mismatch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.embedding.ValidateDimension(tt.expectedDim)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateDimension() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errContains != "" {
				if !containsSubstring(err.Error(), tt.errContains) {
					t.Errorf("ValidateDimension() error = %q, want to contain %q", err.Error(), tt.errContains)
				}
			}
		})
	}
}

func TestEmbedding_Fields(t *testing.T) {
	now := time.Now()
	embedding := Embedding{
		ChunkID:   "chunk_001",
		TurnID:    "turn_001",
		BlockID:   "block_001",
		Vector:    []float64{0.1, 0.2, 0.3},
		CreatedAt: now,
	}

	if embedding.ChunkID != "chunk_001" {
		t.Errorf("ChunkID = %q, want %q", embedding.ChunkID, "chunk_001")
	}
	if embedding.TurnID != "turn_001" {
		t.Errorf("TurnID = %q, want %q", embedding.TurnID, "turn_001")
	}
	if embedding.BlockID != "block_001" {
		t.Errorf("BlockID = %q, want %q", embedding.BlockID, "block_001")
	}
	if len(embedding.Vector) != 3 {
		t.Errorf("Vector length = %d, want 3", len(embedding.Vector))
	}
	if embedding.CreatedAt != now {
		t.Errorf("CreatedAt = %v, want %v", embedding.CreatedAt, now)
	}
}

func TestVectorSearchResult_Fields(t *testing.T) {
	result := VectorSearchResult{
		ChunkID:         "chunk_001",
		TurnID:          "turn_001",
		BlockID:         "block_001",
		SimilarityScore: 0.95,
	}

	if result.ChunkID != "chunk_001" {
		t.Errorf("ChunkID = %q, want %q", result.ChunkID, "chunk_001")
	}
	if result.TurnID != "turn_001" {
		t.Errorf("TurnID = %q, want %q", result.TurnID, "turn_001")
	}
	if result.BlockID != "block_001" {
		t.Errorf("BlockID = %q, want %q", result.BlockID, "block_001")
	}
	if result.SimilarityScore != 0.95 {
		t.Errorf("SimilarityScore = %v, want %v", result.SimilarityScore, 0.95)
	}
}

func TestEmbedding_ValidateDimension_RealisticSizes(t *testing.T) {
	// Test with realistic embedding dimensions from various models
	dims := []int{
		384,  // text-embedding-3-small
		512,  // MiniLM
		768,  // MPNet
		1024, // large models
		1536, // text-embedding-ada-002, text-embedding-3-large
		3072, // text-embedding-3-large max
	}

	for _, dim := range dims {
		t.Run("dimension_"+string(rune('0'+dim)), func(t *testing.T) {
			embedding := Embedding{
				ChunkID: "chunk_test",
				Vector:  make([]float64, dim),
			}

			err := embedding.ValidateDimension(dim)
			if err != nil {
				t.Errorf("ValidateDimension(%d) should succeed with matching vector, got error: %v", dim, err)
			}
		})
	}
}

// containsSubstring checks if s contains substr
func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
