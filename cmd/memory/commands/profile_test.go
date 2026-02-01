// ABOUTME: Tests for profile command
// ABOUTME: Verifies profile display and set subcommand

package commands

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestNewProfileCmd(t *testing.T) {
	cmd := NewProfileCmd()

	if cmd.Use != "profile" {
		t.Errorf("Use = %q, want %q", cmd.Use, "profile")
	}

	if cmd.Short == "" {
		t.Error("Short description should not be empty")
	}

	if cmd.Long == "" {
		t.Error("Long description should not be empty")
	}
}

func TestProfileCmd_SetSubcommand(t *testing.T) {
	cmd := NewProfileCmd()

	// Find set subcommand
	var setCmd *cobra.Command
	for _, sub := range cmd.Commands() {
		if sub.Use == "set" {
			setCmd = sub
			break
		}
	}

	if setCmd == nil {
		t.Fatal("set subcommand not found")
	}

	if setCmd.Short == "" {
		t.Error("set subcommand Short description should not be empty")
	}
}

func TestProfileCmd_SetFlags(t *testing.T) {
	cmd := NewProfileCmd()

	// Find set subcommand
	var setCmd *cobra.Command
	for _, sub := range cmd.Commands() {
		if sub.Use == "set" {
			setCmd = sub
			break
		}
	}

	if setCmd == nil {
		t.Fatal("set subcommand not found")
	}

	tests := []struct {
		flagName string
	}{
		{"name"},
		{"preference"},
		{"topic"},
	}

	for _, tt := range tests {
		t.Run(tt.flagName, func(t *testing.T) {
			flag := setCmd.Flags().Lookup(tt.flagName)
			if flag == nil {
				t.Fatalf("--%s flag not found", tt.flagName)
			}
		})
	}
}

func TestProfileCmd_Examples(t *testing.T) {
	cmd := NewProfileCmd()

	// Long description should contain examples
	expectedParts := []string{
		"memory profile",
		"--format json",
		"--name",
		"--preference",
		"--topic",
	}

	for _, part := range expectedParts {
		if !strings.Contains(cmd.Long, part) {
			t.Errorf("Long description should contain %q", part)
		}
	}
}

func TestProfileCmd_Description(t *testing.T) {
	cmd := NewProfileCmd()

	// Should mention profile stores
	if !strings.Contains(cmd.Long, "preferences") {
		t.Error("Long description should mention preferences")
	}

	if !strings.Contains(cmd.Long, "topics of interest") {
		t.Error("Long description should mention topics of interest")
	}
}
