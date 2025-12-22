// ABOUTME: Centralized configuration for the memory MCP server
// ABOUTME: Loads from environment variables with validation and defaults
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all configuration for the memory system
type Config struct {
	// Charm settings
	CharmHost   string
	CharmDBName string
	AutoSync    bool

	// OpenAI settings
	OpenAIKey      string
	ChatModel      string
	EmbeddingModel string
	Timeout        time.Duration
	MaxRetries     int
	RetryDelay     time.Duration

	// Memory settings
	TopicMatchThreshold float64
	VectorDimension     int
}

// Load reads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{
		// Defaults
		CharmHost:           getEnv("CHARM_HOST", "cloud.charm.sh"),
		CharmDBName:         getEnv("CHARM_DB", "memory"),
		AutoSync:            getEnvBool("CHARM_AUTO_SYNC", true),
		OpenAIKey:           os.Getenv("OPENAI_API_KEY"),
		ChatModel:           getEnv("MEMORY_OPENAI_MODEL", "gpt-4o-mini"),
		EmbeddingModel:      getEnv("MEMORY_EMBEDDING_MODEL", "text-embedding-3-small"),
		Timeout:             getEnvDuration("OPENAI_TIMEOUT", 30*time.Second),
		MaxRetries:          getEnvInt("OPENAI_MAX_RETRIES", 3),
		RetryDelay:          getEnvDuration("OPENAI_RETRY_DELAY", 2*time.Second),
		TopicMatchThreshold: getEnvFloat("TOPIC_MATCH_THRESHOLD", 0.3),
		VectorDimension:     getEnvInt("VECTOR_DIMENSION", 1536),
	}

	return cfg, cfg.Validate()
}

func (c *Config) Validate() error {
	if c.TopicMatchThreshold < 0 || c.TopicMatchThreshold > 1 {
		return fmt.Errorf("TOPIC_MATCH_THRESHOLD must be 0-1, got %f", c.TopicMatchThreshold)
	}
	if c.MaxRetries < 0 || c.MaxRetries > 10 {
		return fmt.Errorf("OPENAI_MAX_RETRIES must be 0-10, got %d", c.MaxRetries)
	}
	return nil
}

// Helper functions
func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func getEnvBool(key string, defaultVal bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}
	return v == "true" || v == "1"
}

func getEnvInt(key string, defaultVal int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return defaultVal
}

func getEnvFloat(key string, defaultVal float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return defaultVal
}

func getEnvDuration(key string, defaultVal time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return defaultVal
}
