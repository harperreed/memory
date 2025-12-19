// ABOUTME: Sync commands for Charm cloud synchronization
// ABOUTME: Provides status, wipe, and keys management
package commands

import (
	"fmt"
	"os"

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
	return &cobra.Command{
		Use:   "repair",
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

func newSyncWipeCmd() *cobra.Command {
	var confirm bool

	cmd := &cobra.Command{
		Use:   "wipe",
		Short: "Wipe all local data (nuclear option)",
		Long: `Completely wipe all local Charm data.

WARNING: This deletes all locally cached data. Your cloud data
remains intact and will be re-synced on next access.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirm {
				fmt.Println("This will wipe ALL local data!")
				fmt.Println("Run with --confirm to proceed")
				return nil
			}

			client, err := charm.GetClient()
			if err != nil {
				return fmt.Errorf("failed to connect to Charm: %w", err)
			}

			if err := client.Reset(); err != nil {
				return fmt.Errorf("failed to wipe data: %w", err)
			}

			fmt.Println("Local data wiped successfully")
			return nil
		},
	}

	cmd.Flags().BoolVar(&confirm, "confirm", false, "Confirm the wipe operation")

	return cmd
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
