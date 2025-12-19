// ABOUTME: Vector storage with Charm KV backend and cosine similarity search
// ABOUTME: Stores embeddings in Charm KV for cloud-synced vector storage
package storage

import (
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/harper/remember-standalone/internal/charm"
	"github.com/harper/remember-standalone/internal/models"
)

// Expected embedding dimension for OpenAI text-embedding-3-small
const ExpectedEmbeddingDimension = 1536

// SkipDimensionValidation can be set to true in tests to allow non-1536D vectors
var SkipDimensionValidation = false

// VectorStorage manages embedding storage and similarity search using Charm KV
type VectorStorage struct {
	charm *charm.Client
}

// NewVectorStorage creates a new VectorStorage instance with Charm backend
func NewVectorStorage(charmClient *charm.Client) (*VectorStorage, error) {
	return &VectorStorage{
		charm: charmClient,
	}, nil
}

// SaveEmbedding saves an embedding vector to Charm KV
func (vs *VectorStorage) SaveEmbedding(chunkID, turnID, blockID string, vector []float64) error {
	// Validate embedding dimension (skip in tests for smaller test vectors)
	if !SkipDimensionValidation && len(vector) != ExpectedEmbeddingDimension {
		return fmt.Errorf("invalid embedding dimension: expected %d, got %d", ExpectedEmbeddingDimension, len(vector))
	}

	embedding := models.Embedding{
		ChunkID:   chunkID,
		TurnID:    turnID,
		BlockID:   blockID,
		Vector:    vector,
		CreatedAt: time.Now(),
	}

	key := charm.EmbeddingKey(chunkID)
	return vs.charm.SetJSON(key, embedding)
}

// SearchSimilar performs cosine similarity search across all stored embeddings
func (vs *VectorStorage) SearchSimilar(queryVector []float64, maxResults int) ([]models.VectorSearchResult, error) {
	var allResults []models.VectorSearchResult

	// Get all embedding keys
	keys, err := vs.charm.ListKeys(charm.EmbeddingPrefix)
	if err != nil {
		return nil, fmt.Errorf("failed to list embedding keys: %w", err)
	}

	// Calculate cosine similarity for each embedding
	for _, key := range keys {
		var emb models.Embedding
		if err := vs.charm.GetJSON(key, &emb); err != nil {
			continue
		}

		similarity := cosineSimilarity(queryVector, emb.Vector)
		allResults = append(allResults, models.VectorSearchResult{
			ChunkID:         emb.ChunkID,
			TurnID:          emb.TurnID,
			BlockID:         emb.BlockID,
			SimilarityScore: similarity,
		})
	}

	// Sort by similarity score (descending)
	sort.Slice(allResults, func(i, j int) bool {
		return allResults[i].SimilarityScore > allResults[j].SimilarityScore
	})

	// Return top maxResults
	if len(allResults) > maxResults {
		allResults = allResults[:maxResults]
	}

	return allResults, nil
}

// cosineSimilarity calculates cosine similarity between two vectors
func cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) {
		return 0.0
	}

	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0.0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}
