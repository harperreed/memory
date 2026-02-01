// ABOUTME: Tests for SQLite database connection and schema initialization
// ABOUTME: Verifies database creation, schema, and basic operations
package sqlite

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpenInMemory(t *testing.T) {
	db, err := OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory() error = %v", err)
	}
	defer func() { _ = db.Close() }()

	if db.Conn() == nil {
		t.Error("Conn() should not be nil")
	}

	if db.Path() != ":memory:" {
		t.Errorf("Path() = %v, want :memory:", db.Path())
	}
}

func TestSchemaInitialization(t *testing.T) {
	db, err := OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory() error = %v", err)
	}
	defer func() { _ = db.Close() }()

	// Verify all tables exist
	tables := []string{"user_profile", "bridge_blocks", "turns", "facts", "embeddings"}
	for _, table := range tables {
		var name string
		err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&name)
		if err != nil {
			t.Errorf("Table %s does not exist: %v", table, err)
		}
	}
}

func TestOpenCreatesDirectory(t *testing.T) {
	// Use a temp directory
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "subdir", "nested", "memory.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer func() { _ = db.Close() }()

	// Verify file was created
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("Database file was not created")
	}
}

func TestDefaultDataDir(t *testing.T) {
	dir := DefaultDataDir()
	if dir == "" {
		t.Error("DefaultDataDir() returned empty string")
	}
	// Should contain .local/share/memory
	if !filepath.IsAbs(dir) && dir != ".local/share/memory" {
		t.Errorf("DefaultDataDir() = %v, expected absolute path or fallback", dir)
	}
}

func TestDefaultDBPath(t *testing.T) {
	path := DefaultDBPath()
	if path == "" {
		t.Error("DefaultDBPath() returned empty string")
	}
	// Should end with memory.db
	if filepath.Base(path) != "memory.db" {
		t.Errorf("DefaultDBPath() = %v, should end with memory.db", path)
	}
}

func TestCloseMultipleTimes(t *testing.T) {
	db, err := OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory() error = %v", err)
	}

	// First close should succeed
	if err := db.Close(); err != nil {
		t.Errorf("First Close() error = %v", err)
	}

	// Second close should be safe (conn is closed but shouldn't panic)
	// Note: This may return an error which is acceptable
	_ = db.Close()
}

func TestForeignKeysEnabled(t *testing.T) {
	db, err := OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory() error = %v", err)
	}
	defer func() { _ = db.Close() }()

	var fkEnabled int
	err = db.QueryRow("PRAGMA foreign_keys").Scan(&fkEnabled)
	if err != nil {
		t.Fatalf("Failed to check foreign_keys pragma: %v", err)
	}

	if fkEnabled != 1 {
		t.Error("Foreign keys are not enabled")
	}
}

func TestIndexesExist(t *testing.T) {
	db, err := OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory() error = %v", err)
	}
	defer func() { _ = db.Close() }()

	// Check for expected indexes
	indexes := []string{
		"idx_blocks_day",
		"idx_blocks_status",
		"idx_turns_block",
		"idx_facts_key",
		"idx_facts_block",
		"idx_embeddings_block",
		"idx_embeddings_chunk",
	}

	for _, idx := range indexes {
		var name string
		err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='index' AND name=?", idx).Scan(&name)
		if err != nil {
			t.Errorf("Index %s does not exist: %v", idx, err)
		}
	}
}
