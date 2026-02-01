// ABOUTME: Embedding storage operations for SQLite
// ABOUTME: Implements vector storage as BLOB and cosine similarity search
package sqlite

import (
	"database/sql"
	"encoding/binary"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/harper/remember-standalone/internal/models"
)

// EmbeddingStore handles embedding persistence
type EmbeddingStore struct {
	db *DB
}

// NewEmbeddingStore creates a new EmbeddingStore
func NewEmbeddingStore(db *DB) *EmbeddingStore {
	return &EmbeddingStore{db: db}
}

// ExpectedDimension is the expected vector dimension for OpenAI embeddings
const ExpectedDimension = 1536

// Save saves an embedding vector (validates 1536 dimension)
func (s *EmbeddingStore) Save(chunkID, turnID, blockID string, vector []float64) error {
	if len(vector) != ExpectedDimension {
		return fmt.Errorf("invalid embedding dimension: expected %d, got %d", ExpectedDimension, len(vector))
	}
	return s.saveVector(chunkID, turnID, blockID, vector)
}

// SaveWithDimension saves an embedding vector with custom dimension (for testing)
func (s *EmbeddingStore) SaveWithDimension(chunkID, turnID, blockID string, vector []float64, expectedDim int) error {
	if len(vector) != expectedDim {
		return fmt.Errorf("invalid embedding dimension: expected %d, got %d", expectedDim, len(vector))
	}
	return s.saveVector(chunkID, turnID, blockID, vector)
}

// saveVector saves a vector to the database
func (s *EmbeddingStore) saveVector(chunkID, turnID, blockID string, vector []float64) error {
	blob := vectorToBlob(vector)
	embID := fmt.Sprintf("emb_%s", chunkID)

	_, err := s.db.Exec(`
		INSERT INTO embeddings (id, chunk_id, turn_id, block_id, vector, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			vector = excluded.vector,
			turn_id = excluded.turn_id,
			block_id = excluded.block_id
	`, embID, chunkID, nullString(turnID), nullString(blockID), blob, time.Now())

	return err
}

// GetByChunkID retrieves an embedding by chunk ID
func (s *EmbeddingStore) GetByChunkID(chunkID string) (*models.Embedding, error) {
	var (
		emb     models.Embedding
		turnID  sql.NullString
		blockID sql.NullString
		blob    []byte
	)

	err := s.db.QueryRow(`
		SELECT chunk_id, turn_id, block_id, vector, created_at
		FROM embeddings
		WHERE chunk_id = ?
	`, chunkID).Scan(&emb.ChunkID, &turnID, &blockID, &blob, &emb.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if turnID.Valid {
		emb.TurnID = turnID.String
	}
	if blockID.Valid {
		emb.BlockID = blockID.String
	}
	emb.Vector = blobToVector(blob)

	return &emb, nil
}

// GetByBlock retrieves all embeddings for a block
func (s *EmbeddingStore) GetByBlock(blockID string) ([]models.Embedding, error) {
	rows, err := s.db.Query(`
		SELECT chunk_id, turn_id, block_id, vector, created_at
		FROM embeddings
		WHERE block_id = ?
		ORDER BY created_at ASC
	`, blockID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	return s.scanEmbeddings(rows)
}

// SearchSimilar performs cosine similarity search
func (s *EmbeddingStore) SearchSimilar(queryVector []float64, maxResults int) ([]models.VectorSearchResult, error) {
	rows, err := s.db.Query(`
		SELECT chunk_id, turn_id, block_id, vector
		FROM embeddings
	`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var results []models.VectorSearchResult

	for rows.Next() {
		var (
			chunkID string
			turnID  sql.NullString
			blockID sql.NullString
			blob    []byte
		)

		if err := rows.Scan(&chunkID, &turnID, &blockID, &blob); err != nil {
			return nil, err
		}

		vector := blobToVector(blob)
		similarity := CosineSimilarity(queryVector, vector)

		result := models.VectorSearchResult{
			ChunkID:         chunkID,
			SimilarityScore: similarity,
		}
		if turnID.Valid {
			result.TurnID = turnID.String
		}
		if blockID.Valid {
			result.BlockID = blockID.String
		}

		results = append(results, result)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Sort by similarity descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].SimilarityScore > results[j].SimilarityScore
	})

	// Limit results
	if len(results) > maxResults {
		results = results[:maxResults]
	}

	return results, nil
}

// Delete removes an embedding by chunk ID
func (s *EmbeddingStore) Delete(chunkID string) error {
	_, err := s.db.Exec("DELETE FROM embeddings WHERE chunk_id = ?", chunkID)
	return err
}

// scanEmbeddings scans rows into embeddings
func (s *EmbeddingStore) scanEmbeddings(rows *sql.Rows) ([]models.Embedding, error) {
	var embeddings []models.Embedding

	for rows.Next() {
		var (
			emb     models.Embedding
			turnID  sql.NullString
			blockID sql.NullString
			blob    []byte
		)

		if err := rows.Scan(&emb.ChunkID, &turnID, &blockID, &blob, &emb.CreatedAt); err != nil {
			return nil, err
		}

		if turnID.Valid {
			emb.TurnID = turnID.String
		}
		if blockID.Valid {
			emb.BlockID = blockID.String
		}
		emb.Vector = blobToVector(blob)

		embeddings = append(embeddings, emb)
	}

	return embeddings, rows.Err()
}

// vectorToBlob converts a float64 slice to binary blob
func vectorToBlob(vector []float64) []byte {
	blob := make([]byte, len(vector)*8)
	for i, v := range vector {
		binary.LittleEndian.PutUint64(blob[i*8:], math.Float64bits(v))
	}
	return blob
}

// blobToVector converts a binary blob to float64 slice
func blobToVector(blob []byte) []float64 {
	count := len(blob) / 8
	vector := make([]float64, count)
	for i := 0; i < count; i++ {
		bits := binary.LittleEndian.Uint64(blob[i*8:])
		vector[i] = math.Float64frombits(bits)
	}
	return vector
}

// CosineSimilarity calculates cosine similarity between two vectors
func CosineSimilarity(a, b []float64) float64 {
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
