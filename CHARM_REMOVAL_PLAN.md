# Memory - Charm Removal Plan

## Charmbracelet Dependencies

**Direct:**
- `github.com/charmbracelet/charm v0.17.0` (replaced with 2389-research fork)

**Indirect (removed with charm):**
- bubbles, bubbletea, keygen, lipgloss, log, x/ansi, x/term

## Files Using Charm

| File | Imports | Purpose |
|------|---------|---------|
| `internal/charm/client.go` | `charm/client`, `charm/kv` | Core KV storage - Get/Set/Delete/List |
| `internal/charm/wal_test.go` | `charm/kv` | WAL concurrency test |
| `cmd/memory/commands/sync.go` | `charm/kv` | Sync status, repair, reset, wipe |

## Data Entities

1. **BridgeBlocks** (key: `block:*`) - Conversations with topics, turns, summaries
2. **Facts** (key: `fact:*`, `fact:bykey:*`) - Extracted facts with confidence
3. **Embeddings** (key: `embedding:*`) - 1536-float vectors
4. **UserProfile** (key: `profile:user`) - Name, preferences, interests

## Removal Strategy

### Phase 1: Create SQLite Storage Backend

New package: `internal/storage/` (standardized across suite)

Use `modernc.org/sqlite` (pure Go, no CGO) for consistency with other tools.

**Schema:**
```sql
CREATE TABLE user_profile (
    id INTEGER PRIMARY KEY CHECK (id = 1),  -- Singleton
    name TEXT,
    preferences TEXT,  -- JSON array
    topics_of_interest TEXT,  -- JSON array
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE bridge_blocks (
    id TEXT PRIMARY KEY,
    day_id TEXT NOT NULL,
    topic_label TEXT,
    keywords TEXT,  -- JSON array
    status TEXT DEFAULT 'ACTIVE',
    summary TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE turns (
    id TEXT PRIMARY KEY,
    block_id TEXT NOT NULL REFERENCES bridge_blocks(id) ON DELETE CASCADE,
    user_message TEXT,
    ai_response TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE facts (
    id TEXT PRIMARY KEY,
    block_id TEXT REFERENCES bridge_blocks(id) ON DELETE SET NULL,
    turn_id TEXT,
    key TEXT NOT NULL,
    value TEXT NOT NULL,
    confidence REAL DEFAULT 1.0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE embeddings (
    id TEXT PRIMARY KEY,
    chunk_id TEXT,
    turn_id TEXT,
    block_id TEXT REFERENCES bridge_blocks(id) ON DELETE CASCADE,
    vector BLOB NOT NULL,  -- 1536 floats as binary
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_blocks_day ON bridge_blocks(day_id);
CREATE INDEX idx_blocks_status ON bridge_blocks(status);
CREATE INDEX idx_turns_block ON turns(block_id);
CREATE INDEX idx_facts_key ON facts(key);
CREATE INDEX idx_facts_block ON facts(block_id);
CREATE INDEX idx_embeddings_block ON embeddings(block_id);
```

### Phase 2: Add Export Commands

**YAML Format:**
```yaml
version: "1.0"
exported_at: "2026-01-31T12:00:00Z"

profile:
  name: "Harper"
  preferences: ["prefers dark mode"]
  topics_of_interest: ["Go programming"]

blocks:
  - block_id: "block_20260131_abc12345"
    topic_label: "Project Planning"
    status: "ACTIVE"
    turns:
      - turn_id: "turn_def456"
        user_message: "Let's design the system"
        ai_response: "Sure, here's the architecture..."

facts:
  - fact_id: "fact_abc123"
    key: "user_name"
    value: "Harper"
    confidence: 1.0

embeddings_file: "embeddings-2026-01-31.json"
```

**Markdown Format:**
```markdown
# Memory Export - 2026-01-31

## User Profile
- **Name:** Harper
- **Topics of Interest:** Go programming

## Facts
| Key | Value | Confidence |
|-----|-------|------------|
| user_name | Harper | 1.00 |

## Conversations

### Project Planning (ACTIVE)
**User:** Let's design the system
**AI:** Sure, here's the architecture...
```

### Phase 3: Migration Tool

`memory migrate charm` - Migrates from Charm KV to local storage.

## Files to Modify

### DELETE:
- `internal/charm/client.go`
- `internal/charm/wal_test.go`

### CREATE:
- `internal/storage/sqlite.go` - SQLite storage implementation
- `internal/storage/schema.go` - Schema and migrations
- `internal/storage/blocks.go` - BridgeBlock CRUD
- `internal/storage/facts.go` - Fact CRUD
- `internal/storage/embeddings.go` - Embedding storage
- `internal/storage/profile.go` - User profile
- `internal/storage/migration.go` - Charm KV migration
- `cmd/memory/commands/export.go` - Export command
- `cmd/memory/commands/import.go` - Import command

### MODIFY:
- `go.mod` - Remove charm, add modernc.org/sqlite, gopkg.in/yaml.v3
- `internal/storage/storage.go` - Update to use SQLite
- `internal/storage/vector_storage.go` - Update to use SQLite
- `cmd/memory/commands/sync.go` - Remove charm commands
- `internal/config/config.go` - Remove CharmHost, add DataDir

## Configuration Changes

**Remove:**
- `CharmHost`
- `CharmDBName`
- `AutoSync`
- `StaleThreshold`

**Add:**
- `DataDir` - Local storage path (`~/.local/share/memory/`)

## Implementation Order

1. Create `internal/storage/` SQLite implementation
2. Add migration command (before removing Charm access)
3. Add export command
4. Add import command
5. Update storage layer to use SQLite
6. Remove Charm sync commands
7. Update configuration
8. Remove Charm from go.mod
9. Delete `internal/charm/`
10. Update documentation

## Data Path

`~/.local/share/memory/memory.db`
