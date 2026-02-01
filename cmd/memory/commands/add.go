// ABOUTME: CLI command to add new memories
// ABOUTME: Handles text input and memory creation with fact extraction
package commands

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/harper/remember-standalone/internal/core"
	"github.com/harper/remember-standalone/internal/llm"
	"github.com/harper/remember-standalone/internal/models"
	"github.com/harper/remember-standalone/internal/storage"
	"github.com/joho/godotenv"
)

var (
	addFile string
	addTags []string
)

// NewAddCmd creates add command
func NewAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add [text]",
		Short: "Add a new memory",
		Long: `Add a new memory from text or file.

Examples:
  memory add "Met with Alice about project X"
  memory add --file notes.txt
  memory add --tags=meeting,project-x "Discussed timeline"`,
		Args: cobra.MaximumNArgs(1),
		RunE: runAdd,
	}

	cmd.Flags().StringVar(&addFile, "file", "", "Read memory from file")
	cmd.Flags().StringSliceVar(&addTags, "tags", []string{}, "Tags for memory (comma-separated)")

	return cmd
}

func runAdd(cmd *cobra.Command, args []string) error {
	// Load .env for API keys
	_ = godotenv.Load()

	// Get memory text
	var text string
	if addFile != "" {
		data, err := os.ReadFile(addFile)
		if err != nil {
			return fmt.Errorf("reading file: %w", err)
		}
		text = string(data)
	} else if len(args) > 0 {
		text = args[0]
	} else {
		// Read from stdin
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("reading stdin: %w", err)
		}
		text = string(data)
	}

	text = strings.TrimSpace(text)
	if text == "" {
		return fmt.Errorf("no text provided")
	}

	// Initialize storage
	store, err := storage.NewStorage()
	if err != nil {
		return fmt.Errorf("initializing storage: %w", err)
	}
	defer func() { _ = store.Close() }()

	// Create turn
	turn := &models.Turn{
		TurnID:      fmt.Sprintf("turn_%s_%s", time.Now().Format("20060102_150405"), uuid.New().String()[:8]),
		Timestamp:   time.Now(),
		UserMessage: text,
		AIResponse:  "",
		Keywords:    addTags,
		Topics:      addTags,
	}

	// Extract metadata if OpenAI is available
	if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
		openaiClient, err := llm.NewOpenAIClient(apiKey)
		if err != nil {
			if verbose {
				fmt.Fprintf(os.Stderr, "Warning: Could not initialize OpenAI client: %v\n", err)
			}
		} else {
			metadata, err := openaiClient.ExtractMetadata(text)
			if err != nil {
				if verbose {
					fmt.Fprintf(os.Stderr, "Warning: Could not extract metadata: %v\n", err)
				}
			} else {
				// Extract keywords and topics from metadata
				if keywords := extractStringArray(metadata, "keywords"); len(keywords) > 0 {
					turn.Keywords = append(turn.Keywords, keywords...)
				}
				if topics := extractStringArray(metadata, "topics"); len(topics) > 0 {
					turn.Topics = append(turn.Topics, topics...)
				}
			}

			// Extract facts
			factScrubber := core.NewFactScrubber(openaiClient)
			blockID, err := store.StoreTurn(turn)
			if err != nil {
				return fmt.Errorf("storing turn: %w", err)
			}

			if err := factScrubber.ExtractAndSave(turn, blockID, store); err != nil {
				if verbose {
					fmt.Fprintf(os.Stderr, "Warning: Could not extract facts: %v\n", err)
				}
			}

			if !quiet {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "✓ Added memory %s with facts and metadata\n", turn.TurnID)
			}
			return nil
		}
	}

	// Store without OpenAI features
	blockID, err := store.StoreTurn(turn)
	if err != nil {
		return fmt.Errorf("storing turn: %w", err)
	}

	if !quiet {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "✓ Added memory %s (block: %s)\n", turn.TurnID, blockID)
	}
	return nil
}

func extractStringArray(metadata map[string]interface{}, key string) []string {
	if val, ok := metadata[key]; ok {
		if arr, ok := val.([]interface{}); ok {
			result := make([]string, 0, len(arr))
			for _, item := range arr {
				if str, ok := item.(string); ok {
					result = append(result, str)
				}
			}
			return result
		}
	}
	return []string{}
}
