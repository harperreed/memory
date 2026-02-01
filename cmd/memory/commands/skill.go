// ABOUTME: Install Claude Code skill for memory
// ABOUTME: Embeds and installs the skill definition to ~/.claude/skills/

package commands

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

//go:embed skill/SKILL.md
var skillFS embed.FS

// NewInstallSkillCmd creates the install-skill command
func NewInstallSkillCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install-skill",
		Short: "Install Claude Code skill",
		Long: `Install the memory skill for Claude Code.

This copies the skill definition to ~/.claude/skills/memory/
so Claude Code can use memory commands contextually.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return installSkill(cmd)
		},
	}

	return cmd
}

func installSkill(cmd *cobra.Command) error {
	// Read embedded skill file
	content, err := skillFS.ReadFile("skill/SKILL.md")
	if err != nil {
		return fmt.Errorf("failed to read embedded skill: %w", err)
	}

	// Determine destination
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	skillDir := filepath.Join(home, ".claude", "skills", "memory")
	skillPath := filepath.Join(skillDir, "SKILL.md")

	// Create directory
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		return fmt.Errorf("failed to create skill directory: %w", err)
	}

	// Write skill file
	if err := os.WriteFile(skillPath, content, 0644); err != nil {
		return fmt.Errorf("failed to write skill file: %w", err)
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Installed memory skill to %s\n", skillPath)
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Claude Code will now recognize /memory commands.")
	return nil
}
