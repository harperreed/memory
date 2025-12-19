// ABOUTME: Charm KV client wrapper for cloud-synced storage
// ABOUTME: Replaces SQLite and file-based storage with automatic SSH key auth
package charm

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/charmbracelet/charm/client"
	"github.com/charmbracelet/charm/kv"
)

// Key prefixes for different entity types
const (
	BlockPrefix     = "block:"
	FactPrefix      = "fact:"
	ProfilePrefix   = "profile:"
	EmbeddingPrefix = "embedding:"
)

// Config holds charm client configuration
type Config struct {
	Host     string
	DBName   string
	AutoSync bool
}

// DefaultConfig returns default configuration for charm client
func DefaultConfig() *Config {
	host := os.Getenv("CHARM_HOST")
	if host == "" {
		host = "charm.2389.dev"
	}
	return &Config{
		Host:     host,
		DBName:   "memory",
		AutoSync: true,
	}
}

var (
	globalClient *Client
	clientOnce   sync.Once
	clientErr    error
	clientMu     sync.Mutex
)

// Client wraps charm KV for storage operations
type Client struct {
	kv     *kv.KV
	config *Config
	mu     sync.Mutex
}

// InitClient initializes the global charm client (thread-safe singleton)
func InitClient() error {
	clientOnce.Do(func() {
		globalClient, clientErr = NewClient(DefaultConfig())
	})
	return clientErr
}

// GetClient returns the global client, initializing if needed
func GetClient() (*Client, error) {
	clientMu.Lock()
	defer clientMu.Unlock()

	// If client was closed, reinitialize
	if globalClient != nil && globalClient.kv == nil {
		clientOnce = sync.Once{}
		globalClient = nil
	}

	if err := InitClient(); err != nil {
		return nil, err
	}
	return globalClient, nil
}

// ResetGlobalClient resets the global client (for testing)
func ResetGlobalClient() {
	clientMu.Lock()
	defer clientMu.Unlock()
	if globalClient != nil {
		_ = globalClient.Close()
	}
	clientOnce = sync.Once{}
	globalClient = nil
	clientErr = nil
}

// NewClient creates a new charm client with the given config
func NewClient(cfg *Config) (*Client, error) {
	// Set CHARM_HOST before opening KV
	os.Setenv("CHARM_HOST", cfg.Host)

	db, err := kv.OpenWithDefaults(cfg.DBName)
	if err != nil {
		return nil, fmt.Errorf("failed to open charm kv: %w", err)
	}

	c := &Client{
		kv:     db,
		config: cfg,
	}

	// Pull remote data on startup
	if cfg.AutoSync {
		_ = db.Sync()
	}

	return c, nil
}

// Close closes the KV database
func (c *Client) Close() error {
	if c.kv != nil {
		err := c.kv.Close()
		c.kv = nil // Mark as closed so GetClient knows to reinitialize
		return err
	}
	return nil
}

// syncIfEnabled syncs to cloud after writes
func (c *Client) syncIfEnabled() {
	if c.config.AutoSync {
		_ = c.kv.Sync()
	}
}

// ID returns the charm user ID
func (c *Client) ID() (string, error) {
	cc, err := client.NewClientWithDefaults()
	if err != nil {
		return "", fmt.Errorf("failed to create charm client: %w", err)
	}
	return cc.ID()
}

// Set stores a value with the given key
func (c *Client) Set(key string, value []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.kv.Set([]byte(key), value); err != nil {
		return fmt.Errorf("failed to set key %s: %w", key, err)
	}
	c.syncIfEnabled()
	return nil
}

// Get retrieves a value by key
func (c *Client) Get(key string) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.kv.Get([]byte(key))
}

// Delete removes a key
func (c *Client) Delete(key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.kv.Delete([]byte(key)); err != nil {
		return fmt.Errorf("failed to delete key %s: %w", key, err)
	}
	c.syncIfEnabled()
	return nil
}

// SetJSON marshals and stores a value as JSON
func (c *Client) SetJSON(key string, value interface{}) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}
	return c.Set(key, data)
}

// GetJSON retrieves and unmarshals a JSON value
func (c *Client) GetJSON(key string, dest interface{}) error {
	data, err := c.Get(key)
	if err != nil {
		return err
	}
	if data == nil {
		return fmt.Errorf("key not found: %s", key)
	}
	return json.Unmarshal(data, dest)
}

// ListKeys returns all keys with the given prefix
func (c *Client) ListKeys(prefix string) ([]string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	keys, err := c.kv.Keys()
	if err != nil {
		return nil, fmt.Errorf("failed to list keys: %w", err)
	}

	var result []string
	for _, key := range keys {
		keyStr := string(key)
		if strings.HasPrefix(keyStr, prefix) {
			result = append(result, keyStr)
		}
	}
	return result, nil
}

// Sync manually triggers a sync with the cloud
func (c *Client) Sync() error {
	return c.kv.Sync()
}

// Reset wipes all local data (nuclear option)
func (c *Client) Reset() error {
	return c.kv.Reset()
}

// GetAuthorizedKeys returns the list of linked devices/keys
func (c *Client) GetAuthorizedKeys() (string, error) {
	cc, err := client.NewClientWithDefaults()
	if err != nil {
		return "", fmt.Errorf("failed to create charm client: %w", err)
	}
	return cc.AuthorizedKeys()
}

// UnlinkKey removes an authorized key from the account
func (c *Client) UnlinkKey(key string) error {
	cc, err := client.NewClientWithDefaults()
	if err != nil {
		return fmt.Errorf("failed to create charm client: %w", err)
	}
	return cc.UnlinkAuthorizedKey(key)
}

// BlockKey generates a key for a BridgeBlock
func BlockKey(blockID string) string {
	return BlockPrefix + blockID
}

// FactKey generates a key for a Fact
func FactKey(factID string) string {
	return FactPrefix + factID
}

// FactByKeyKey generates a lookup key for facts by their key field
func FactByKeyKey(factKey string) string {
	return FactPrefix + "bykey:" + factKey
}

// ProfileKey generates a key for UserProfile
func ProfileKey() string {
	return ProfilePrefix + "user"
}

// EmbeddingKey generates a key for an Embedding
func EmbeddingKey(chunkID string) string {
	return EmbeddingPrefix + chunkID
}
