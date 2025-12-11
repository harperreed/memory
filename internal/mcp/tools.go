// ABOUTME: MCP tool definitions and registration for HMLR server
// ABOUTME: Defines JSON schemas for all 11 MCP tools following DESIGN.md spec
package mcp

import (
	"sync"

	"github.com/harper/remember-standalone/internal/core"
	"github.com/harper/remember-standalone/internal/llm"
	"github.com/harper/remember-standalone/internal/storage"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// RegisterTools registers all MCP tools with the server
func RegisterTools(server *mcpserver.MCPServer, store *storage.Storage, governor *core.Governor, chunkEngine *core.ChunkEngine, scribe *core.Scribe, openaiClient *llm.OpenAIClient) *Handlers {
	// Initialize handlers
	handlers := &Handlers{
		storage:      store,
		governor:     governor,
		chunkEngine:  chunkEngine,
		scribe:       scribe,
		openaiClient: openaiClient,
		shutdownWg:   &sync.WaitGroup{},
	}

	// 1. store_conversation - Store a conversation turn in HMLR memory system
	server.AddTool(mcp.Tool{
		Name:        "store_conversation",
		Description: "Store a conversation turn in HMLR memory system. Automatically routes to the correct Bridge Block based on topic matching.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"message": map[string]interface{}{
					"type":        "string",
					"description": "User message to store",
				},
				"context": map[string]interface{}{
					"type":        "string",
					"description": "Optional additional context",
				},
			},
			Required: []string{"message"},
		},
	}, handlers.StoreConversation)

	// 2. retrieve_memory - Retrieve relevant memories from HMLR system
	server.AddTool(mcp.Tool{
		Name:        "retrieve_memory",
		Description: "Retrieve relevant memories from HMLR system based on semantic search and fact lookup.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "Search query for memory retrieval",
				},
				"max_results": map[string]interface{}{
					"type":        "number",
					"description": "Maximum number of results to return (default: 5)",
					"default":     5,
				},
			},
			Required: []string{"query"},
		},
	}, handlers.RetrieveMemory)

	// 3. list_active_topics - List all active Bridge Block topics
	server.AddTool(mcp.Tool{
		Name:        "list_active_topics",
		Description: "List all active Bridge Block topics with their metadata.",
		InputSchema: mcp.ToolInputSchema{
			Type:       "object",
			Properties: map[string]interface{}{},
		},
	}, handlers.ListActiveTopics)

	// 4. get_topic_history - Get conversation history for a specific topic
	server.AddTool(mcp.Tool{
		Name:        "get_topic_history",
		Description: "Get the complete conversation history for a specific Bridge Block topic.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"block_id": map[string]interface{}{
					"type":        "string",
					"description": "Bridge Block ID to retrieve history for",
				},
			},
			Required: []string{"block_id"},
		},
	}, handlers.GetTopicHistory)

	// 5. get_user_profile - Get the user profile summary
	server.AddTool(mcp.Tool{
		Name:        "get_user_profile",
		Description: "Get the user profile summary with preferences and topics of interest.",
		InputSchema: mcp.ToolInputSchema{
			Type:       "object",
			Properties: map[string]interface{}{},
		},
	}, handlers.GetUserProfile)

	// 6. update_user_profile - Update user profile preferences directly
	server.AddTool(mcp.Tool{
		Name:        "update_user_profile",
		Description: "Update user profile with name, preferences, or topics of interest. All fields are optional - only provided fields will be updated.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"name": map[string]interface{}{
					"type":        "string",
					"description": "User's name",
				},
				"preferences": map[string]interface{}{
					"type":        "array",
					"items":       map[string]interface{}{"type": "string"},
					"description": "User preferences to add (e.g., 'prefers dark mode', 'uses vim keybindings')",
				},
				"topics_of_interest": map[string]interface{}{
					"type":        "array",
					"items":       map[string]interface{}{"type": "string"},
					"description": "Topics the user is interested in (e.g., 'Go programming', 'distributed systems')",
				},
			},
		},
	}, handlers.UpdateUserProfile)

	// 7. add_fact - Directly add a key-value fact
	server.AddTool(mcp.Tool{
		Name:        "add_fact",
		Description: "Directly add a key-value fact to memory without storing a conversation. Useful for storing API keys, settings, or explicit user data.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"key": map[string]interface{}{
					"type":        "string",
					"description": "Fact key (e.g., 'weather_api_key', 'preferred_language', 'user_timezone')",
				},
				"value": map[string]interface{}{
					"type":        "string",
					"description": "Fact value",
				},
				"confidence": map[string]interface{}{
					"type":        "number",
					"description": "Confidence score 0.0-1.0 (default: 1.0)",
					"default":     1.0,
				},
			},
			Required: []string{"key", "value"},
		},
	}, handlers.AddFact)

	// 8. get_fact - Look up a specific fact by key
	server.AddTool(mcp.Tool{
		Name:        "get_fact",
		Description: "Look up a specific fact by its key. Returns the most recent value if multiple exist.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"key": map[string]interface{}{
					"type":        "string",
					"description": "Fact key to look up",
				},
			},
			Required: []string{"key"},
		},
	}, handlers.GetFact)

	// 9. delete_fact - Remove a fact by key
	server.AddTool(mcp.Tool{
		Name:        "delete_fact",
		Description: "Delete a fact by its key. Removes all facts with the given key.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"key": map[string]interface{}{
					"type":        "string",
					"description": "Fact key to delete",
				},
			},
			Required: []string{"key"},
		},
	}, handlers.DeleteFact)

	// 10. archive_topic - Mark a topic as archived/completed
	server.AddTool(mcp.Tool{
		Name:        "archive_topic",
		Description: "Mark a topic (Bridge Block) as archived/completed. The topic will no longer appear in active topics but its data is preserved.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"block_id": map[string]interface{}{
					"type":        "string",
					"description": "Bridge Block ID to archive",
				},
			},
			Required: []string{"block_id"},
		},
	}, handlers.ArchiveTopic)

	// 11. delete_topic - Permanently delete a topic
	server.AddTool(mcp.Tool{
		Name:        "delete_topic",
		Description: "Permanently delete a topic (Bridge Block) and all its associated data including facts and embeddings. This action cannot be undone.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"block_id": map[string]interface{}{
					"type":        "string",
					"description": "Bridge Block ID to delete",
				},
			},
			Required: []string{"block_id"},
		},
	}, handlers.DeleteTopic)

	return handlers
}
