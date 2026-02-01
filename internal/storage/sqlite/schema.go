// ABOUTME: SQLite database schema for memory storage
// ABOUTME: Creates all tables, indexes, and migrations for local storage
package sqlite

// Schema contains all SQL statements for database initialization
const Schema = `
-- User profile singleton table
CREATE TABLE IF NOT EXISTS user_profile (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    name TEXT,
    preferences TEXT,
    topics_of_interest TEXT,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Bridge blocks table (conversation threads)
CREATE TABLE IF NOT EXISTS bridge_blocks (
    id TEXT PRIMARY KEY,
    day_id TEXT NOT NULL,
    topic_label TEXT,
    keywords TEXT,
    status TEXT DEFAULT 'ACTIVE',
    summary TEXT,
    turn_count INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Turns table (individual conversation exchanges)
CREATE TABLE IF NOT EXISTS turns (
    id TEXT PRIMARY KEY,
    block_id TEXT NOT NULL REFERENCES bridge_blocks(id) ON DELETE CASCADE,
    user_message TEXT,
    ai_response TEXT,
    keywords TEXT,
    topics TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Facts table (extracted key-value pairs)
CREATE TABLE IF NOT EXISTS facts (
    id TEXT PRIMARY KEY,
    block_id TEXT REFERENCES bridge_blocks(id) ON DELETE SET NULL,
    turn_id TEXT,
    key TEXT NOT NULL,
    value TEXT NOT NULL,
    confidence REAL DEFAULT 1.0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Embeddings table (vector storage)
CREATE TABLE IF NOT EXISTS embeddings (
    id TEXT PRIMARY KEY,
    chunk_id TEXT,
    turn_id TEXT,
    block_id TEXT REFERENCES bridge_blocks(id) ON DELETE CASCADE,
    vector BLOB NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_blocks_day ON bridge_blocks(day_id);
CREATE INDEX IF NOT EXISTS idx_blocks_status ON bridge_blocks(status);
CREATE INDEX IF NOT EXISTS idx_turns_block ON turns(block_id);
CREATE INDEX IF NOT EXISTS idx_facts_key ON facts(key);
CREATE INDEX IF NOT EXISTS idx_facts_block ON facts(block_id);
CREATE INDEX IF NOT EXISTS idx_embeddings_block ON embeddings(block_id);
CREATE INDEX IF NOT EXISTS idx_embeddings_chunk ON embeddings(chunk_id);
`

// SchemaVersion is the current schema version for migrations
const SchemaVersion = 1
