// ABOUTME: Tests for Chunk model and ChunkType validation
// ABOUTME: Verifies chunk type validity across hierarchy levels
package models

import "testing"

func TestChunkType_IsValid(t *testing.T) {
	tests := []struct {
		name      string
		chunkType ChunkType
		want      bool
	}{
		{
			name:      "TURN is valid",
			chunkType: ChunkTypeTurn,
			want:      true,
		},
		{
			name:      "PARAGRAPH is valid",
			chunkType: ChunkTypeParagraph,
			want:      true,
		},
		{
			name:      "SENTENCE is valid",
			chunkType: ChunkTypeSentence,
			want:      true,
		},
		{
			name:      "empty string is invalid",
			chunkType: ChunkType(""),
			want:      false,
		},
		{
			name:      "arbitrary string is invalid",
			chunkType: ChunkType("WORD"),
			want:      false,
		},
		{
			name:      "lowercase turn is invalid",
			chunkType: ChunkType("turn"),
			want:      false,
		},
		{
			name:      "mixed case is invalid",
			chunkType: ChunkType("Turn"),
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.chunkType.IsValid()
			if got != tt.want {
				t.Errorf("IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChunkType_Constants(t *testing.T) {
	// Verify the string values of chunk type constants
	if ChunkTypeTurn != "TURN" {
		t.Errorf("ChunkTypeTurn = %q, want %q", ChunkTypeTurn, "TURN")
	}
	if ChunkTypeParagraph != "PARAGRAPH" {
		t.Errorf("ChunkTypeParagraph = %q, want %q", ChunkTypeParagraph, "PARAGRAPH")
	}
	if ChunkTypeSentence != "SENTENCE" {
		t.Errorf("ChunkTypeSentence = %q, want %q", ChunkTypeSentence, "SENTENCE")
	}
}

func TestChunk_Fields(t *testing.T) {
	chunk := Chunk{
		ChunkID:       "chunk_001",
		ChunkType:     ChunkTypeTurn,
		Content:       "This is test content",
		ParentChunkID: "parent_001",
		TurnID:        "turn_001",
	}

	if chunk.ChunkID != "chunk_001" {
		t.Errorf("ChunkID = %q, want %q", chunk.ChunkID, "chunk_001")
	}
	if chunk.ChunkType != ChunkTypeTurn {
		t.Errorf("ChunkType = %q, want %q", chunk.ChunkType, ChunkTypeTurn)
	}
	if chunk.Content != "This is test content" {
		t.Errorf("Content = %q, want %q", chunk.Content, "This is test content")
	}
	if chunk.ParentChunkID != "parent_001" {
		t.Errorf("ParentChunkID = %q, want %q", chunk.ParentChunkID, "parent_001")
	}
	if chunk.TurnID != "turn_001" {
		t.Errorf("TurnID = %q, want %q", chunk.TurnID, "turn_001")
	}
}

func TestChunk_Hierarchy(t *testing.T) {
	// Create a turn-level chunk
	turnChunk := Chunk{
		ChunkID:   "chunk_turn",
		ChunkType: ChunkTypeTurn,
		Content:   "Full turn content",
		TurnID:    "turn_001",
	}

	// Create paragraph chunks that belong to the turn
	para1 := Chunk{
		ChunkID:       "chunk_para_1",
		ChunkType:     ChunkTypeParagraph,
		Content:       "First paragraph",
		ParentChunkID: turnChunk.ChunkID,
		TurnID:        "turn_001",
	}

	para2 := Chunk{
		ChunkID:       "chunk_para_2",
		ChunkType:     ChunkTypeParagraph,
		Content:       "Second paragraph",
		ParentChunkID: turnChunk.ChunkID,
		TurnID:        "turn_001",
	}

	// Create sentence chunks that belong to paragraphs
	sent1 := Chunk{
		ChunkID:       "chunk_sent_1",
		ChunkType:     ChunkTypeSentence,
		Content:       "First sentence.",
		ParentChunkID: para1.ChunkID,
		TurnID:        "turn_001",
	}

	// Validate hierarchy relationships
	if para1.ParentChunkID != turnChunk.ChunkID {
		t.Error("Paragraph should reference turn as parent")
	}
	if para2.ParentChunkID != turnChunk.ChunkID {
		t.Error("Paragraph should reference turn as parent")
	}
	if sent1.ParentChunkID != para1.ChunkID {
		t.Error("Sentence should reference paragraph as parent")
	}

	// All chunks should share the same TurnID
	if para1.TurnID != turnChunk.TurnID {
		t.Error("All chunks should share the same TurnID")
	}
	if sent1.TurnID != turnChunk.TurnID {
		t.Error("All chunks should share the same TurnID")
	}
}

func TestChunk_EmptyParent(t *testing.T) {
	// Turn-level chunks typically have no parent
	turnChunk := Chunk{
		ChunkID:       "chunk_turn",
		ChunkType:     ChunkTypeTurn,
		Content:       "Turn content",
		ParentChunkID: "", // No parent for top-level chunks
		TurnID:        "turn_001",
	}

	if turnChunk.ParentChunkID != "" {
		t.Error("Top-level chunks should have empty ParentChunkID")
	}
}
