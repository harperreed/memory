// ABOUTME: Tests for centralized configuration system
// ABOUTME: Verifies environment variable parsing and validation
package config

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestLoad_Defaults(t *testing.T) {
	// Clear environment to test defaults
	os.Clearenv()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Verify defaults
	if !strings.Contains(cfg.DataDir, "memory") {
		t.Errorf("DataDir = %s, expected to contain 'memory'", cfg.DataDir)
	}
	if cfg.ChatModel != "gpt-4o-mini" {
		t.Errorf("ChatModel = %s, want gpt-4o-mini", cfg.ChatModel)
	}
	if cfg.EmbeddingModel != "text-embedding-3-small" {
		t.Errorf("EmbeddingModel = %s, want text-embedding-3-small", cfg.EmbeddingModel)
	}
	if cfg.Timeout != 30*time.Second {
		t.Errorf("Timeout = %v, want 30s", cfg.Timeout)
	}
	if cfg.MaxRetries != 3 {
		t.Errorf("MaxRetries = %d, want 3", cfg.MaxRetries)
	}
	if cfg.RetryDelay != 2*time.Second {
		t.Errorf("RetryDelay = %v, want 2s", cfg.RetryDelay)
	}
	if cfg.TopicMatchThreshold != 0.3 {
		t.Errorf("TopicMatchThreshold = %f, want 0.3", cfg.TopicMatchThreshold)
	}
	if cfg.VectorDimension != 1536 {
		t.Errorf("VectorDimension = %d, want 1536", cfg.VectorDimension)
	}
}

func TestLoad_CustomValues(t *testing.T) {
	// Set custom environment variables
	os.Clearenv()
	_ = os.Setenv("MEMORY_DATA_DIR", "/tmp/memory-test")
	_ = os.Setenv("OPENAI_API_KEY", "test-key")
	_ = os.Setenv("MEMORY_OPENAI_MODEL", "gpt-4")
	_ = os.Setenv("MEMORY_EMBEDDING_MODEL", "text-embedding-3-large")
	_ = os.Setenv("OPENAI_TIMEOUT", "60s")
	_ = os.Setenv("OPENAI_MAX_RETRIES", "5")
	_ = os.Setenv("OPENAI_RETRY_DELAY", "3s")
	_ = os.Setenv("TOPIC_MATCH_THRESHOLD", "0.5")
	_ = os.Setenv("VECTOR_DIMENSION", "3072")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Verify custom values
	if cfg.DataDir != "/tmp/memory-test" {
		t.Errorf("DataDir = %s, want /tmp/memory-test", cfg.DataDir)
	}
	if cfg.OpenAIKey != "test-key" {
		t.Errorf("OpenAIKey = %s, want test-key", cfg.OpenAIKey)
	}
	if cfg.ChatModel != "gpt-4" {
		t.Errorf("ChatModel = %s, want gpt-4", cfg.ChatModel)
	}
	if cfg.EmbeddingModel != "text-embedding-3-large" {
		t.Errorf("EmbeddingModel = %s, want text-embedding-3-large", cfg.EmbeddingModel)
	}
	if cfg.Timeout != 60*time.Second {
		t.Errorf("Timeout = %v, want 60s", cfg.Timeout)
	}
	if cfg.MaxRetries != 5 {
		t.Errorf("MaxRetries = %d, want 5", cfg.MaxRetries)
	}
	if cfg.RetryDelay != 3*time.Second {
		t.Errorf("RetryDelay = %v, want 3s", cfg.RetryDelay)
	}
	if cfg.TopicMatchThreshold != 0.5 {
		t.Errorf("TopicMatchThreshold = %f, want 0.5", cfg.TopicMatchThreshold)
	}
	if cfg.VectorDimension != 3072 {
		t.Errorf("VectorDimension = %d, want 3072", cfg.VectorDimension)
	}
}

func TestValidate_InvalidThreshold(t *testing.T) {
	cfg := &Config{
		TopicMatchThreshold: 1.5,
		MaxRetries:          3,
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Validate() should fail for threshold > 1")
	}

	cfg.TopicMatchThreshold = -0.1
	err = cfg.Validate()
	if err == nil {
		t.Error("Validate() should fail for threshold < 0")
	}
}

func TestValidate_InvalidMaxRetries(t *testing.T) {
	cfg := &Config{
		TopicMatchThreshold: 0.5,
		MaxRetries:          15,
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Validate() should fail for MaxRetries > 10")
	}

	cfg.MaxRetries = -1
	err = cfg.Validate()
	if err == nil {
		t.Error("Validate() should fail for MaxRetries < 0")
	}
}

func TestDBPath(t *testing.T) {
	cfg := &Config{
		DataDir: "/tmp/memory-test",
	}

	expected := "/tmp/memory-test/memory.db"
	if cfg.DBPath() != expected {
		t.Errorf("DBPath() = %s, want %s", cfg.DBPath(), expected)
	}
}

func TestDefaultDataDir(t *testing.T) {
	dir := DefaultDataDir()
	if dir == "" {
		t.Error("DefaultDataDir() returned empty string")
	}
	if !strings.Contains(dir, "memory") {
		t.Errorf("DefaultDataDir() = %s, expected to contain 'memory'", dir)
	}
}
