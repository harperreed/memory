// ABOUTME: MCP command starts Model Context Protocol server
// ABOUTME: Enables LLM agents like Claude to use Memory via stdio
package commands

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/harper/remember-standalone/internal/core"
	"github.com/harper/remember-standalone/internal/llm"
	"github.com/harper/remember-standalone/internal/mcp"
	"github.com/harper/remember-standalone/internal/storage"
	"github.com/joho/godotenv"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// NewMCPCmd creates the MCP command
func NewMCPCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Start MCP server for LLM agents",
		Long: `Start MCP server for LLM agents

Runs Memory as an MCP (Model Context Protocol) server, enabling
LLM agents like Claude to use hierarchical memory via stdio.

Configure in Claude Desktop's config file to enable memory tools.`,
		RunE: runMCP,
		Example: `  # Start MCP server (typically called by Claude Desktop)
  memory mcp

  # Configure in claude_desktop_config.json:
  # {
  #   "mcpServers": {
  #     "memory": {
  #       "command": "memory",
  #       "args": ["mcp"]
  #     }
  #   }
  # }`,
	}

	return cmd
}

// runMCP starts the MCP server
func runMCP(cmd *cobra.Command, args []string) error {
	// Load .env file if it exists (for API keys)
	if err := godotenv.Load(); err != nil && !quiet {
		log.Printf("No .env file found (this is okay for production): %v", err)
	}

	// Verify we have required API keys
	if os.Getenv("OPENAI_API_KEY") == "" {
		log.Println("Warning: OPENAI_API_KEY not set - embeddings and LLM features will not work")
	}

	// Initialize storage with XDG-compliant paths
	store, err := storage.NewStorage()
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
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
			if verbose {
				log.Println("OpenAI client and Scribe agent initialized")
			}
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

	if !quiet {
		log.Println("HMLR MCP server starting on stdio...")
	}

	// Start server in goroutine
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- mcpserver.ServeStdio(server)
	}()

	// Wait for shutdown signal or server error
	select {
	case <-ctx.Done():
		if !quiet {
			log.Println("Shutdown signal received, gracefully shutting down...")
		}

		// Wait for all async Scribe operations to complete
		handlers.Shutdown()

		// Close storage (flushes pending writes, closes DB)
		if err := store.Close(); err != nil {
			log.Printf("Warning: Error closing storage: %v", err)
		}

		if !quiet {
			log.Println("Shutdown complete")
		}

	case err := <-serverErr:
		if err != nil {
			return fmt.Errorf("server error: %w", err)
		}
	}

	return nil
}
