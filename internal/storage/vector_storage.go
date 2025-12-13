// ABOUTME: Vector storage with file-based JSON storage and cosine similarity search
// ABOUTME: Stores embeddings in daily JSON files at ~/.local/share/memory/embeddings/
package storage

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/harper/remember-standalone/internal/models"
)

// Expected embedding dimension for OpenAI text-embedding-3-small
const ExpectedEmbeddingDimension = 1536

// SkipDimensionValidation can be set to true in tests to allow non-1536D vectors
// This is useful for unit tests that use smaller vectors for readability
var SkipDimensionValidation = false

// VectorStorage manages embedding storage and similarity search
type VectorStorage struct {
	basePath string
}

// NewVectorStorage creates a new VectorStorage instance
func NewVectorStorage(basePath string) (*VectorStorage, error) {
	embeddingsDir := filepath.Join(basePath, "memory", "embeddings")
	if err := os.MkdirAll(embeddingsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create embeddings directory: %w", err)
	}

	return &VectorStorage{
		basePath: basePath,
	}, nil
}

// SaveEmbedding saves an embedding vector to disk
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

	// Determine file path based on current date
	today := time.Now().Format("2006-01-02")
	filePath := vs.getEmbeddingFilePath(today)

	// Load existing embeddings
	embeddings, err := vs.loadEmbeddingsFromFile(filePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to load existing embeddings: %w", err)
	}

	// Append new embedding
	embeddings = append(embeddings, embedding)

	// Save back to disk
	data, err := json.MarshalIndent(embeddings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal embeddings: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write embeddings file: %w", err)
	}

	return nil
}

// SearchSimilar performs cosine similarity search across all stored embeddings
func (vs *VectorStorage) SearchSimilar(queryVector []float64, maxResults int) ([]models.VectorSearchResult, error) {
	var allResults []models.VectorSearchResult

	// Search embeddings from last 30 days
	for i := 0; i < 30; i++ {
		day := time.Now().AddDate(0, 0, -i).Format("2006-01-02")
		filePath := vs.getEmbeddingFilePath(day)

		embeddings, err := vs.loadEmbeddingsFromFile(filePath)
		if err != nil {
			if os.IsNotExist(err) {
				continue // Skip days with no embeddings
			}
			return nil, fmt.Errorf("failed to load embeddings from %s: %w", filePath, err)
		}

		// Calculate cosine similarity for each embedding
		for _, emb := range embeddings {
			similarity := cosineSimilarity(queryVector, emb.Vector)
			allResults = append(allResults, models.VectorSearchResult{
				ChunkID:         emb.ChunkID,
				TurnID:          emb.TurnID,
				BlockID:         emb.BlockID,
				SimilarityScore: similarity,
			})
		}
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

// getEmbeddingFilePath returns the file path for a given date
func (vs *VectorStorage) getEmbeddingFilePath(date string) string {
	return filepath.Join(vs.basePath, "memory", "embeddings", date+".json")
}

// loadEmbeddingsFromFile loads embeddings from a JSON file
func (vs *VectorStorage) loadEmbeddingsFromFile(filePath string) ([]models.Embedding, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var embeddings []models.Embedding
	if err := json.Unmarshal(data, &embeddings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal embeddings: %w", err)
	}

	return embeddings, nil
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
