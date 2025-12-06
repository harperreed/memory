// ABOUTME: MCP tool handler implementations for HMLR server
// ABOUTME: Contains stub implementations with proper error handling for all 5 tools
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/harper/remember-standalone/internal/core"
	"github.com/harper/remember-standalone/internal/models"
	"github.com/harper/remember-standalone/internal/storage"
	"github.com/mark3labs/mcp-go/mcp"
)

// Handlers contains the handler functions for all MCP tools
type Handlers struct {
	storage     *storage.Storage
	governor    *core.Governor
	chunkEngine *core.ChunkEngine
}

// StoreConversation handles the store_conversation tool
func (h *Handlers) StoreConversation(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Extract arguments
	message, err := request.RequireString("message")
	if err != nil {
		return mcp.NewToolResultError("message argument is required and must be a string"), nil
	}

	contextStr := request.GetString("context", "")

	// Create a turn
	turn := &models.Turn{
		TurnID:      fmt.Sprintf("turn_%s_%s", time.Now().Format("20060102_150405"), uuid.New().String()[:8]),
		Timestamp:   time.Now(),
		UserMessage: message,
		AIResponse:  contextStr, // Using context as AI response for now
		Keywords:    []string{}, // TODO: Extract keywords from message
		Topics:      []string{}, // TODO: Extract topics from message
	}

	// Get routing decision from Governor
	decision, err := h.governor.Route(turn)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("routing failed: %v", err)), nil
	}

	var blockID string
	var factsExtracted int

	// Execute routing decision
	switch decision.Scenario {
	case models.TopicContinuation:
		// Append to existing active block
		if err := h.storage.AppendTurnToBlock(decision.MatchedBlockID, turn); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to append turn: %v", err)), nil
		}
		blockID = decision.MatchedBlockID

	case models.TopicResumption:
		// Pause active block, reactivate matched block
		if decision.ActiveBlockID != "" {
			if err := h.storage.UpdateBridgeBlockStatus(decision.ActiveBlockID, models.StatusPaused); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to pause active block: %v", err)), nil
			}
		}
		if err := h.storage.UpdateBridgeBlockStatus(decision.MatchedBlockID, models.StatusActive); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to reactivate block: %v", err)), nil
		}
		if err := h.storage.AppendTurnToBlock(decision.MatchedBlockID, turn); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to append turn: %v", err)), nil
		}
		blockID = decision.MatchedBlockID

	case models.NewTopicFirst:
		// Create new block (first topic)
		blockID, err = h.storage.StoreTurn(turn)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to create new block: %v", err)), nil
		}

	case models.TopicShift:
		// Pause active block, create new block
		if decision.ActiveBlockID != "" {
			if err := h.storage.UpdateBridgeBlockStatus(decision.ActiveBlockID, models.StatusPaused); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to pause active block: %v", err)), nil
			}
		}
		blockID, err = h.storage.StoreTurn(turn)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to create new block: %v", err)), nil
		}
	}

	// Get facts count for the block
	facts, err := h.storage.GetFactsForBlock(blockID)
	if err == nil {
		factsExtracted = len(facts)
	}

	// Build response
	response := map[string]interface{}{
		"block_id":         blockID,
		"turn_id":          turn.TurnID,
		"routing_scenario": string(decision.Scenario),
		"facts_extracted":  factsExtracted,
	}

	responseJSON, err := json.Marshal(response)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal response: %v", err)), nil
	}

	return mcp.NewToolResultText(string(responseJSON)), nil
}

// RetrieveMemory handles the retrieve_memory tool
func (h *Handlers) RetrieveMemory(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Extract arguments
	query, err := request.RequireString("query")
	if err != nil {
		return mcp.NewToolResultError("query argument is required and must be a string"), nil
	}

	maxResults := request.GetInt("max_results", 5)

	// Search for relevant memories
	memories, err := h.storage.SearchMemory(query, maxResults)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("memory search failed: %v", err)), nil
	}

	// Get facts from matched blocks
	factsList := []models.Fact{}
	for _, memory := range memories {
		facts, err := h.storage.GetFactsForBlock(memory.BlockID)
		if err == nil {
			factsList = append(factsList, facts...)
		}
	}

	// Build response
	response := map[string]interface{}{
		"memories": memories,
		"facts":    factsList,
	}

	responseJSON, err := json.Marshal(response)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal response: %v", err)), nil
	}

	return mcp.NewToolResultText(string(responseJSON)), nil
}

// ListActiveTopics handles the list_active_topics tool
func (h *Handlers) ListActiveTopics(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Get all active blocks
	activeBlocks, err := h.storage.GetActiveBridgeBlocks()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get active blocks: %v", err)), nil
	}

	// Build topic summaries
	topics := make([]map[string]interface{}, 0, len(activeBlocks))
	for _, block := range activeBlocks {
		topics = append(topics, map[string]interface{}{
			"block_id":    block.BlockID,
			"topic_label": block.TopicLabel,
			"status":      string(block.Status),
			"turn_count":  block.TurnCount,
			"created_at":  block.CreatedAt.Format(time.RFC3339),
		})
	}

	// Build response
	response := map[string]interface{}{
		"topics": topics,
	}

	responseJSON, err := json.Marshal(response)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal response: %v", err)), nil
	}

	return mcp.NewToolResultText(string(responseJSON)), nil
}

// GetTopicHistory handles the get_topic_history tool
func (h *Handlers) GetTopicHistory(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Extract arguments
	blockID, err := request.RequireString("block_id")
	if err != nil {
		return mcp.NewToolResultError("block_id argument is required and must be a string"), nil
	}

	// Get the block
	block, err := h.storage.GetBridgeBlock(blockID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get block: %v", err)), nil
	}

	// Format turns for response
	turns := make([]map[string]interface{}, 0, len(block.Turns))
	for _, turn := range block.Turns {
		turns = append(turns, map[string]interface{}{
			"turn_id":      turn.TurnID,
			"timestamp":    turn.Timestamp.Format(time.RFC3339),
			"user_message": turn.UserMessage,
			"ai_response":  turn.AIResponse,
		})
	}

	// Build response
	response := map[string]interface{}{
		"block_id":    block.BlockID,
		"topic_label": block.TopicLabel,
		"turns":       turns,
		"summary":     block.Summary,
	}

	responseJSON, err := json.Marshal(response)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal response: %v", err)), nil
	}

	return mcp.NewToolResultText(string(responseJSON)), nil
}

// GetUserProfile handles the get_user_profile tool
func (h *Handlers) GetUserProfile(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Stub implementation - return placeholder profile
	// TODO: Implement actual user profile management with Scribe agent

	profile := map[string]interface{}{
		"profile": map[string]interface{}{
			"name": "User",
			"preferences": map[string]interface{}{
				"language":             "Go",
				"framework_preference": "simple_over_complex",
			},
			"topics_of_interest": []string{"MCP", "AI Memory Systems", "Go Development"},
			"last_updated":       time.Now().Format(time.RFC3339),
		},
	}

	responseJSON, err := json.Marshal(profile)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal response: %v", err)), nil
	}

	return mcp.NewToolResultText(string(responseJSON)), nil
}
