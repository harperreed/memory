// ABOUTME: Charm KV client wrapper using transactional Do API
// ABOUTME: Short-lived connections to avoid lock contention with other MCP servers

package charm

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/charm/client"
	"github.com/charmbracelet/charm/kv"
	"github.com/harper/remember-standalone/internal/config"
)

// Key prefixes for different entity types
const (
	BlockPrefix     = "block:"
	FactPrefix      = "fact:"
	ProfilePrefix   = "profile:"
	EmbeddingPrefix = "embedding:"
)

// Client holds configuration for KV operations.
// Unlike the previous implementation, it does NOT hold a persistent connection.
// Each operation opens the database, performs the operation, and closes it.
type Client struct {
	dbName         string
	autoSync       bool
	staleThreshold time.Duration
}

// NewClient creates a new charm client with the given config.
// Configuration is loaded but no persistent connection is established.
func NewClient(cfg *config.Config) (*Client, error) {
	// Set CHARM_HOST before any KV operations
	if cfg.CharmHost != "" {
		if err := os.Setenv("CHARM_HOST", cfg.CharmHost); err != nil {
			return nil, fmt.Errorf("failed to set CHARM_HOST: %w", err)
		}
	}

	return &Client{
		dbName:         cfg.CharmDBName,
		autoSync:       cfg.AutoSync,
		staleThreshold: cfg.StaleThreshold,
	}, nil
}

// Get retrieves a value by key (read-only, no lock contention).
func (c *Client) Get(key string) ([]byte, error) {
	if err := c.SyncIfStale(); err != nil {
		return nil, fmt.Errorf("failed to sync before read: %w", err)
	}

	var val []byte
	err := kv.DoReadOnly(c.dbName, func(k *kv.KV) error {
		var err error
		val, err = k.Get([]byte(key))
		return err
	})
	return val, err
}

// Set stores a value with the given key.
func (c *Client) Set(key string, value []byte) error {
	return kv.Do(c.dbName, func(k *kv.KV) error {
		if err := k.Set([]byte(key), value); err != nil {
			return fmt.Errorf("failed to set key %s: %w", key, err)
		}
		if c.autoSync {
			return k.Sync()
		}
		return nil
	})
}

// Delete removes a key.
func (c *Client) Delete(key string) error {
	return kv.Do(c.dbName, func(k *kv.KV) error {
		if err := k.Delete([]byte(key)); err != nil {
			return fmt.Errorf("failed to delete key %s: %w", key, err)
		}
		if c.autoSync {
			return k.Sync()
		}
		return nil
	})
}

// SetJSON marshals and stores a value as JSON.
func (c *Client) SetJSON(key string, value interface{}) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}
	return c.Set(key, data)
}

// GetJSON retrieves and unmarshals a JSON value.
func (c *Client) GetJSON(key string, dest interface{}) error {
	// SyncIfStale is called by Get, no need to call it here
	data, err := c.Get(key)
	if err != nil {
		return err
	}
	if data == nil {
		return fmt.Errorf("key not found: %s", key)
	}
	return json.Unmarshal(data, dest)
}

// ListKeys returns all keys with the given prefix.
func (c *Client) ListKeys(prefix string) ([]string, error) {
	if err := c.SyncIfStale(); err != nil {
		return nil, fmt.Errorf("failed to sync before read: %w", err)
	}

	var result []string
	err := kv.DoReadOnly(c.dbName, func(k *kv.KV) error {
		keys, err := k.Keys()
		if err != nil {
			return fmt.Errorf("failed to list keys: %w", err)
		}
		for _, key := range keys {
			keyStr := string(key)
			if strings.HasPrefix(keyStr, prefix) {
				result = append(result, keyStr)
			}
		}
		return nil
	})
	return result, err
}

// DoReadOnly executes a function with read-only database access.
// Use this for batch read operations that need multiple Gets.
func (c *Client) DoReadOnly(fn func(k *kv.KV) error) error {
	if err := c.SyncIfStale(); err != nil {
		return fmt.Errorf("failed to sync before read: %w", err)
	}
	return kv.DoReadOnly(c.dbName, fn)
}

// Do executes a function with write access to the database.
// Use this for batch write operations.
func (c *Client) Do(fn func(k *kv.KV) error) error {
	return kv.Do(c.dbName, func(k *kv.KV) error {
		if err := fn(k); err != nil {
			return err
		}
		if c.autoSync {
			return k.Sync()
		}
		return nil
	})
}

// Sync triggers a manual sync with the charm server.
func (c *Client) Sync() error {
	return kv.Do(c.dbName, func(k *kv.KV) error {
		return k.Sync()
	})
}

// LastSyncTime returns the last time the database was synced with the server.
func (c *Client) LastSyncTime() (time.Time, error) {
	var lastSync time.Time
	err := kv.DoReadOnly(c.dbName, func(k *kv.KV) error {
		lastSync = k.LastSyncTime()
		return nil
	})
	return lastSync, err
}

// IsStale checks if the database needs syncing based on staleThreshold.
func (c *Client) IsStale() (bool, error) {
	var stale bool
	err := kv.DoReadOnly(c.dbName, func(k *kv.KV) error {
		stale = k.IsStale(c.staleThreshold)
		return nil
	})
	return stale, err
}

// SyncIfStale syncs the database if it's considered stale.
func (c *Client) SyncIfStale() error {
	stale, err := c.IsStale()
	if err != nil {
		return fmt.Errorf("failed to check staleness: %w", err)
	}
	if !stale {
		return nil
	}
	return c.Sync()
}

// Reset clears all data (nuclear option).
func (c *Client) Reset() error {
	return kv.Do(c.dbName, func(k *kv.KV) error {
		return k.Reset()
	})
}

// ID returns the charm user ID for this device.
func (c *Client) ID() (string, error) {
	cc, err := client.NewClientWithDefaults()
	if err != nil {
		return "", fmt.Errorf("failed to create charm client: %w", err)
	}
	return cc.ID()
}

// GetAuthorizedKeys returns the list of linked devices/keys.
func (c *Client) GetAuthorizedKeys() (string, error) {
	cc, err := client.NewClientWithDefaults()
	if err != nil {
		return "", fmt.Errorf("failed to create charm client: %w", err)
	}
	return cc.AuthorizedKeys()
}

// UnlinkKey removes an authorized key from the account.
func (c *Client) UnlinkKey(key string) error {
	cc, err := client.NewClientWithDefaults()
	if err != nil {
		return fmt.Errorf("failed to create charm client: %w", err)
	}
	return cc.UnlinkAuthorizedKey(key)
}

// --- Legacy compatibility layer ---
// These functions maintain backwards compatibility with existing code.

var globalClient *Client

// GetClient returns the global client, initializing if needed.
func GetClient() (*Client, error) {
	if globalClient != nil {
		return globalClient, nil
	}
	return InitClient()
}

// InitClient initializes the global charm client.
// With the new architecture, this just creates a Client instance.
func InitClient() (*Client, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	globalClient, err = NewClient(cfg)
	if err != nil {
		return nil, err
	}
	return globalClient, nil
}

// ResetGlobalClient resets the global client singleton (for testing).
func ResetGlobalClient() {
	globalClient = nil
}

// Close is a no-op for backwards compatibility.
// With Do API, connections are automatically closed after each operation.
func (c *Client) Close() error {
	return nil
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
