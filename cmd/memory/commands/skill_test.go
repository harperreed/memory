// ABOUTME: Tests for the install-skill command
// ABOUTME: Verifies skill installation, directory creation, and file content

package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewInstallSkillCmd(t *testing.T) {
	cmd := NewInstallSkillCmd()

	if cmd.Use != "install-skill" {
		t.Errorf("Use = %q, want %q", cmd.Use, "install-skill")
	}

	if cmd.Short == "" {
		t.Error("Short description should not be empty")
	}

	if cmd.Long == "" {
		t.Error("Long description should not be empty")
	}

	// Verify the --yes flag exists
	yesFlag := cmd.Flags().Lookup("yes")
	if yesFlag == nil {
		t.Fatal("--yes flag should exist")
	}
	if yesFlag.Shorthand != "y" {
		t.Errorf("--yes shorthand = %q, want %q", yesFlag.Shorthand, "y")
	}
}

func TestInstallSkill_SuccessfulInstallation(t *testing.T) {
	// Create a temporary home directory
	tmpHome := t.TempDir()
	originalHome := os.Getenv("HOME")
	t.Setenv("HOME", tmpHome)
	defer func() {
		_ = os.Setenv("HOME", originalHome)
	}()

	cmd := NewInstallSkillCmd()
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetErr(&output)

	// Set the --yes flag to skip confirmation
	if err := cmd.Flags().Set("yes", "true"); err != nil {
		t.Fatalf("Failed to set yes flag: %v", err)
	}

	// Execute the command
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Command execution failed: %v", err)
	}

	// Verify the skill directory was created
	skillDir := filepath.Join(tmpHome, ".claude", "skills", "memory")
	if _, err := os.Stat(skillDir); os.IsNotExist(err) {
		t.Errorf("Skill directory was not created at %s", skillDir)
	}

	// Verify the SKILL.md file was created
	skillPath := filepath.Join(skillDir, "SKILL.md")
	if _, err := os.Stat(skillPath); os.IsNotExist(err) {
		t.Errorf("SKILL.md file was not created at %s", skillPath)
	}

	// Verify success message in output
	outputStr := output.String()
	if !strings.Contains(outputStr, "Installed memory skill successfully") {
		t.Errorf("Output should contain success message, got: %s", outputStr)
	}
}

func TestInstallSkill_FileContent(t *testing.T) {
	tmpHome := t.TempDir()
	originalHome := os.Getenv("HOME")
	t.Setenv("HOME", tmpHome)
	defer func() {
		_ = os.Setenv("HOME", originalHome)
	}()

	cmd := NewInstallSkillCmd()
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetErr(&output)

	if err := cmd.Flags().Set("yes", "true"); err != nil {
		t.Fatalf("Failed to set yes flag: %v", err)
	}

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Command execution failed: %v", err)
	}

	// Read the installed SKILL.md file
	skillPath := filepath.Join(tmpHome, ".claude", "skills", "memory", "SKILL.md")
	content, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("Failed to read installed SKILL.md: %v", err)
	}

	// Verify expected content from the embedded file
	contentStr := string(content)
	expectedStrings := []string{
		"name: memory",
		"Semantic memory and context",
		"mcp__memory__store",
		"mcp__memory__search",
		"mcp__memory__list",
		"mcp__memory__delete",
		"mcp__memory__get_context",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(contentStr, expected) {
			t.Errorf("SKILL.md should contain %q", expected)
		}
	}
}

