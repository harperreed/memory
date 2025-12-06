// ABOUTME: Main entry point for HMLR MCP server with stdio transport
// ABOUTME: Initializes storage, governor, and MCP server with all tools
package main

import (
	"log"
	"os"

	"github.com/harper/remember-standalone/internal/core"
	"github.com/harper/remember-standalone/internal/mcp"
	"github.com/harper/remember-standalone/internal/storage"
	"github.com/joho/godotenv"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

func main() {
	// Load .env file if it exists (for API keys)
	if err := godotenv.Load(); err != nil {
		log.Printf("No .env file found (this is okay for production): %v", err)
	}

	// Verify we have required API keys
	if os.Getenv("OPENAI_API_KEY") == "" {
		log.Println("Warning: OPENAI_API_KEY not set - embeddings and LLM features will not work")
	}

	// Initialize storage with XDG-compliant paths
	store, err := storage.NewStorage()
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}
	defer store.Close()

	// Initialize Governor for smart routing
	governor := core.NewGovernor(store)

	// Initialize ChunkEngine for hierarchical chunking
	chunkEngine := core.NewChunkEngine()

	// Create MCP server
	server := mcpserver.NewMCPServer(
		"HMLR Memory System",
		"0.1.0",
	)

	// Register MCP tools
	mcp.RegisterTools(server, store, governor, chunkEngine)

	// Start server with stdio transport
	log.Println("HMLR MCP server starting on stdio...")
	if err := mcpserver.ServeStdio(server); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
