// ABOUTME: Install Claude Code skill for memory
// ABOUTME: Embeds and installs the skill definition to ~/.claude/skills/

package commands

import (
	"bufio"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

//go:embed skill/SKILL.md
var skillFS embed.FS

// NewInstallSkillCmd creates the install-skill command
func NewInstallSkillCmd() *cobra.Command {
	var skipConfirm bool

	cmd := &cobra.Command{
		Use:   "install-skill",
		Short: "Install Claude Code skill",
		Long: `Install the memory skill for Claude Code.

This copies the skill definition to ~/.claude/skills/memory/
so Claude Code can use memory commands contextually.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return installSkill(cmd, skipConfirm)
		},
	}

	cmd.Flags().BoolVarP(&skipConfirm, "yes", "y", false, "Skip confirmation prompt")
	return cmd
}

func installSkill(cmd *cobra.Command, skipConfirm bool) error {
	// Determine destination
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	skillDir := filepath.Join(home, ".claude", "skills", "memory")
	skillPath := filepath.Join(skillDir, "SKILL.md")

	// Show explanation
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "┌─────────────────────────────────────────────────────────────┐")
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "│             Memory Skill for Claude Code                    │")
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "└─────────────────────────────────────────────────────────────┘")
	_, _ = fmt.Fprintln(cmd.OutOrStdout())
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "This will install the memory skill, enabling Claude Code to:")
	_, _ = fmt.Fprintln(cmd.OutOrStdout())
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "  • Store and retrieve information with embeddings")
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "  • Semantic similarity search across memories")
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "  • Long-term context persistence")
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "  • Use the /memory slash command")
	_, _ = fmt.Fprintln(cmd.OutOrStdout())
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Destination:")
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", skillPath)
	_, _ = fmt.Fprintln(cmd.OutOrStdout())

	// Check if already installed
	if _, err := os.Stat(skillPath); err == nil {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Note: A skill file already exists and will be overwritten.")
		_, _ = fmt.Fprintln(cmd.OutOrStdout())
	}

	// Ask for confirmation unless --yes flag is set
	if !skipConfirm {
		_, _ = fmt.Fprint(cmd.OutOrStdout(), "Install the memory skill? [y/N] ")
		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Installation cancelled.")
			return nil
		}
		_, _ = fmt.Fprintln(cmd.OutOrStdout())
	}

	// Read embedded skill file
	content, err := skillFS.ReadFile("skill/SKILL.md")
	if err != nil {
		return fmt.Errorf("failed to read embedded skill: %w", err)
	}

	// Create directory
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		return fmt.Errorf("failed to create skill directory: %w", err)
	}

	// Write skill file
	if err := os.WriteFile(skillPath, content, 0644); err != nil {
		return fmt.Errorf("failed to write skill file: %w", err)
	}

	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "✓ Installed memory skill successfully!")
	_, _ = fmt.Fprintln(cmd.OutOrStdout())
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Claude Code will now recognize /memory commands.")
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Try asking Claude: \"Remember that the API key is stored in 1Password\" or \"What do I know about the project?\"")
	return nil
}
