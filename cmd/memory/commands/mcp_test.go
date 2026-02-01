// ABOUTME: Tests for MCP command structure
// ABOUTME: Verifies MCP command configuration

package commands

import (
	"strings"
	"testing"
)

func TestNewMCPCmd(t *testing.T) {
	cmd := NewMCPCmd()

	if cmd.Use != "mcp" {
		t.Errorf("Use = %q, want %q", cmd.Use, "mcp")
	}

	if cmd.Short == "" {
		t.Error("Short description should not be empty")
	}

	if cmd.Long == "" {
		t.Error("Long description should not be empty")
	}
}

func TestMCPCmd_Description(t *testing.T) {
	cmd := NewMCPCmd()

	// Should mention MCP/Model Context Protocol
	if !strings.Contains(cmd.Long, "MCP") && !strings.Contains(cmd.Long, "Model Context Protocol") {
		t.Error("Long description should mention MCP")
	}

	// Should mention LLM agents
	if !strings.Contains(cmd.Long, "LLM") {
		t.Error("Long description should mention LLM agents")
	}

	// Should mention stdio
	if !strings.Contains(cmd.Long, "stdio") {
		t.Error("Long description should mention stdio")
	}
}

func TestMCPCmd_HasRunE(t *testing.T) {
	cmd := NewMCPCmd()

	if cmd.RunE == nil {
		t.Error("RunE should be set")
	}
}

func TestMCPCmd_Example(t *testing.T) {
	cmd := NewMCPCmd()

	if cmd.Example == "" {
		t.Error("Example should not be empty")
	}

	// Should contain example of running
	if !strings.Contains(cmd.Example, "memory mcp") {
		t.Error("Example should show how to run the command")
	}

	// Should mention claude_desktop_config
	if !strings.Contains(cmd.Example, "claude_desktop_config") {
		t.Error("Example should mention Claude Desktop config")
	}
}
