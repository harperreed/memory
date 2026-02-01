// ABOUTME: Tests for ChunkEngine hierarchical text chunking
// ABOUTME: Verifies turn, paragraph, and sentence level chunking

package core

import (
	"strings"
	"testing"

	"github.com/harper/remember-standalone/internal/models"
)

func TestNewChunkEngine(t *testing.T) {
	ce := NewChunkEngine()
	if ce == nil {
		t.Error("NewChunkEngine() returned nil")
	}
}

func TestChunkTurn_EmptyText(t *testing.T) {
	ce := NewChunkEngine()

	tests := []struct {
		name string
		text string
	}{
		{"empty string", ""},
		{"whitespace only", "   "},
		{"tabs and newlines", "\t\n\r"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunks, err := ce.ChunkTurn(tt.text, "turn_test")
			if err == nil {
				t.Error("Expected error for empty text")
			}
			if chunks != nil {
				t.Errorf("Expected nil chunks, got %d", len(chunks))
			}
		})
	}
}

func TestChunkTurn_SingleSentence(t *testing.T) {
	ce := NewChunkEngine()

	text := "This is a simple sentence."
	chunks, err := ce.ChunkTurn(text, "turn_001")
	if err != nil {
		t.Fatalf("ChunkTurn() error = %v", err)
	}

	// Should have at least turn chunk
	if len(chunks) == 0 {
		t.Fatal("Expected at least one chunk")
	}

	// First chunk should be turn level
	if chunks[0].ChunkType != models.ChunkTypeTurn {
		t.Errorf("First chunk type = %v, want TURN", chunks[0].ChunkType)
	}

	// Turn chunk should contain full text
	if chunks[0].Content != text {
		t.Errorf("Turn chunk content = %q, want %q", chunks[0].Content, text)
	}

	// Verify turn ID is set
	for _, chunk := range chunks {
		if chunk.TurnID != "turn_001" {
			t.Errorf("Chunk TurnID = %q, want turn_001", chunk.TurnID)
		}
	}
}

func TestChunkTurn_MultipleSentences(t *testing.T) {
	ce := NewChunkEngine()

	text := "First sentence here. Second sentence follows. Third one too."
	chunks, err := ce.ChunkTurn(text, "turn_002")
	if err != nil {
		t.Fatalf("ChunkTurn() error = %v", err)
	}

	// Count chunk types
	var turnCount, paraCount, sentCount int
	for _, chunk := range chunks {
		switch chunk.ChunkType {
		case models.ChunkTypeTurn:
			turnCount++
		case models.ChunkTypeParagraph:
			paraCount++
		case models.ChunkTypeSentence:
			sentCount++
		}
	}

	if turnCount != 1 {
		t.Errorf("Turn chunks = %d, want 1", turnCount)
	}

	// Should have multiple sentences
	if sentCount < 2 {
		t.Errorf("Sentence chunks = %d, want at least 2", sentCount)
	}
}

func TestChunkTurn_MultipleParagraphs(t *testing.T) {
	ce := NewChunkEngine()

	text := "First paragraph with some text.\n\nSecond paragraph here.\n\nThird paragraph too."
	chunks, err := ce.ChunkTurn(text, "turn_003")
	if err != nil {
		t.Fatalf("ChunkTurn() error = %v", err)
	}

	// Count paragraph chunks
	var paraCount int
	for _, chunk := range chunks {
		if chunk.ChunkType == models.ChunkTypeParagraph {
			paraCount++
		}
	}

	if paraCount != 3 {
		t.Errorf("Paragraph chunks = %d, want 3", paraCount)
	}
}

