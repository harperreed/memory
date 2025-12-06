// ABOUTME: Governor implements smart routing logic for HMLR memory system
// ABOUTME: Determines which of 4 routing scenarios to apply for incoming turns
package core

import (
	"fmt"

	"github.com/harper/remember-standalone/internal/models"
	"github.com/harper/remember-standalone/internal/storage"
)

// Governor is the smart router that decides routing scenarios
type Governor struct {
	storage *storage.Storage
}

// NewGovernor creates a new Governor instance
func NewGovernor(store *storage.Storage) *Governor {
	return &Governor{
		storage: store,
	}
}

// Route determines which routing scenario applies for the given turn
// Returns a RoutingDecision indicating the scenario and relevant block IDs
func (g *Governor) Route(turn *models.Turn) (models.RoutingDecision, error) {
	// Get active blocks
	activeBlocks, err := g.storage.GetActiveBridgeBlocks()
	if err != nil {
		return models.RoutingDecision{}, fmt.Errorf("failed to get active blocks: %w", err)
	}

	// Get paused blocks
	pausedBlocks, err := g.storage.GetPausedBridgeBlocks()
	if err != nil {
		return models.RoutingDecision{}, fmt.Errorf("failed to get paused blocks: %w", err)
	}

	// Scenario 3: No active blocks → create new block (first topic)
	if len(activeBlocks) == 0 {
		return models.RoutingDecision{
			Scenario:       models.NewTopicFirst,
			MatchedBlockID: "",
			ActiveBlockID:  "",
		}, nil
	}

	// Check if turn matches any active block (Scenario 1: Continuation)
	for _, block := range activeBlocks {
		if g.matchesTopic(turn, &block) {
			return models.RoutingDecision{
				Scenario:       models.TopicContinuation,
				MatchedBlockID: block.BlockID,
				ActiveBlockID:  block.BlockID,
			}, nil
		}
	}

	// Check if turn matches any paused block (Scenario 2: Resumption)
	for _, block := range pausedBlocks {
		if g.matchesTopic(turn, &block) {
			activeBlockID := ""
			if len(activeBlocks) > 0 {
				activeBlockID = activeBlocks[0].BlockID
			}
			return models.RoutingDecision{
				Scenario:       models.TopicResumption,
				MatchedBlockID: block.BlockID,
				ActiveBlockID:  activeBlockID,
			}, nil
		}
	}

	// Scenario 4: New topic while one is active → shift topics
	activeBlockID := ""
	if len(activeBlocks) > 0 {
		activeBlockID = activeBlocks[0].BlockID
	}
	return models.RoutingDecision{
		Scenario:       models.TopicShift,
		MatchedBlockID: "",
		ActiveBlockID:  activeBlockID,
	}, nil
}

// matchesTopic determines if a turn matches a block's topic based on keywords and topics
func (g *Governor) matchesTopic(turn *models.Turn, block *models.BridgeBlock) bool {
	// Match by topic label
	for _, turnTopic := range turn.Topics {
		if turnTopic == block.TopicLabel {
			return true
		}
	}

	// Match by keywords (need at least 1 keyword overlap)
	matchCount := 0
	for _, turnKeyword := range turn.Keywords {
		for _, blockKeyword := range block.Keywords {
			if g.keywordMatch(turnKeyword, blockKeyword) {
				matchCount++
			}
		}
	}

	// Consider it a match if we have at least 1 keyword overlap
	// This is a simple heuristic; could be enhanced with similarity scoring
	return matchCount > 0
}

// keywordMatch checks if two keywords match (case-insensitive)
func (g *Governor) keywordMatch(k1, k2 string) bool {
	return toLower(k1) == toLower(k2)
}

// Helper function for case-insensitive comparison
func toLower(s string) string {
	b := make([]byte, len(s))
	for i := range s {
		c := s[i]
		if 'A' <= c && c <= 'Z' {
			c += 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}
