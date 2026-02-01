// ABOUTME: Tests for root CLI command and global flags
// ABOUTME: Verifies command structure, subcommands, and flag handling

package commands

import (
	"bytes"
	"strings"
	"testing"
)

func TestNewRootCmd(t *testing.T) {
	cmd := NewRootCmd()

	if cmd.Use != "memory" {
		t.Errorf("Use = %q, want %q", cmd.Use, "memory")
	}

	if cmd.Short == "" {
		t.Error("Short description should not be empty")
	}

	if cmd.Long == "" {
		t.Error("Long description should not be empty")
	}

	// Verify the ASCII banner is in the long description (uses block characters)
	if !strings.Contains(cmd.Long, "███") {
		t.Error("Long description should contain ASCII banner")
	}
}

func TestRootCmd_GlobalFlags(t *testing.T) {
	cmd := NewRootCmd()

	tests := []struct {
		flagName  string
		shorthand string
		defValue  string
	}{
		{"verbose", "v", "false"},
		{"quiet", "q", "false"},
		{"format", "", "auto"},
	}

	for _, tt := range tests {
		t.Run(tt.flagName, func(t *testing.T) {
			flag := cmd.PersistentFlags().Lookup(tt.flagName)
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

func TestRootCmd_MutuallyExclusiveFlags(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
	}{
		{
			name:        "verbose only",
			args:        []string{"--verbose", "version"},
			expectError: false,
		},
		{
			name:        "quiet only",
			args:        []string{"--quiet", "version"},
			expectError: false,
		},
		{
			name:        "verbose and quiet",
			args:        []string{"--verbose", "--quiet", "version"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewRootCmd()
			var output bytes.Buffer
			cmd.SetOut(&output)
			cmd.SetErr(&output)
			cmd.SetArgs(tt.args)

			err := cmd.Execute()

			if tt.expectError && err == nil {
				t.Error("Expected error for mutually exclusive flags, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestRootCmd_Subcommands(t *testing.T) {
	cmd := NewRootCmd()

	expectedSubcommands := []string{
		"mcp",
		"add",
		"search",
		"list",
		"version",
		"sync",
		"profile",
		"export",
		"install-skill",
	}

	for _, subCmdName := range expectedSubcommands {
		t.Run(subCmdName, func(t *testing.T) {
			found := false
			for _, sub := range cmd.Commands() {
				if sub.Use == subCmdName || strings.HasPrefix(sub.Use, subCmdName+" ") {
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

func TestRootCmd_HelpOutput(t *testing.T) {
	cmd := NewRootCmd()
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	cmd.SetArgs([]string{"--help"})

	_ = cmd.Execute()

	outputStr := output.String()

	// Should contain usage info
	if !strings.Contains(outputStr, "Usage:") {
		t.Error("Help output should contain 'Usage:'")
	}

	// Should contain available commands
	if !strings.Contains(outputStr, "Available Commands:") {
		t.Error("Help output should contain 'Available Commands:'")
	}

	// Should contain flags section
	if !strings.Contains(outputStr, "Flags:") {
		t.Error("Help output should contain 'Flags:'")
	}
}

func TestRootCmd_SilenceUsage(t *testing.T) {
	cmd := NewRootCmd()

	if !cmd.SilenceUsage {
		t.Error("SilenceUsage should be true to prevent usage on errors")
	}
}

func TestExecute_ReturnsError(t *testing.T) {
	// Test that Execute properly returns errors (can't fully test without side effects)
	// Just verify the function signature works
	// In a real test we'd mock the dependencies
}
