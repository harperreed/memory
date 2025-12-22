// ABOUTME: CLI command to view and update user profile
// ABOUTME: Shows name, preferences, and topics of interest
package commands

import (
	"encoding/json"
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/harper/remember-standalone/internal/models"
	"github.com/harper/remember-standalone/internal/storage"
	"github.com/joho/godotenv"
)

var (
	profileName        string
	profilePreferences []string
	profileTopics      []string
)

// NewProfileCmd creates profile command
func NewProfileCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "profile",
		Short: "View and manage user profile",
		Long: `View and manage your user profile.

The profile stores your name, preferences, and topics of interest
that help personalize memory retrieval and context.

Examples:
  memory profile
  memory profile --format json
  memory profile set --name "Doctor Biz"
  memory profile set --preference "prefers TDD"
  memory profile set --topic "Go programming"`,
		RunE: runProfileShow,
	}

	// Add set subcommand
	setCmd := &cobra.Command{
		Use:   "set",
		Short: "Update profile fields",
		Long: `Update profile fields.

Examples:
  memory profile set --name "Doctor Biz"
  memory profile set --preference "prefers simple solutions"
  memory profile set --topic "MCP servers" --topic "Go programming"`,
		RunE: runProfileSet,
	}

	setCmd.Flags().StringVar(&profileName, "name", "", "Set user name")
	setCmd.Flags().StringArrayVar(&profilePreferences, "preference", nil, "Add a preference (can be repeated)")
	setCmd.Flags().StringArrayVar(&profileTopics, "topic", nil, "Add a topic of interest (can be repeated)")

	cmd.AddCommand(setCmd)

	return cmd
}

func runProfileShow(cmd *cobra.Command, args []string) error {
	// Load .env for API keys
	_ = godotenv.Load()

	// Initialize storage
	store, err := storage.NewStorage()
	if err != nil {
		return fmt.Errorf("initializing storage: %w", err)
	}
	defer store.Close()

	profile, err := store.GetUserProfile()
	if err != nil {
		return fmt.Errorf("getting profile: %w", err)
	}

	if profile == nil {
		if !quiet {
			fmt.Fprintf(cmd.OutOrStdout(), "No profile found. Create one with: memory profile set --name \"Your Name\"\n")
		}
		return nil
	}

	// Format output
	if outputFormat == "json" {
		jsonData, err := json.MarshalIndent(profile, "", "  ")
		if err != nil {
			return fmt.Errorf("marshaling JSON: %w", err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "%s\n", jsonData)
	} else {
		// Table format
		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)

		fmt.Fprintf(w, "FIELD\tVALUE\n")
		fmt.Fprintf(w, "-----\t-----\n")

		name := profile.Name
		if name == "" {
			name = "(not set)"
		}
		fmt.Fprintf(w, "Name\t%s\n", name)

		prefs := "(none)"
		if len(profile.Preferences) > 0 {
			prefs = strings.Join(profile.Preferences, ", ")
		}
		fmt.Fprintf(w, "Preferences\t%s\n", truncate(prefs, 60))

		topics := "(none)"
		if len(profile.TopicsOfInterest) > 0 {
			topics = strings.Join(profile.TopicsOfInterest, ", ")
		}
		fmt.Fprintf(w, "Topics\t%s\n", truncate(topics, 60))

		fmt.Fprintf(w, "Last Updated\t%s\n", formatTime(profile.LastUpdated))

		w.Flush()

		// Show full lists if truncated
		if len(profile.Preferences) > 0 && len(prefs) > 60 {
			fmt.Fprintf(cmd.OutOrStdout(), "\nPreferences:\n")
			for _, p := range profile.Preferences {
				fmt.Fprintf(cmd.OutOrStdout(), "  • %s\n", p)
			}
		}

		if len(profile.TopicsOfInterest) > 0 && len(topics) > 60 {
			fmt.Fprintf(cmd.OutOrStdout(), "\nTopics of Interest:\n")
			for _, t := range profile.TopicsOfInterest {
				fmt.Fprintf(cmd.OutOrStdout(), "  • %s\n", t)
			}
		}
	}

	return nil
}

func runProfileSet(cmd *cobra.Command, args []string) error {
	// Load .env for API keys
	_ = godotenv.Load()

	// Check if any flags were provided
	if profileName == "" && len(profilePreferences) == 0 && len(profileTopics) == 0 {
		return fmt.Errorf("no updates specified. Use --name, --preference, or --topic")
	}

	// Initialize storage
	store, err := storage.NewStorage()
	if err != nil {
		return fmt.Errorf("initializing storage: %w", err)
	}
	defer store.Close()

	// Get existing profile or create new one
	profile, err := store.GetUserProfile()
	if err != nil {
		return fmt.Errorf("getting profile: %w", err)
	}

	if profile == nil {
		profile = &models.UserProfile{}
	}

	// Apply updates
	if profileName != "" {
		profile.Name = profileName
	}

	for _, pref := range profilePreferences {
		if !containsString(profile.Preferences, pref) {
			profile.Preferences = append(profile.Preferences, pref)
		}
	}

	for _, topic := range profileTopics {
		if !containsString(profile.TopicsOfInterest, topic) {
			profile.TopicsOfInterest = append(profile.TopicsOfInterest, topic)
		}
	}

	// Save profile
	if err := store.SaveUserProfile(profile); err != nil {
		return fmt.Errorf("saving profile: %w", err)
	}

	if !quiet {
		fmt.Fprintf(cmd.OutOrStdout(), "Profile updated successfully\n")
	}

	return nil
}
