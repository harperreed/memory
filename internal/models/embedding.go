// ABOUTME: Embedding models for vector storage and semantic search
// ABOUTME: Defines Embedding and VectorSearchResult structures
package models

import (
	"errors"
	"fmt"
	"time"
)

// Embedding represents a stored embedding vector for a text chunk
type Embedding struct {
	ChunkID   string    `json:"chunk_id"`
	TurnID    string    `json:"turn_id"`
	BlockID   string    `json:"block_id"`
	Vector    []float64 `json:"vector"`
	CreatedAt time.Time `json:"created_at"`
}

// VectorSearchResult represents a search result with similarity score
type VectorSearchResult struct {
	ChunkID         string  `json:"chunk_id"`
	TurnID          string  `json:"turn_id"`
	BlockID         string  `json:"block_id"`
	SimilarityScore float64 `json:"similarity_score"`
}

// ValidateDimension checks if the embedding vector has the expected dimension
func (e *Embedding) ValidateDimension(expectedDim int) error {
	if len(e.Vector) == 0 {
		return errors.New("embedding vector cannot be empty")
	}
	if len(e.Vector) != expectedDim {
		return fmt.Errorf("embedding dimension mismatch: expected %d, got %d", expectedDim, len(e.Vector))
	}
	return nil
}
