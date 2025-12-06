// ABOUTME: Governor implements smart routing logic for HMLR memory system
// ABOUTME: Determines which of 4 routing scenarios to apply for incoming turns
package core

import (
	"fmt"
	"strings"

	"github.com/harper/remember-standalone/internal/models"
	"github.com/harper/remember-standalone/internal/storage"
)

// Governor is the smart router that decides routing scenarios
type Governor struct {
	storage               *storage.Storage
	topicMatchThreshold   float64 // Threshold for keyword overlap (0.0-1.0, default 0.3 for 30%)
}

// NewGovernor creates a new Governor instance
func NewGovernor(store *storage.Storage) *Governor {
	return &Governor{
		storage:             store,
		topicMatchThreshold: 0.3, // Default to 30% keyword overlap
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

	// Match by keywords - require at least 30% keyword overlap
	if len(turn.Keywords) == 0 || len(block.Keywords) == 0 {
		return false
	}

	matchCount := 0
	for _, turnKeyword := range turn.Keywords {
		for _, blockKeyword := range block.Keywords {
			if g.keywordMatch(turnKeyword, blockKeyword) {
				matchCount++
				break // Count each turn keyword only once
			}
		}
	}

	// Calculate overlap as a percentage of turn keywords
	overlapRatio := float64(matchCount) / float64(len(turn.Keywords))
	return overlapRatio >= g.topicMatchThreshold
}

// keywordMatch checks if two keywords match (case-insensitive)
func (g *Governor) keywordMatch(k1, k2 string) bool {
	return strings.ToLower(k1) == strings.ToLower(k2)
}