func TestInstallSkill_DirectoryCreation(t *testing.T) {
	tmpHome := t.TempDir()
	originalHome := os.Getenv("HOME")
	t.Setenv("HOME", tmpHome)
	defer func() {
		_ = os.Setenv("HOME", originalHome)
	}()

	// Ensure the .claude directory doesn't exist yet
	claudeDir := filepath.Join(tmpHome, ".claude")
	if _, err := os.Stat(claudeDir); !os.IsNotExist(err) {
		t.Fatal(".claude directory should not exist before test")
	}

	cmd := NewInstallSkillCmd()
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetErr(&output)

	if err := cmd.Flags().Set("yes", "true"); err != nil {
		t.Fatalf("Failed to set yes flag: %v", err)
	}

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Command execution failed: %v", err)
	}

	// Verify nested directory structure was created
	expectedDirs := []string{
		filepath.Join(tmpHome, ".claude"),
		filepath.Join(tmpHome, ".claude", "skills"),
		filepath.Join(tmpHome, ".claude", "skills", "memory"),
	}

	for _, dir := range expectedDirs {
		info, err := os.Stat(dir)
		if os.IsNotExist(err) {
			t.Errorf("Directory was not created: %s", dir)
			continue
		}
		if !info.IsDir() {
			t.Errorf("%s should be a directory", dir)
		}
	}
}

func TestInstallSkill_OverwriteScenario(t *testing.T) {
	tmpHome := t.TempDir()
	originalHome := os.Getenv("HOME")
	t.Setenv("HOME", tmpHome)
	defer func() {
		_ = os.Setenv("HOME", originalHome)
	}()

	// Pre-create the skill directory and file with different content
	skillDir := filepath.Join(tmpHome, ".claude", "skills", "memory")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatalf("Failed to create skill directory: %v", err)
	}

	skillPath := filepath.Join(skillDir, "SKILL.md")
	oldContent := "# Old skill content\nThis should be overwritten"
	if err := os.WriteFile(skillPath, []byte(oldContent), 0644); err != nil {
		t.Fatalf("Failed to write old skill file: %v", err)
	}

	// Execute the install command
	cmd := NewInstallSkillCmd()
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetErr(&output)

	if err := cmd.Flags().Set("yes", "true"); err != nil {
		t.Fatalf("Failed to set yes flag: %v", err)
	}

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Command execution failed: %v", err)
	}

	// Verify the output mentions overwriting
	outputStr := output.String()
	if !strings.Contains(outputStr, "already exists") {
		t.Errorf("Output should mention existing file, got: %s", outputStr)
	}

	// Verify the file was overwritten with new content
	newContent, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("Failed to read skill file: %v", err)
	}

	if string(newContent) == oldContent {
		t.Error("SKILL.md should have been overwritten with new content")
	}

	if !strings.Contains(string(newContent), "name: memory") {
		t.Error("Overwritten SKILL.md should contain the embedded content")
	}
}

func TestInstallSkill_OutputContainsDestination(t *testing.T) {
	tmpHome := t.TempDir()
	originalHome := os.Getenv("HOME")
	t.Setenv("HOME", tmpHome)
	defer func() {
		_ = os.Setenv("HOME", originalHome)
	}()

	cmd := NewInstallSkillCmd()
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetErr(&output)

	if err := cmd.Flags().Set("yes", "true"); err != nil {
		t.Fatalf("Failed to set yes flag: %v", err)
	}

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Command execution failed: %v", err)
	}

	// Verify the output contains the destination path
	outputStr := output.String()
	expectedPath := filepath.Join(tmpHome, ".claude", "skills", "memory", "SKILL.md")
	if !strings.Contains(outputStr, expectedPath) {
		t.Errorf("Output should contain destination path %q, got: %s", expectedPath, outputStr)
	}
}

func TestInstallSkill_OutputContainsExplanation(t *testing.T) {
	tmpHome := t.TempDir()
	originalHome := os.Getenv("HOME")
	t.Setenv("HOME", tmpHome)
	defer func() {
		_ = os.Setenv("HOME", originalHome)
	}()

	cmd := NewInstallSkillCmd()
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetErr(&output)

	if err := cmd.Flags().Set("yes", "true"); err != nil {
		t.Fatalf("Failed to set yes flag: %v", err)
	}

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Command execution failed: %v", err)
	}

	outputStr := output.String()

	// Verify the output contains the explanation box
	expectedStrings := []string{
		"Memory Skill for Claude Code",
		"Store and retrieve information with embeddings",
		"Semantic similarity search",
		"/memory slash command",
		"Destination:",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(outputStr, expected) {
			t.Errorf("Output should contain %q, got: %s", expected, outputStr)
		}
	}
}