func TestChunkTurn_ChunkHierarchy(t *testing.T) {
	ce := NewChunkEngine()

	text := "First sentence. Second sentence.\n\nAnother paragraph here."
	chunks, err := ce.ChunkTurn(text, "turn_004")
	if err != nil {
		t.Fatalf("ChunkTurn() error = %v", err)
	}

	// Build a map of chunk IDs
	chunkByID := make(map[string]models.Chunk)
	for _, chunk := range chunks {
		chunkByID[chunk.ChunkID] = chunk
	}

	// Verify hierarchy
	for _, chunk := range chunks {
		switch chunk.ChunkType {
		case models.ChunkTypeTurn:
			if chunk.ParentChunkID != "" {
				t.Error("Turn chunk should not have parent")
			}
		case models.ChunkTypeParagraph:
			if chunk.ParentChunkID == "" {
				t.Error("Paragraph chunk should have turn as parent")
			}
			parent, exists := chunkByID[chunk.ParentChunkID]
			if !exists {
				t.Error("Paragraph parent not found")
			} else if parent.ChunkType != models.ChunkTypeTurn {
				t.Error("Paragraph parent should be turn")
			}
		case models.ChunkTypeSentence:
			if chunk.ParentChunkID == "" {
				t.Error("Sentence chunk should have paragraph as parent")
			}
			parent, exists := chunkByID[chunk.ParentChunkID]
			if !exists {
				t.Error("Sentence parent not found")
			} else if parent.ChunkType != models.ChunkTypeParagraph {
				t.Error("Sentence parent should be paragraph")
			}
		}
	}
}

func TestChunkTurn_UniqueChunkIDs(t *testing.T) {
	ce := NewChunkEngine()

	text := "First. Second. Third.\n\nNew paragraph. More text."
	chunks, err := ce.ChunkTurn(text, "turn_005")
	if err != nil {
		t.Fatalf("ChunkTurn() error = %v", err)
	}

	seen := make(map[string]bool)
	for _, chunk := range chunks {
		if seen[chunk.ChunkID] {
			t.Errorf("Duplicate chunk ID: %s", chunk.ChunkID)
		}
		seen[chunk.ChunkID] = true

		// Verify chunk ID format
		if !strings.HasPrefix(chunk.ChunkID, "chunk_") {
			t.Errorf("Chunk ID should start with 'chunk_': %s", chunk.ChunkID)
		}
	}
}

func TestSplitParagraphs(t *testing.T) {
	tests := []struct {
		name  string
		text  string
		count int
	}{
		{
			name:  "single paragraph",
			text:  "Just one paragraph here.",
			count: 1,
		},
		{
			name:  "two paragraphs unix",
			text:  "First paragraph.\n\nSecond paragraph.",
			count: 2,
		},
		{
			name:  "three paragraphs",
			text:  "One.\n\nTwo.\n\nThree.",
			count: 3,
		},
		{
			name:  "single newline not paragraph break",
			text:  "Line one.\nLine two.",
			count: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paras := splitParagraphs(tt.text)
			if len(paras) != tt.count {
				t.Errorf("splitParagraphs() = %d paragraphs, want %d", len(paras), tt.count)
			}
		})
	}
}

func TestSplitSentences(t *testing.T) {
	tests := []struct {
		name  string
		text  string
		count int
	}{
		{
			name:  "single sentence",
			text:  "One sentence here.",
			count: 1,
		},
		{
			name:  "two sentences",
			text:  "First sentence. Second sentence.",
			count: 2,
		},
		{
			name:  "three sentences",
			text:  "One. Two. Three.",
			count: 3,
		},
		{
			name:  "sentence without final period",
			text:  "With period. No period",
			count: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sents := splitSentences(tt.text)
			if len(sents) != tt.count {
				t.Errorf("splitSentences(%q) = %d sentences, want %d", tt.text, len(sents), tt.count)
			}
		})
	}
}

func TestSplitSentences_PeriodsRestored(t *testing.T) {
	text := "First sentence. Second sentence. Third."
	sents := splitSentences(text)

	// All but the last sentence should end with period
	for i := 0; i < len(sents)-1; i++ {
		if !strings.HasSuffix(sents[i], ".") {
			t.Errorf("Sentence %d should end with period: %q", i, sents[i])
		}
	}
}

func TestGenerateChunkID(t *testing.T) {
	id1 := generateChunkID()
	id2 := generateChunkID()

	if id1 == id2 {
		t.Error("generateChunkID() should produce unique IDs")
	}

	if !strings.HasPrefix(id1, "chunk_") {
		t.Errorf("generateChunkID() = %q, should start with 'chunk_'", id1)
	}

	// Should be UUID format after prefix
	if len(id1) < 40 { // "chunk_" + UUID
		t.Errorf("generateChunkID() = %q, too short", id1)
	}
}

