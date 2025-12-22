// ABOUTME: Chunk represents a hierarchical text fragment for embedding
// ABOUTME: Supports turn → paragraph → sentence chunking hierarchy
package models

// ChunkType represents the level in the chunking hierarchy
type ChunkType string

const (
	ChunkTypeTurn      ChunkType = "TURN"
	ChunkTypeParagraph ChunkType = "PARAGRAPH"
	ChunkTypeSentence  ChunkType = "SENTENCE"
)

// IsValid checks if the ChunkType is a valid value
func (c ChunkType) IsValid() bool {
	return c == ChunkTypeTurn || c == ChunkTypeParagraph || c == ChunkTypeSentence
}

// Chunk represents a hierarchical piece of text for embedding
type Chunk struct {
	ChunkID       string    `json:"chunk_id"`
	ChunkType     ChunkType `json:"chunk_type"`
	Content       string    `json:"content"`
	ParentChunkID string    `json:"parent_chunk_id,omitempty"`
	TurnID        string    `json:"turn_id"`
}
