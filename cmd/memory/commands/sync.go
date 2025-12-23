// ABOUTME: Sync commands for Charm cloud synchronization
// ABOUTME: Provides status, wipe, and keys management
package commands

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/charm/kv"
	"github.com/harper/remember-standalone/internal/charm"
	"github.com/harper/remember-standalone/internal/storage"
	"github.com/spf13/cobra"
)

// NewSyncCmd creates the sync command group
func NewSyncCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Manage Charm cloud synchronization",
		Long: `Manage synchronization with Charm cloud.

Memory uses Charm for automatic cloud sync via SSH keys. Your data
syncs automatically across devices linked to the same Charm account.`,
	}

	cmd.AddCommand(newSyncStatusCmd())
	cmd.AddCommand(newSyncNowCmd())
	cmd.AddCommand(newSyncRepairCmd())
	cmd.AddCommand(newSyncRepairBlocksCmd())
	cmd.AddCommand(newSyncResetCmd())
	cmd.AddCommand(newSyncWipeCmd())
	cmd.AddCommand(newSyncKeysCmd())

	return cmd
}

func newSyncStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show sync status and connection info",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := charm.GetClient()
			if err != nil {
				return fmt.Errorf("failed to connect to Charm: %w", err)
			}

			id, err := client.ID()
			if err != nil {
				fmt.Println("Status: Not connected")
				fmt.Println("Run 'memory sync keys' to check your SSH keys")
				return nil
			}

			fmt.Println("Status: Connected")
			fmt.Printf("User ID: %s\n", id)
			fmt.Printf("Host: %s\n", os.Getenv("CHARM_HOST"))

			return nil
		},
	}
}

func newSyncNowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "now",
		Short: "Force immediate sync with Charm cloud",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := charm.GetClient()
			if err != nil {
				return fmt.Errorf("failed to connect to Charm: %w", err)
			}

			fmt.Println("Syncing...")
			if err := client.Sync(); err != nil {
				return fmt.Errorf("sync failed: %w", err)
			}

			fmt.Println("Sync complete")
			return nil
		},
	}
}

func newSyncRepairCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "repair",
		Short: "Repair corrupted database",
		Long: `Repair a corrupted database by checkpointing WAL, removing SHM, and verifying integrity.

This command performs the following steps:
1. Checkpoint WAL (merge pending writes into main DB)
2. Remove stale SHM file
3. Run integrity check
4. Vacuum database

If --force is specified and integrity check fails:
5. Attempt REINDEX recovery
6. Reset from cloud as last resort

Use this when you encounter database corruption errors.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Repairing database...")
			result, err := kv.Repair("memory", force)

			if result.WalCheckpointed {
				fmt.Println("  ✓ WAL checkpointed")
			}
			if result.ShmRemoved {
				fmt.Println("  ✓ SHM file removed")
			}
			if result.IntegrityOK {
				fmt.Println("  ✓ Integrity check passed")
			} else {
				fmt.Println("  ✗ Integrity check failed")
			}
			if result.Vacuumed {
				fmt.Println("  ✓ Database vacuumed")
			}

			if err != nil {
				if !force {
					fmt.Println("\nRun with --force to attempt recovery.")
				}
				return fmt.Errorf("repair failed: %w", err)
			}

			fmt.Println("\nRepair completed successfully")
			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Attempt recovery if integrity check fails")
	return cmd
}

func newSyncRepairBlocksCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "repair-blocks",
		Short: "Repair invariant violations (e.g., multiple active blocks)",
		Long: `Repair storage invariant violations that can occur with distributed sync.

When multiple devices create blocks concurrently, you may end up with
multiple ACTIVE blocks. This command keeps only the most recent block
as ACTIVE and pauses the others.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := storage.NewStorage()
			if err != nil {
				return fmt.Errorf("failed to open storage: %w", err)
			}
			defer store.Close()

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

func newSyncResetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "reset",
		Short: "Delete local database and re-sync from cloud",
		Long: `Delete the local database and rebuild by syncing from Charm Cloud.

This discards any unsynced local data and pulls a fresh copy from the cloud.
Useful when the local database is corrupted but cloud data is intact.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			reader := bufio.NewReader(os.Stdin)

			fmt.Println("This will delete your local database and re-download from Charm Cloud.")
			fmt.Println("Any unsynced local data will be lost.")
			fmt.Println()
			fmt.Print("Continue? [y/N] ")

			input, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("failed to read input: %w", err)
			}

			input = strings.TrimSpace(input)
			if input != "y" && input != "Y" {
				fmt.Println("Reset cancelled")
				return nil
			}

			fmt.Println("\nResetting database...")
			if err := kv.Reset("memory"); err != nil {
				return fmt.Errorf("reset failed: %w", err)
			}

			fmt.Println("Database reset successfully")
			return nil
		},
	}
}

func newSyncWipeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "wipe",
		Short: "Permanently delete ALL data (local and cloud)",
		Long: `Permanently delete ALL data for this database, both local and cloud.

WARNING: This is DESTRUCTIVE and CANNOT BE UNDONE!
This will delete:
- Local database files
- All cloud backups

Use 'sync reset' if you only want to delete local data and re-sync from cloud.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			reader := bufio.NewReader(os.Stdin)

			fmt.Println("WARNING: This will permanently delete ALL data!")
			fmt.Println("This includes local AND cloud data. This cannot be undone.")
			fmt.Println()
			fmt.Print("Type 'wipe' to confirm: ")

			input, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("failed to read input: %w", err)
			}

			input = strings.TrimSpace(input)
			if input != "wipe" {
				fmt.Println("Wipe cancelled")
				return nil
			}

			fmt.Println("\nWiping all data...")
			result, err := kv.Wipe("memory")
			if err != nil {
				return fmt.Errorf("wipe failed: %w", err)
			}

			fmt.Printf("Deleted %d cloud backups\n", result.CloudBackupsDeleted)
			fmt.Printf("Deleted %d local files\n", result.LocalFilesDeleted)
			fmt.Println("\nAll data permanently deleted")

			return nil
		},
	}
}

func newSyncKeysCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "keys",
		Short: "List authorized SSH keys",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := charm.GetClient()
			if err != nil {
				return fmt.Errorf("failed to connect to Charm: %w", err)
			}

			keys, err := client.GetAuthorizedKeys()
			if err != nil {
				return fmt.Errorf("failed to get authorized keys: %w", err)
			}

			if keys == "" {
				fmt.Println("No authorized keys found")
				return nil
			}

			fmt.Println("Authorized SSH keys:")
			fmt.Println(keys)

			return nil
		},
	}
}
