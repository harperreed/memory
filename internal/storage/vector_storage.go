// ABOUTME: Vector storage compatibility layer for SQLite backend
// ABOUTME: Provides same interface as old Charm-based VectorStorage
package storage

import (
	"github.com/harper/remember-standalone/internal/models"
	"github.com/harper/remember-standalone/internal/storage/sqlite"
)

// SaveEmbedding saves an embedding vector using the EmbeddingStore
// This is a helper function for backward compatibility
func SaveEmbedding(store *VectorStorage, chunkID, turnID, blockID string, vector []float64) error {
	if SkipDimensionValidation {
		return store.SaveWithDimension(chunkID, turnID, blockID, vector, len(vector))
	}
	return store.Save(chunkID, turnID, blockID, vector)
}

// SearchSimilarVectors provides backward-compatible similarity search
func SearchSimilarVectors(store *VectorStorage, queryVector []float64, maxResults int) ([]models.VectorSearchResult, error) {
	return store.SearchSimilar(queryVector, maxResults)
}

// CosineSimilarity is exposed for testing
func CosineSimilarity(a, b []float64) float64 {
	return sqlite.CosineSimilarity(a, b)
}
