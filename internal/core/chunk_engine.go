// ABOUTME: ChunkEngine splits incoming messages into hierarchical chunks for embedding
// ABOUTME: Implements turn → paragraph → sentence hierarchy
package core

import (
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/harper/remember-standalone/internal/models"
)

// ChunkEngine handles hierarchical text chunking
type ChunkEngine struct{}

// NewChunkEngine creates a new ChunkEngine instance
func NewChunkEngine() *ChunkEngine {
	return &ChunkEngine{}
}

// ChunkTurn splits text hierarchically into turn → paragraph → sentence chunks
func (ce *ChunkEngine) ChunkTurn(text string, turnID string) ([]models.Chunk, error) {
	if strings.TrimSpace(text) == "" {
		return nil, errors.New("cannot chunk empty text")
	}

	var chunks []models.Chunk

	// Create turn-level chunk
	turnChunk := models.Chunk{
		ChunkID:       generateChunkID(),
		ChunkType:     models.ChunkTypeTurn,
		Content:       text,
		ParentChunkID: "",
		TurnID:        turnID,
	}
	chunks = append(chunks, turnChunk)

	// Split into paragraphs (by double newline)
	paragraphs := splitParagraphs(text)

	for _, paraText := range paragraphs {
		if strings.TrimSpace(paraText) == "" {
			continue
		}

		// Create paragraph chunk
		paraChunk := models.Chunk{
			ChunkID:       generateChunkID(),
			ChunkType:     models.ChunkTypeParagraph,
			Content:       paraText,
			ParentChunkID: turnChunk.ChunkID,
			TurnID:        turnID,
		}
		chunks = append(chunks, paraChunk)

		// Split paragraph into sentences
		sentences := splitSentences(paraText)

		for _, sentText := range sentences {
			if strings.TrimSpace(sentText) == "" {
				continue
			}

			// Create sentence chunk
			sentChunk := models.Chunk{
				ChunkID:       generateChunkID(),
				ChunkType:     models.ChunkTypeSentence,
				Content:       sentText,
				ParentChunkID: paraChunk.ChunkID,
				TurnID:        turnID,
			}
			chunks = append(chunks, sentChunk)
		}
	}

	return chunks, nil
}

// splitParagraphs splits text by double newlines
func splitParagraphs(text string) []string {
	// Split by double newline (handles both \n\n and \r\n\r\n)
	paragraphs := strings.Split(text, "\n\n")

	// Also handle \r\n\r\n for Windows
	var result []string
	for _, para := range paragraphs {
		if strings.Contains(para, "\r\n\r\n") {
			subParas := strings.Split(para, "\r\n\r\n")
			result = append(result, subParas...)
		} else {
			result = append(result, para)
		}
	}

	return result
}

// splitSentences splits text by ". " (period + space)
func splitSentences(text string) []string {
	// Split by period followed by space
	sentences := strings.Split(text, ". ")

	var result []string
	for i, sent := range sentences {
		sent = strings.TrimSpace(sent)
		if sent == "" {
			continue
		}

		// Add back the period (except for the last sentence which might already have it)
		if i < len(sentences)-1 && !strings.HasSuffix(sent, ".") {
			sent = sent + "."
		}

		result = append(result, sent)
	}

	return result
}

// generateChunkID generates a unique chunk ID
func generateChunkID() string {
	return "chunk_" + uuid.New().String()
}
