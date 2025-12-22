// ABOUTME: CLI command to search memories
// ABOUTME: Supports keyword and semantic search across stored conversations
package commands

import (
	"encoding/json"
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/harper/remember-standalone/internal/storage"
	"github.com/joho/godotenv"
)

var (
	searchLimit int
)

// NewSearchCmd creates search command
func NewSearchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search memories",
		Long: `Search memories using keyword or semantic search.

Searches across conversation history using the HMLR LatticeCrawler
with vector similarity when OpenAI is configured.

Examples:
  memory search "python programming"
  memory search --limit 10 "machine learning"
  memory search --format json "API keys"`,
		Args: cobra.ExactArgs(1),
		RunE: runSearch,
	}

	cmd.Flags().IntVar(&searchLimit, "limit", 5, "Maximum results to return")

	return cmd
}

func runSearch(cmd *cobra.Command, args []string) error {
	// Load .env for API keys
	_ = godotenv.Load()

	// Validate limit flag
	if err := validatePositiveInt(searchLimit, "limit"); err != nil {
		return err
	}

	query := args[0]

	// Initialize storage
	store, err := storage.NewStorage()
	if err != nil {
		return fmt.Errorf("initializing storage: %w", err)
	}
	defer store.Close()

	// Search memories
	results, err := store.SearchMemory(query, searchLimit)
	if err != nil {
		return fmt.Errorf("searching memories: %w", err)
	}

	if len(results) == 0 {
		if !quiet {
			fmt.Fprintf(cmd.OutOrStdout(), "No memories found for query: %s\n", query)
		}
		return nil
	}

	// Format output
	if outputFormat == "json" {
		jsonData, err := json.MarshalIndent(results, "", "  ")
		if err != nil {
			return fmt.Errorf("marshaling JSON: %w", err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "%s\n", jsonData)
	} else {
		// Table format
		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
		fmt.Fprintf(w, "SCORE\tTOPIC\tBLOCK ID\tPREVIEW\n")
		fmt.Fprintf(w, "-----\t-----\t--------\t-------\n")

		for _, result := range results {
			topic := result.TopicLabel
			if topic == "" {
				topic = "(no topic)"
			}

			preview := result.Summary
			if preview == "" && len(result.Turns) > 0 {
				preview = result.Turns[0].UserMessage
			}

			fmt.Fprintf(w, "%.3f\t%s\t%s\t%s\n",
				result.RelevanceScore,
				truncate(topic, 20),
				truncate(result.BlockID, 25),
				truncate(preview, 60))
		}
		w.Flush()

		if !quiet {
			fmt.Fprintf(cmd.OutOrStdout(), "\nFound %d result(s)\n", len(results))
		}
	}

	return nil
}
