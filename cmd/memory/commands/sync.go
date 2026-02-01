// ABOUTME: Data management commands for memory storage
// ABOUTME: Provides export and repair functionality (no cloud sync)
package commands

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/harper/remember-standalone/internal/storage"
	"github.com/spf13/cobra"
)

// NewSyncCmd creates the sync command group (kept for backward compatibility)
// Note: Cloud sync has been removed. This now provides local data management.
func NewSyncCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Manage memory data (local storage)",
		Long: `Manage memory data in local SQLite storage.

Note: Cloud sync has been removed. Memory now uses local SQLite storage.
Data is stored at ~/.local/share/memory/memory.db

Use 'memory export' for data backup and migration.`,
	}

	cmd.AddCommand(newSyncStatusCmd())
	cmd.AddCommand(newSyncRepairBlocksCmd())

	return cmd
}

func newSyncStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show storage status",
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := storage.NewStorage()
			if err != nil {
				return fmt.Errorf("failed to open storage: %w", err)
			}
			defer func() { _ = store.Close() }()

			fmt.Println("Storage: SQLite (local)")
			fmt.Printf("Database: %s\n", storage.DefaultDBPath())

			// Count blocks
			active, err := store.GetActiveBridgeBlocks()
			if err == nil {
				fmt.Printf("Active blocks: %d\n", len(active))
			}

			paused, err := store.GetPausedBridgeBlocks()
			if err == nil {
				fmt.Printf("Paused blocks: %d\n", len(paused))
			}

			// Get user profile
			profile, err := store.GetUserProfile()
			if err == nil && profile != nil {
				fmt.Printf("Profile: %s\n", profile.Name)
			}

			return nil
		},
	}
}

func newSyncRepairBlocksCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "repair-blocks",
		Short: "Repair invariant violations (e.g., multiple active blocks)",
		Long: `Repair storage invariant violations.

If multiple blocks are marked as ACTIVE (which violates the single-active-block
invariant), this command keeps only the most recent block as ACTIVE and pauses
the others.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := storage.NewStorage()
			if err != nil {
				return fmt.Errorf("failed to open storage: %w", err)
			}
			defer func() { _ = store.Close() }()

			repaired, err := store.RepairActiveBlockInvariant()
			if err != nil {
				return fmt.Errorf("repair failed: %w", err)
			}

			if repaired {
				fmt.Println("Repaired: Multiple active blocks found and fixed")
				fmt.Println("Only the most recent block is now ACTIVE")
			} else {
				fmt.Println("No repair needed: Invariant is satisfied")
			}

			return nil
		},
	}
}

// NewExportCmd creates the export command
func NewExportCmd() *cobra.Command {
	var (
		outputPath string
		format     string
	)

	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export memory data to file",
		Long: `Export all memory data to YAML or Markdown format.

Formats:
  yaml      Machine-readable YAML export (default)
  markdown  Human-readable Markdown export

Examples:
  memory export                           # Export to memory-export-2026-01-31.yaml
  memory export -o backup.yaml            # Export to specific file
  memory export -f markdown -o readme.md  # Export as Markdown`,
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := storage.NewStorage()
			if err != nil {
				return fmt.Errorf("failed to open storage: %w", err)
			}
			defer func() { _ = store.Close() }()

			// Generate default output path if not specified
			if outputPath == "" {
				dateStr := time.Now().Format("2006-01-02")
				switch format {
				case "markdown", "md":
					outputPath = fmt.Sprintf("memory-export-%s.md", dateStr)
				default:
					outputPath = fmt.Sprintf("memory-export-%s.yaml", dateStr)
				}
			}

			// Make absolute path
			if !filepath.IsAbs(outputPath) {
				outputPath, _ = filepath.Abs(outputPath)
			}

			switch format {
			case "markdown", "md":
				if err := store.ExportToMarkdown(outputPath); err != nil {
					return fmt.Errorf("export failed: %w", err)
				}
			default:
				if err := store.ExportToYAML(outputPath); err != nil {
					return fmt.Errorf("export failed: %w", err)
				}
			}

			fmt.Printf("Exported to: %s\n", outputPath)
			return nil
		},
	}

	cmd.Flags().StringVarP(&outputPath, "output", "o", "", "Output file path")
	cmd.Flags().StringVarP(&format, "format", "f", "yaml", "Output format (yaml, markdown)")

	return cmd
}

// DefaultDBPath returns the default database path for display
func DefaultDBPath() string {
	return storage.DefaultDBPath()
}
