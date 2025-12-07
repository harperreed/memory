// ABOUTME: LatticeCrawler performs vector-based semantic search for memory retrieval
// ABOUTME: Wraps VectorStorage to provide candidate memory blocks based on query embeddings
package core

import (
	"fmt"
	"sort"

	"github.com/harper/remember-standalone/internal/llm"
	"github.com/harper/remember-standalone/internal/models"
	"github.com/harper/remember-standalone/internal/storage"
)

// LatticeCrawler retrieves candidate memories using vector similarity search
type LatticeCrawler struct {
	client  *llm.OpenAIClient
	storage *storage.Storage
}

// NewLatticeCrawler creates a new LatticeCrawler with the given OpenAI client and storage
func NewLatticeCrawler(client *llm.OpenAIClient, storage *storage.Storage) *LatticeCrawler {
	return &LatticeCrawler{
		client:  client,
		storage: storage,
	}
}

// CandidateMemory represents a raw candidate memory from vector search
type CandidateMemory struct {
	BlockID         string
	Block           *models.BridgeBlock
	SimilarityScore float64
	MatchedChunks   []string // Chunk IDs that matched
}

// RetrieveCandidates performs semantic search to find candidate memories
// Returns raw candidates sorted by relevance score (descending)
func (lc *LatticeCrawler) RetrieveCandidates(query string, maxResults int) ([]CandidateMemory, error) {
	// Use Storage's SearchMemory which does hybrid keyword + semantic search
	memoryResults, err := lc.storage.SearchMemory(query, maxResults)
	if err != nil {
		return nil, fmt.Errorf("memory search failed: %w", err)
	}

	// Convert MemorySearchResults to CandidateMemories
	candidates := make([]CandidateMemory, 0, len(memoryResults))

	for _, result := range memoryResults {
		// Retrieve full Bridge Block
		block, err := lc.storage.GetBridgeBlock(result.BlockID)
		if err != nil || block == nil {
			// Skip blocks that can't be retrieved (may have been deleted)
			continue
		}

		candidates = append(candidates, CandidateMemory{
			BlockID:         result.BlockID,
			Block:           block,
			SimilarityScore: result.RelevanceScore,
			MatchedChunks:   []string{}, // SearchMemory doesn't return chunk IDs
		})
	}

	return candidates, nil
}

// RetrieveCandidatesByVector performs vector search using a pre-computed embedding
// Useful when the query embedding is already available
func (lc *LatticeCrawler) RetrieveCandidatesByVector(queryVector []float64, maxResults int) ([]CandidateMemory, error) {
	// Perform vector similarity search directly
	vectorResults, err := lc.storage.GetVectorStorage().SearchSimilar(queryVector, maxResults)
	if err != nil {
		return nil, fmt.Errorf("vector search failed: %w", err)
	}

	// Build candidate memories from vector results
	blockScores := make(map[string]float64)
	blockChunks := make(map[string][]string)

	for _, result := range vectorResults {
		if score, exists := blockScores[result.BlockID]; !exists || result.SimilarityScore > score {
			blockScores[result.BlockID] = result.SimilarityScore
		}
		blockChunks[result.BlockID] = append(blockChunks[result.BlockID], result.ChunkID)
	}

	// Retrieve full Bridge Blocks
	candidates := make([]CandidateMemory, 0, len(blockScores))
	for blockID, score := range blockScores {
		block, err := lc.storage.GetBridgeBlock(blockID)
		if err != nil || block == nil {
			continue
		}

		candidates = append(candidates, CandidateMemory{
			BlockID:         blockID,
			Block:           block,
			SimilarityScore: score,
			MatchedChunks:   blockChunks[blockID],
		})
	}

	// Sort by score (descending) using Go's efficient sort
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].SimilarityScore > candidates[j].SimilarityScore
	})

	return candidates, nil
}