func TestChunkTurn_WindowsLineEndings(t *testing.T) {
	ce := NewChunkEngine()

	// Windows style line endings
	text := "First paragraph.\r\n\r\nSecond paragraph."
	chunks, err := ce.ChunkTurn(text, "turn_win")
	if err != nil {
		t.Fatalf("ChunkTurn() error = %v", err)
	}

	// Should still create chunks (may not split perfectly on Windows newlines)
	if len(chunks) == 0 {
		t.Error("Expected chunks for Windows line endings")
	}
}

func TestChunkTurn_LongText(t *testing.T) {
	ce := NewChunkEngine()

	// Build a long text with many paragraphs and sentences
	var sb strings.Builder
	for i := 0; i < 10; i++ {
		sb.WriteString("Sentence one of paragraph. Sentence two here. Sentence three too.\n\n")
	}

	chunks, err := ce.ChunkTurn(sb.String(), "turn_long")
	if err != nil {
		t.Fatalf("ChunkTurn() error = %v", err)
	}

	// Should have many chunks
	if len(chunks) < 30 { // 1 turn + 10 paras + many sentences
		t.Errorf("Expected many chunks for long text, got %d", len(chunks))
	}
}

func TestSplitSentences_LastSentenceWithPeriod(t *testing.T) {
	// Test when the last sentence already ends with a period
	text := "First sentence. Last sentence."
	sents := splitSentences(text)

	if len(sents) != 2 {
		t.Errorf("Expected 2 sentences, got %d", len(sents))
	}

	// Last sentence should keep its period
	if len(sents) > 1 && !strings.HasSuffix(sents[1], ".") {
		t.Errorf("Last sentence should end with period: %q", sents[1])
	}
}

func TestSplitSentences_OnlySentenceWithPeriod(t *testing.T) {
	// Single sentence that already ends with period
	text := "Just one sentence."
	sents := splitSentences(text)

	if len(sents) != 1 {
		t.Errorf("Expected 1 sentence, got %d", len(sents))
	}

	if sents[0] != "Just one sentence." {
		t.Errorf("Single sentence = %q, want %q", sents[0], "Just one sentence.")
	}
}

func TestSplitSentences_EmptyAfterTrim(t *testing.T) {
	// Text with empty segments after splitting
	text := "First.  .  .  Last."
	sents := splitSentences(text)

	// Empty segments should be filtered out
	for _, sent := range sents {
		if strings.TrimSpace(sent) == "" {
			t.Error("Empty sentences should be filtered out")
		}
	}
}

func TestChunkTurn_EmptyParagraphs(t *testing.T) {
	ce := NewChunkEngine()

	// Text with empty paragraphs between non-empty ones
	text := "First paragraph.\n\n\n\nSecond paragraph."
	chunks, err := ce.ChunkTurn(text, "turn_empty_para")
	if err != nil {
		t.Fatalf("ChunkTurn() error = %v", err)
	}

	// Count paragraph chunks - empty ones should be filtered
	var paraCount int
	for _, chunk := range chunks {
		if chunk.ChunkType == models.ChunkTypeParagraph {
			paraCount++
		}
	}

	// Should have 2 non-empty paragraphs
	if paraCount != 2 {
		t.Errorf("Paragraph chunks = %d, want 2", paraCount)
	}
}

func TestChunkTurn_EmptySentences(t *testing.T) {
	ce := NewChunkEngine()

	// Text with empty sentences
	text := "First.  . Second."
	chunks, err := ce.ChunkTurn(text, "turn_empty_sent")
	if err != nil {
		t.Fatalf("ChunkTurn() error = %v", err)
	}

	// Empty sentences should be filtered
	for _, chunk := range chunks {
		if chunk.ChunkType == models.ChunkTypeSentence {
			if strings.TrimSpace(chunk.Content) == "" {
				t.Error("Empty sentence chunks should be filtered out")
			}
		}
	}
}
