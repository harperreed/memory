// ABOUTME: CLI command to list memories
// ABOUTME: Shows active topics and recent conversations
package commands

import (
	"encoding/json"
	"fmt"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/harper/remember-standalone/internal/storage"
	"github.com/joho/godotenv"
)

var (
	listAll bool
)

// NewListCmd creates list command
func NewListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List active topics and memories",
		Long: `List active topics and recent conversations.

Shows Bridge Blocks organized by topic with their status
(ACTIVE, PAUSED, or CLOSED).

Examples:
  memory list
  memory list --all
  memory list --format json`,
		RunE: runList,
	}

	cmd.Flags().BoolVar(&listAll, "all", false, "Show all blocks (not just active)")

	return cmd
}

func runList(cmd *cobra.Command, args []string) error {
	// Load .env for API keys
	_ = godotenv.Load()

	// Initialize storage
	store, err := storage.NewStorage()
	if err != nil {
		return fmt.Errorf("initializing storage: %w", err)
	}
	defer store.Close()

	var blocks []*storage.BridgeBlockInfo
	if listAll {
		// Get all active blocks
		activeBlocks, err := store.GetActiveBridgeBlocks()
		if err != nil {
			return fmt.Errorf("getting active blocks: %w", err)
		}

		blocks = make([]*storage.BridgeBlockInfo, len(activeBlocks))
		for i, block := range activeBlocks {
			blocks[i] = &storage.BridgeBlockInfo{
				BlockID:   block.BlockID,
				Topic:     block.TopicLabel,
				Status:    string(block.Status),
				TurnCount: len(block.Turns),
				CreatedAt: block.CreatedAt,
				UpdatedAt: block.UpdatedAt,
			}
		}
	} else {
		// Get active blocks only
		activeBlocks, err := store.GetActiveBridgeBlocks()
		if err != nil {
			return fmt.Errorf("getting active blocks: %w", err)
		}

		blocks = make([]*storage.BridgeBlockInfo, 0)
		for _, block := range activeBlocks {
			if block.Status == "ACTIVE" {
				blocks = append(blocks, &storage.BridgeBlockInfo{
					BlockID:   block.BlockID,
					Topic:     block.TopicLabel,
					Status:    string(block.Status),
					TurnCount: len(block.Turns),
					CreatedAt: block.CreatedAt,
					UpdatedAt: block.UpdatedAt,
				})
			}
		}
	}

	if len(blocks) == 0 {
		if !quiet {
			fmt.Fprintf(cmd.OutOrStdout(), "No memories found\n")
		}
		return nil
	}

	// Format output
	if outputFormat == "json" {
		jsonData, err := json.MarshalIndent(blocks, "", "  ")
		if err != nil {
			return fmt.Errorf("marshaling JSON: %w", err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "%s\n", jsonData)
	} else {
		// Table format
		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
		fmt.Fprintf(w, "TOPIC\tSTATUS\tTURNS\tCREATED\tBLOCK ID\n")
		fmt.Fprintf(w, "-----\t------\t-----\t-------\t--------\n")

		for _, block := range blocks {
			createdStr := formatTime(block.CreatedAt)
			topic := block.Topic
			if topic == "" {
				topic = "(no topic)"
			}

			fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%s\n",
				truncateList(topic, 30),
				block.Status,
				block.TurnCount,
				createdStr,
				truncateList(block.BlockID, 25))
		}
		w.Flush()

		if !quiet {
			fmt.Fprintf(cmd.OutOrStdout(), "\nTotal: %d block(s)\n", len(blocks))
		}
	}

	return nil
}

func truncateList(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}

func formatTime(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

	if diff < time.Minute {
		return "just now"
	} else if diff < time.Hour {
		mins := int(diff.Minutes())
		return fmt.Sprintf("%dm ago", mins)
	} else if diff < 24*time.Hour {
		hours := int(diff.Hours())
		return fmt.Sprintf("%dh ago", hours)
	} else if diff < 7*24*time.Hour {
		days := int(diff.Hours() / 24)
		return fmt.Sprintf("%dd ago", days)
	}
	return t.Format("2006-01-02")
}
