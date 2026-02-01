// ABOUTME: Tests for sync and export commands
// ABOUTME: Verifies data management command structure

package commands

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestNewSyncCmd(t *testing.T) {
	cmd := NewSyncCmd()

	if cmd.Use != "sync" {
		t.Errorf("Use = %q, want %q", cmd.Use, "sync")
	}

	if cmd.Short == "" {
		t.Error("Short description should not be empty")
	}

	if cmd.Long == "" {
		t.Error("Long description should not be empty")
	}

	// Should mention SQLite local storage
	if !strings.Contains(cmd.Long, "SQLite") && !strings.Contains(cmd.Long, "local") {
		t.Error("Long description should mention local storage")
	}
}

func TestSyncCmd_Subcommands(t *testing.T) {
	cmd := NewSyncCmd()

	expectedSubcommands := []string{
		"status",
		"repair-blocks",
	}

	for _, subCmdName := range expectedSubcommands {
		t.Run(subCmdName, func(t *testing.T) {
			found := false
			for _, sub := range cmd.Commands() {
				if sub.Use == subCmdName {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Subcommand %q not found", subCmdName)
			}
		})
	}
}

func TestSyncCmd_StatusSubcommand(t *testing.T) {
	cmd := NewSyncCmd()

	var statusCmd *cobra.Command
	for _, sub := range cmd.Commands() {
		if sub.Use == "status" {
			statusCmd = sub
			break
		}
	}

	if statusCmd == nil {
		t.Fatal("status subcommand not found")
	}

	if statusCmd.Short == "" {
		t.Error("status subcommand Short description should not be empty")
	}

	if statusCmd.RunE == nil {
		t.Error("status subcommand RunE should be set")
	}
}

func TestSyncCmd_RepairBlocksSubcommand(t *testing.T) {
	cmd := NewSyncCmd()

	var repairCmd *cobra.Command
	for _, sub := range cmd.Commands() {
		if sub.Use == "repair-blocks" {
			repairCmd = sub
			break
		}
	}

	if repairCmd == nil {
		t.Fatal("repair-blocks subcommand not found")
	}

	if repairCmd.Short == "" {
		t.Error("repair-blocks subcommand Short description should not be empty")
	}

	if repairCmd.Long == "" {
		t.Error("repair-blocks subcommand Long description should not be empty")
	}

	// Should explain what it repairs
	if !strings.Contains(repairCmd.Long, "ACTIVE") || !strings.Contains(repairCmd.Long, "invariant") {
		t.Error("repair-blocks Long description should explain the invariant")
	}
}

func TestNewExportCmd(t *testing.T) {
	cmd := NewExportCmd()

	if cmd.Use != "export" {
		t.Errorf("Use = %q, want %q", cmd.Use, "export")
	}

	if cmd.Short == "" {
		t.Error("Short description should not be empty")
	}

	if cmd.Long == "" {
		t.Error("Long description should not be empty")
	}
}

func TestExportCmd_Flags(t *testing.T) {
	cmd := NewExportCmd()

	tests := []struct {
		flagName  string
		shorthand string
		defValue  string
	}{
		{"output", "o", ""},
		{"format", "f", "yaml"},
	}

	for _, tt := range tests {
		t.Run(tt.flagName, func(t *testing.T) {
			flag := cmd.Flags().Lookup(tt.flagName)
			if flag == nil {
				t.Fatalf("--%s flag not found", tt.flagName)
			}

			if tt.shorthand != "" && flag.Shorthand != tt.shorthand {
				t.Errorf("--%s shorthand = %q, want %q", tt.flagName, flag.Shorthand, tt.shorthand)
			}

			if flag.DefValue != tt.defValue {
				t.Errorf("--%s default = %q, want %q", tt.flagName, flag.DefValue, tt.defValue)
			}
		})
	}
}

func TestExportCmd_FormatOptions(t *testing.T) {
	cmd := NewExportCmd()

	// Long description should mention supported formats
	expectedFormats := []string{
		"yaml",
		"markdown",
	}

	for _, format := range expectedFormats {
		if !strings.Contains(strings.ToLower(cmd.Long), format) {
			t.Errorf("Long description should mention %q format", format)
		}
	}
}

func TestExportCmd_Examples(t *testing.T) {
	cmd := NewExportCmd()

	// Long description should contain examples
	expectedParts := []string{
		"memory export",
		"-o",
	}

	for _, part := range expectedParts {
		if !strings.Contains(cmd.Long, part) {
			t.Errorf("Long description should contain %q", part)
		}
	}
}

func TestDefaultDBPath_NotEmpty(t *testing.T) {
	path := DefaultDBPath()

	if path == "" {
		t.Error("DefaultDBPath() returned empty string")
	}
}
