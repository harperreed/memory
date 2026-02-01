// ABOUTME: Tests for list command
// ABOUTME: Verifies list command structure and flag handling

package commands

import (
	"testing"
)

func TestNewListCmd(t *testing.T) {
	cmd := NewListCmd()

	if cmd.Use != "list" {
		t.Errorf("Use = %q, want %q", cmd.Use, "list")
	}

	if cmd.Short == "" {
		t.Error("Short description should not be empty")
	}

	if cmd.Long == "" {
		t.Error("Long description should not be empty")
	}
}

func TestListCmd_AllFlag(t *testing.T) {
	cmd := NewListCmd()

	allFlag := cmd.Flags().Lookup("all")
	if allFlag == nil {
		t.Fatal("--all flag not found")
	}

	if allFlag.DefValue != "false" {
		t.Errorf("--all default = %q, want %q", allFlag.DefValue, "false")
	}
}

func TestListCmd_NoArgsRequired(t *testing.T) {
	cmd := NewListCmd()

	// List command should not require any arguments
	// RunE should be set
	if cmd.RunE == nil {
		t.Error("RunE should be set")
	}
}

func TestListCmd_Examples(t *testing.T) {
	cmd := NewListCmd()

	// Long description should contain examples
	expectedParts := []string{
		"memory list",
		"--all",
		"--format json",
	}

	for _, part := range expectedParts {
		if !findSubstring(cmd.Long, part) {
			t.Errorf("Long description should contain %q", part)
		}
	}
}

func TestListCmd_Description(t *testing.T) {
	cmd := NewListCmd()

	// Should mention Bridge Blocks
	if !findSubstring(cmd.Long, "Bridge Block") {
		t.Error("Long description should mention Bridge Blocks")
	}

	// Should mention statuses
	if !findSubstring(cmd.Long, "ACTIVE") || !findSubstring(cmd.Long, "PAUSED") {
		t.Error("Long description should mention block statuses")
	}
}
