// ABOUTME: Main storage implementation for HMLR memory system
// ABOUTME: Uses local SQLite for storage (replaces Charm KV)
package storage

import (
	"github.com/harper/remember-standalone/internal/models"
	"github.com/harper/remember-standalone/internal/storage/sqlite"
)

// Storage manages all persistent data for HMLR using SQLite
// This is a thin wrapper around sqlite.Storage for backward compatibility
type Storage = sqlite.Storage

// BridgeBlockInfo contains summary information about a Bridge Block
type BridgeBlockInfo = sqlite.BridgeBlockInfo

// VectorStorage handles embedding storage and similarity search
// This is a thin wrapper around sqlite.EmbeddingStore for backward compatibility
type VectorStorage = sqlite.EmbeddingStore

// ExpectedEmbeddingDimension is the expected dimension for OpenAI embeddings
const ExpectedEmbeddingDimension = sqlite.ExpectedDimension

// NewStorage initializes storage with SQLite backend
func NewStorage() (*Storage, error) {
	return sqlite.NewStorage()
}

// NewStorageWithPath initializes storage with a custom database path
func NewStorageWithPath(dbPath string) (*Storage, error) {
	return sqlite.NewStorageWithPath(dbPath)
}

// NewStorageInMemory creates an in-memory storage (for testing)
func NewStorageInMemory() (*Storage, error) {
	return sqlite.NewStorageInMemory()
}

// NewVectorStorage creates a new VectorStorage (embedding store)
// For compatibility - in SQLite mode, this is obtained via Storage.GetVectorStorage()
func NewVectorStorage(db *sqlite.DB) (*VectorStorage, error) {
	return sqlite.NewEmbeddingStore(db), nil
}

// SkipDimensionValidation can be set to true in tests to allow non-1536D vectors
var SkipDimensionValidation = false

// ExportData represents the complete exportable data structure
type ExportData = sqlite.ExportData

// ExportProfile represents user profile for export
type ExportProfile = sqlite.ExportProfile

// ExportBlock represents a bridge block for export
type ExportBlock = sqlite.ExportBlock

// ExportTurn represents a turn for export
type ExportTurn = sqlite.ExportTurn

// ExportFact represents a fact for export
type ExportFact = sqlite.ExportFact

// Helper function for tests that need to work with turns from blocks
func GetTurnsFromBlock(block *models.BridgeBlock) []models.Turn {
	return block.Turns
}

// DefaultDBPath returns the default database path
func DefaultDBPath() string {
	return sqlite.DefaultDBPath()
}
