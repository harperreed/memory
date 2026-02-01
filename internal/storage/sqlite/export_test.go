// ABOUTME: Tests for export functionality
// ABOUTME: Verifies YAML, Markdown, and JSON export formats
package sqlite

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/harper/remember-standalone/internal/models"
	"gopkg.in/yaml.v3"
)

func TestExport(t *testing.T) {
	store, err := NewStorageInMemory()
	if err != nil {
		t.Fatalf("NewStorageInMemory() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	// Setup test data
	profile := &models.UserProfile{
		Name:             "Doctor Biz",
		Preferences:      []string{"TDD", "dark mode"},
		TopicsOfInterest: []string{"Go", "SQLite"},
	}
	_ = store.SaveUserProfile(profile)

	turn := &models.Turn{
		TurnID:      "turn_export_1",
		Timestamp:   time.Now(),
		UserMessage: "Hello!",
		AIResponse:  "Hi there!",
		Topics:      []string{"greeting"},
	}
	_, _ = store.StoreTurn(turn)

	fact := &models.Fact{
		FactID:     "fact_export_1",
		Key:        "user_name",
		Value:      "Harper",
		Confidence: 1.0,
		CreatedAt:  time.Now(),
	}
	_ = store.SaveFact(fact)

	// Export
	data, err := store.Export()
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}

	if data.Version != "1.0" {
		t.Errorf("Version = %v, want 1.0", data.Version)
	}
	if data.Tool != "memory" {
		t.Errorf("Tool = %v, want memory", data.Tool)
	}
	if data.Profile == nil {
		t.Fatal("Profile is nil")
	}
	if data.Profile.Name != "Doctor Biz" {
		t.Errorf("Profile.Name = %v, want Doctor Biz", data.Profile.Name)
	}
	if len(data.Blocks) != 1 {
		t.Errorf("Blocks count = %v, want 1", len(data.Blocks))
	}
	if len(data.Facts) != 1 {
		t.Errorf("Facts count = %v, want 1", len(data.Facts))
	}
}

func TestExportToYAML(t *testing.T) {
	store, err := NewStorageInMemory()
	if err != nil {
		t.Fatalf("NewStorageInMemory() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	// Setup minimal data
	profile := &models.UserProfile{Name: "Test User"}
	_ = store.SaveUserProfile(profile)

	// Export to temp file
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "export.yaml")

	err = store.ExportToYAML(outputPath)
	if err != nil {
		t.Fatalf("ExportToYAML() error = %v", err)
	}

	// Verify file exists and is valid YAML
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	var data ExportData
	if err := yaml.Unmarshal(content, &data); err != nil {
		t.Fatalf("Failed to parse YAML: %v", err)
	}

	if data.Profile.Name != "Test User" {
		t.Errorf("Profile.Name = %v, want Test User", data.Profile.Name)
	}
}

func TestExportToMarkdown(t *testing.T) {
	store, err := NewStorageInMemory()
	if err != nil {
		t.Fatalf("NewStorageInMemory() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	// Setup data
	profile := &models.UserProfile{
		Name:        "Test User",
		Preferences: []string{"vim"},
	}
	_ = store.SaveUserProfile(profile)

	fact := &models.Fact{
		FactID:     "fact_md_1",
		Key:        "editor",
		Value:      "vim",
		Confidence: 1.0,
		CreatedAt:  time.Now(),
	}
	_ = store.SaveFact(fact)

	turn := &models.Turn{
		TurnID:      "turn_md_1",
		Timestamp:   time.Now(),
		UserMessage: "Hello",
		AIResponse:  "Hi!",
		Topics:      []string{"greeting"},
	}
	_, _ = store.StoreTurn(turn)

	// Export to temp file
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "export.md")

	err = store.ExportToMarkdown(outputPath)
	if err != nil {
		t.Fatalf("ExportToMarkdown() error = %v", err)
	}

	// Verify file exists and has expected content
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	contentStr := string(content)

	// Check for expected sections
	if !strings.Contains(contentStr, "# Memory Export") {
		t.Error("Missing Memory Export header")
	}
	if !strings.Contains(contentStr, "## User Profile") {
		t.Error("Missing User Profile section")
	}
	if !strings.Contains(contentStr, "Test User") {
		t.Error("Missing user name")
	}
	if !strings.Contains(contentStr, "## Facts") {
		t.Error("Missing Facts section")
	}
	if !strings.Contains(contentStr, "editor") {
		t.Error("Missing fact key")
	}
	if !strings.Contains(contentStr, "## Conversations") {
		t.Error("Missing Conversations section")
	}
}

func TestExportEmptyDatabase(t *testing.T) {
	store, err := NewStorageInMemory()
	if err != nil {
		t.Fatalf("NewStorageInMemory() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	data, err := store.Export()
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}

	if data.Profile != nil {
		t.Error("Expected nil profile for empty database")
	}
	if len(data.Blocks) != 0 {
		t.Errorf("Expected 0 blocks, got %d", len(data.Blocks))
	}
	if len(data.Facts) != 0 {
		t.Errorf("Expected 0 facts, got %d", len(data.Facts))
	}
}
