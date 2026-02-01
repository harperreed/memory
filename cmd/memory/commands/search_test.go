// ABOUTME: Tests for search command
// ABOUTME: Verifies search command structure and flag validation

package commands

import (
	"testing"
)

func TestNewSearchCmd(t *testing.T) {
	cmd := NewSearchCmd()

	if cmd.Use != "search <query>" {
		t.Errorf("Use = %q, want %q", cmd.Use, "search <query>")
	}

	if cmd.Short == "" {
		t.Error("Short description should not be empty")
	}

	if cmd.Long == "" {
		t.Error("Long description should not be empty")
	}
}

func TestSearchCmd_LimitFlag(t *testing.T) {
	cmd := NewSearchCmd()

	limitFlag := cmd.Flags().Lookup("limit")
	if limitFlag == nil {
		t.Fatal("--limit flag not found")
	}

	if limitFlag.DefValue != "5" {
		t.Errorf("--limit default = %q, want %q", limitFlag.DefValue, "5")
	}
}

func TestSearchCmd_ArgsValidation(t *testing.T) {
	cmd := NewSearchCmd()

	// Should require exactly 1 argument
	if cmd.Args == nil {
		t.Error("Args validator should be set")
	}
}

func TestSearchCmd_Examples(t *testing.T) {
	cmd := NewSearchCmd()

	// Long description should contain examples
	expectedParts := []string{
		"--limit",
		"--format json",
	}

	for _, part := range expectedParts {
		if !findSubstring(cmd.Long, part) {
			t.Errorf("Long description should contain %q", part)
		}
	}
}

func TestSearchCmd_Description(t *testing.T) {
	cmd := NewSearchCmd()

	// Should mention HMLR or semantic search
	if !findSubstring(cmd.Long, "LatticeCrawler") && !findSubstring(cmd.Long, "semantic") {
		t.Error("Long description should mention LatticeCrawler or semantic search")
	}
}
