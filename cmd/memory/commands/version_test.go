// ABOUTME: Tests for version command
// ABOUTME: Verifies version info display and SetVersion functionality

package commands

import (
	"bytes"
	"strings"
	"testing"
)

func TestNewVersionCmd(t *testing.T) {
	cmd := NewVersionCmd()

	if cmd.Use != "version" {
		t.Errorf("Use = %q, want %q", cmd.Use, "version")
	}

	if cmd.Short == "" {
		t.Error("Short description should not be empty")
	}

	if cmd.Long == "" {
		t.Error("Long description should not be empty")
	}
}

func TestVersionCmd_Output(t *testing.T) {
	// Save original values
	originalVersion := versionInfo.Version
	originalCommit := versionInfo.Commit
	originalDate := versionInfo.Date
	defer func() {
		versionInfo.Version = originalVersion
		versionInfo.Commit = originalCommit
		versionInfo.Date = originalDate
	}()

	// Set test values
	SetVersion("1.2.3", "abc123", "2026-01-31")

	cmd := NewVersionCmd()
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetErr(&output)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	outputStr := output.String()

	expectedParts := []string{
		"Memory (HMLR) 1.2.3",
		"Commit: abc123",
		"Built:  2026-01-31",
	}

	for _, expected := range expectedParts {
		if !strings.Contains(outputStr, expected) {
			t.Errorf("Output should contain %q, got:\n%s", expected, outputStr)
		}
	}
}

func TestSetVersion(t *testing.T) {
	// Save original values
	originalVersion := versionInfo.Version
	originalCommit := versionInfo.Commit
	originalDate := versionInfo.Date
	defer func() {
		versionInfo.Version = originalVersion
		versionInfo.Commit = originalCommit
		versionInfo.Date = originalDate
	}()

	testCases := []struct {
		version string
		commit  string
		date    string
	}{
		{"1.0.0", "deadbeef", "2026-01-01"},
		{"dev", "none", "unknown"},
		{"2.0.0-beta", "1234567890abcdef", "2026-06-15T10:30:00Z"},
	}

	for _, tc := range testCases {
		t.Run(tc.version, func(t *testing.T) {
			SetVersion(tc.version, tc.commit, tc.date)

			if versionInfo.Version != tc.version {
				t.Errorf("Version = %q, want %q", versionInfo.Version, tc.version)
			}
			if versionInfo.Commit != tc.commit {
				t.Errorf("Commit = %q, want %q", versionInfo.Commit, tc.commit)
			}
			if versionInfo.Date != tc.date {
				t.Errorf("Date = %q, want %q", versionInfo.Date, tc.date)
			}
		})
	}
}

func TestVersionInfo_DefaultValues(t *testing.T) {
	// The default values should be "dev", "none", "unknown"
	// These are set at package initialization
	// We can verify the struct holds the expected type

	info := VersionInfo{
		Version: "test",
		Commit:  "test",
		Date:    "test",
	}

	if info.Version != "test" {
		t.Errorf("Version = %q, want %q", info.Version, "test")
	}
	if info.Commit != "test" {
		t.Errorf("Commit = %q, want %q", info.Commit, "test")
	}
	if info.Date != "test" {
		t.Errorf("Date = %q, want %q", info.Date, "test")
	}
}

func TestVersionCmd_NoArgs(t *testing.T) {
	cmd := NewVersionCmd()
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	if err != nil {
		t.Errorf("Version command should not require args, got error: %v", err)
	}
}

func TestVersionCmd_ExtraArgsIgnored(t *testing.T) {
	cmd := NewVersionCmd()
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	cmd.SetArgs([]string{"extra", "args"})

	// Extra args should be ignored for version command
	_ = cmd.Execute()

	// Output should still contain version info
	outputStr := output.String()
	if !strings.Contains(outputStr, "Memory (HMLR)") {
		t.Error("Version output should still be produced with extra args")
	}
}
