// ABOUTME: Main entry point for HMLR MCP server with stdio transport
// ABOUTME: Initializes storage, governor, and MCP server with all tools
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/harper/remember-standalone/internal/core"
	"github.com/harper/remember-standalone/internal/llm"
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

	// Initialize Governor for smart routing
	governor := core.NewGovernor(store)

	// Initialize ChunkEngine for hierarchical chunking
	chunkEngine := core.NewChunkEngine()

	// Initialize OpenAI client and Scribe for user profile learning (optional - only if API key is set)
	var scribe *core.Scribe
	var openaiClient *llm.OpenAIClient
	if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
		client, err := llm.NewOpenAIClient(apiKey)
		if err != nil {
			log.Printf("Warning: Failed to initialize OpenAI client: %v", err)
		} else {
			openaiClient = client
			scribe = core.NewScribe(openaiClient)
			log.Println("OpenAI client and Scribe agent initialized")
		}
	}

	// Create MCP server
	server := mcpserver.NewMCPServer(
		"HMLR Memory System",
		"0.1.0",
	)

	// Register MCP tools and get handlers for shutdown
	handlers := mcp.RegisterTools(server, store, governor, chunkEngine, scribe, openaiClient)

	// Setup graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(),
		os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Println("HMLR MCP server starting on stdio...")

	// Start server in goroutine
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- mcpserver.ServeStdio(server)
	}()

	// Wait for shutdown signal or server error
	select {
	case <-ctx.Done():
		log.Println("Shutdown signal received, gracefully shutting down...")

		// Wait for all async Scribe operations to complete
		handlers.Shutdown()

		// Close storage (flushes pending writes, closes DB)
		if err := store.Close(); err != nil {
			log.Printf("Warning: Error closing storage: %v", err)
		}

		log.Println("Shutdown complete")

	case err := <-serverErr:
		if err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}
}