func TestInstallSkill_FilePermissions(t *testing.T) {
	tmpHome := t.TempDir()
	originalHome := os.Getenv("HOME")
	t.Setenv("HOME", tmpHome)
	defer func() {
		_ = os.Setenv("HOME", originalHome)
	}()

	cmd := NewInstallSkillCmd()
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetErr(&output)

	if err := cmd.Flags().Set("yes", "true"); err != nil {
		t.Fatalf("Failed to set yes flag: %v", err)
	}

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Command execution failed: %v", err)
	}

	// Check file permissions
	skillPath := filepath.Join(tmpHome, ".claude", "skills", "memory", "SKILL.md")
	info, err := os.Stat(skillPath)
	if err != nil {
		t.Fatalf("Failed to stat skill file: %v", err)
	}

	// File should be readable by owner (0644 = -rw-r--r--)
	mode := info.Mode().Perm()
	if mode&0600 != 0600 {
		t.Errorf("File should be readable/writable by owner, got mode %o", mode)
	}
}

func TestInstallSkill_CobraCommandFlags(t *testing.T) {
	cmd := NewInstallSkillCmd()

	// Test default value of --yes flag
	yesFlag := cmd.Flags().Lookup("yes")
	if yesFlag == nil {
		t.Fatal("--yes flag not found")
	}

	if yesFlag.DefValue != "false" {
		t.Errorf("--yes default value = %q, want %q", yesFlag.DefValue, "false")
	}

	// Test that the flag can be set
	if err := cmd.Flags().Set("yes", "true"); err != nil {
		t.Errorf("Failed to set --yes flag: %v", err)
	}

	val, err := cmd.Flags().GetBool("yes")
	if err != nil {
		t.Errorf("Failed to get --yes flag value: %v", err)
	}
	if !val {
		t.Error("--yes flag should be true after setting")
	}
}

func TestInstallSkill_MultipleInstallations(t *testing.T) {
	tmpHome := t.TempDir()
	originalHome := os.Getenv("HOME")
	t.Setenv("HOME", tmpHome)
	defer func() {
		_ = os.Setenv("HOME", originalHome)
	}()

	// Run installation twice
	for i := 0; i < 2; i++ {
		cmd := NewInstallSkillCmd()
		var output bytes.Buffer
		cmd.SetOut(&output)
		cmd.SetErr(&output)

		if err := cmd.Flags().Set("yes", "true"); err != nil {
			t.Fatalf("Iteration %d: Failed to set yes flag: %v", i, err)
		}

		if err := cmd.Execute(); err != nil {
			t.Fatalf("Iteration %d: Command execution failed: %v", i, err)
		}
	}

	// Verify the file still exists and has correct content
	skillPath := filepath.Join(tmpHome, ".claude", "skills", "memory", "SKILL.md")
	content, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("Failed to read skill file after multiple installations: %v", err)
	}

	if !strings.Contains(string(content), "name: memory") {
		t.Error("SKILL.md should contain expected content after multiple installations")
	}
}

func TestSkillFS_EmbeddedFileReadable(t *testing.T) {
	// Test that the embedded file system contains the skill file
	content, err := skillFS.ReadFile("skill/SKILL.md")
	if err != nil {
		t.Fatalf("Failed to read embedded skill file: %v", err)
	}

	if len(content) == 0 {
		t.Error("Embedded skill file should not be empty")
	}

	if !strings.Contains(string(content), "name: memory") {
		t.Error("Embedded skill file should contain 'name: memory'")
	}
}
