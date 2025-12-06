// ABOUTME: FactScrubber extracts facts from conversation turns using LLM
// ABOUTME: Links extracted facts to blocks and turns in SQLite storage
package core

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/harper/remember-standalone/internal/llm"
	"github.com/harper/remember-standalone/internal/models"
	"github.com/harper/remember-standalone/internal/storage"
)

// FactScrubber extracts and saves facts from conversation turns
type FactScrubber struct {
	client *llm.OpenAIClient
}

// NewFactScrubber creates a new FactScrubber with the given OpenAI client
func NewFactScrubber(client *llm.OpenAIClient) *FactScrubber {
	return &FactScrubber{
		client: client,
	}
}

// ExtractAndSave extracts facts from a turn and saves them to storage
// Links facts to the specified block_id and turn_id
func (fs *FactScrubber) ExtractAndSave(turn *models.Turn, blockID string, store *storage.Storage) error {
	// Combine user message and AI response for fact extraction
	conversationText := fmt.Sprintf("User: %s\nAI: %s", turn.UserMessage, turn.AIResponse)

	// Extract facts using OpenAI client
	facts, err := fs.client.ExtractFacts(conversationText)
	if err != nil {
		return fmt.Errorf("failed to extract facts: %w", err)
	}

	// If no facts extracted, that's OK - not every conversation has extractable facts
	if len(facts) == 0 {
		return nil
	}

	// Enrich facts with IDs, block_id, turn_id, and timestamps
	for i := range facts {
		facts[i].FactID = "fact_" + uuid.New().String()
		facts[i].BlockID = blockID
		facts[i].TurnID = turn.TurnID
		facts[i].CreatedAt = time.Now()
	}

	// Save facts to storage
	if err := store.SaveFacts(facts); err != nil {
		return fmt.Errorf("failed to save facts: %w", err)
	}

	return nil
}
