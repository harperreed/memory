# HMLR Go MCP Server - Design Document

## Overview

This is a Go port of the [HMLR (Hierarchical Memory Lookup & Routing)](https://github.com/Sean-V-Dev/HMLR-Agentic-AI-Memory-System) system, exposed as an MCP (Model Context Protocol) server for integration with Claude Code and other MCP clients.

HMLR replaces traditional vector-only RAG systems with a structured, state-aware memory architecture that achieves perfect (1.00) faithfulness and context recall across adversarial benchmarks.

## What HMLR Does

- **Temporal Resolution**: Automatically handles conflicting facts across time (newer information overrides older)
- **Policy Enforcement**: Maintains persistent user constraints across different conversation topics
- **Multi-hop Reasoning**: Performs sophisticated reasoning chains over long-forgotten information
- **Semantic Recall**: Retrieves relevant memories even with zero keyword overlap
- **Long-term Persistence**: Sustains conversation coherence across 50+ turns with 30-day temporal gaps

## Architecture

### Core Components (6 Parallel Engines)

1. **ChunkEngine** - Splits incoming messages into hierarchical chunks (turn → paragraph → sentence) for embedding
2. **Scribe Agent** - Runs in background, updates user profile asynchronously (learns about the user over time)
3. **FactScrubber** - Extracts key-value facts from conversations and stores them in SQLite
4. **LatticeCrawler** - Does vector-based candidate retrieval when needed
5. **Governor** - The smart router that decides "is this a new topic or continuation?" and filters memories
6. **ContextHydrator** - Assembles the final prompt with Bridge Block history + facts + user profile

### Bridge Blocks (Core Memory Unit)

Each Bridge Block represents a topic/conversation thread with:
- `block_id` (unique identifier)
- `topic_label` (what it's about)
- `keywords` (for matching future queries)
- `turns[]` (the actual conversation history)
- `summary` (generated when paused/closed)
- `status`: `ACTIVE`, `PAUSED`, or `CLOSED`

### The 4 Routing Scenarios

The Governor decides what to do with each incoming message:

1. **Topic Continuation** - Same topic as last active block → append turn
2. **Topic Resumption** - Match old paused block → reactivate it, pause current
3. **New Topic (first)** - No active blocks → create new block
4. **Topic Shift** - New topic while one is active → pause old, create new

### Processing Flow

```
User Query arrives
    ↓
┌─────────────────────────────────────────────────────────┐
│ 6 Parallel Tasks:                                       │
│                                                          │
│ 1. ChunkEngine: Chunk message into embeddings          │
│ 2. Scribe: Update user profile (background)            │
│ 3. Governor (3 sub-tasks in parallel):                 │
│    a. Routing: "New topic or continuation?"            │
│    b. Memory retrieval: Vector search for blocks       │
│    c. Fact lookup: Query SQLite for relevant facts     │
└─────────────────────────────────────────────────────────┘
    ↓
Execute 1 of 4 routing scenarios → block_id
    ↓
FactScrubber: Extract facts → save to SQLite
    ↓
ContextHydrator: Build final prompt:
  - System prompt
  - User profile context (top 300 tokens)
  - Bridge Block history (current topic's turns)
  - Retrieved memories (from other blocks)
  - Facts (from SQLite)
  - Current user message
    ↓
Main LLM: Generate response + optional metadata JSON
    ↓
Parse metadata, update Bridge Block header
    ↓
Append turn to Bridge Block, save to disk
```

## Project Structure

```
remember-standalone/
├── cmd/
│   └── server/
│       └── main.go                 # MCP server entry point
├── internal/
│   ├── mcp/
│   │   ├── server.go              # MCP protocol handler
│   │   └── tools.go               # MCP tool definitions
│   ├── core/
│   │   ├── governor.go            # Routing + filtering
│   │   ├── chunk_engine.go        # Hierarchical chunking
│   │   ├── fact_scrubber.go       # Fact extraction
│   │   ├── scribe.go              # User profile updates
│   │   ├── hydrator.go            # Context assembly
│   │   └── crawler.go             # Vector search
│   ├── storage/
│   │   ├── bridge_blocks.go       # Bridge Block CRUD
│   │   ├── facts.go               # SQLite fact store
│   │   ├── embeddings.go          # Vector storage
│   │   └── user_profile.go        # User profile manager
│   └── models/
│       ├── bridge_block.go        # Bridge Block data structures
│       ├── fact.go                # Fact data structures
│       ├── turn.go                # Turn data structures
│       └── profile.go             # User profile structures
├── go.mod
└── go.sum
```

## Storage Layout (XDG)

```
~/.local/share/remember/
├── bridge_blocks/
│   └── 2025-12-06/              # Day-based organization
│       ├── block_*.json         # Individual Bridge Blocks
│       └── day_metadata.json
├── facts.db                      # SQLite database
├── embeddings/
│   └── *.json                    # Vector embeddings (simple file-based)
└── user_profile.json             # Long-term user profile
```

### SQLite Schema (facts.db)

```sql
CREATE TABLE facts (
    fact_id TEXT PRIMARY KEY,
    block_id TEXT NOT NULL,
    turn_id TEXT NOT NULL,
    key TEXT NOT NULL,
    value TEXT NOT NULL,
    confidence REAL DEFAULT 1.0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_facts_block ON facts(block_id);
CREATE INDEX idx_facts_key ON facts(key);
```

## MCP Tools

The server exposes these MCP tools:

### 1. `store_conversation`

Store a conversation turn in HMLR memory system.

**Input:**
```json
{
  "message": "User message to store",
  "context": "Optional additional context"
}
```

**Output:**
```json
{
  "block_id": "block_20251206_143022",
  "turn_id": "turn_20251206_143022_abc123",
  "routing_scenario": "topic_continuation",
  "facts_extracted": 3
}
```

### 2. `retrieve_memory`

Retrieve relevant memories from HMLR system.

**Input:**
```json
{
  "query": "What was that API key I mentioned?",
  "max_results": 5
}
```

**Output:**
```json
{
  "memories": [
    {
      "block_id": "block_20251205_120033",
      "topic": "API Configuration",
      "relevance_score": 0.95,
      "summary": "Discussed setting up API keys",
      "turns": [...]
    }
  ],
  "facts": [
    {
      "key": "api_key",
      "value": "XYZ789",
      "block_id": "block_20251205_120033",
      "confidence": 1.0
    }
  ]
}
```

### 3. `list_active_topics`

List all active Bridge Block topics.

**Input:** (none)

**Output:**
```json
{
  "topics": [
    {
      "block_id": "block_20251206_143022",
      "topic_label": "Go MCP Development",
      "status": "ACTIVE",
      "turn_count": 12,
      "created_at": "2025-12-06T14:30:22Z"
    }
  ]
}
```

### 4. `get_topic_history`

Get conversation history for a specific topic.

**Input:**
```json
{
  "block_id": "block_20251206_143022"
}
```

**Output:**
```json
{
  "block_id": "block_20251206_143022",
  "topic_label": "Go MCP Development",
  "turns": [
    {
      "turn_id": "turn_20251206_143022_abc123",
      "timestamp": "2025-12-06T14:30:22Z",
      "user_message": "i want to make a GO mcp server",
      "ai_response": "Let me help you with that..."
    }
  ],
  "summary": "Discussion about building HMLR Go MCP server"
}
```

### 5. `get_user_profile`

Get the user profile summary.

**Input:** (none)

**Output:**
```json
{
  "profile": {
    "name": "Doctor Biz",
    "preferences": {
      "language": "Go",
      "framework_preference": "simple_over_complex"
    },
    "topics_of_interest": ["MCP", "AI Memory Systems", "Go Development"],
    "last_updated": "2025-12-06T14:30:22Z"
  }
}
```

## Dependencies

```go
require (
    // MCP Protocol
    github.com/mark3labs/mcp-go v0.x.x

    // Storage
    github.com/mattn/go-sqlite3 v1.14.x
    github.com/adrg/xdg v0.5.x

    // LLM Integration
    github.com/sashabaranov/go-openai v1.x.x

    // Utilities
    github.com/google/uuid v1.6.x
)
```

## Key Design Decisions

1. **Embeddings**: Use OpenAI's `text-embedding-3-small` (same as Python HMLR)

2. **Vector Store**: Start with simple JSON files (like Python version), can upgrade to Qdrant/Chroma later

3. **Facts DB**: SQLite for efficient key-value fact queries

4. **Concurrency**: Use goroutines for:
   - Scribe (background user profile updates)
   - FactScrubber (parallel fact extraction)
   - Governor sub-tasks (routing, memory retrieval, fact lookup in parallel)

5. **XDG Compliance**: Store all data in `~/.local/share/remember/` for proper Linux/macOS integration

## Implementation Phases

### Phase 1: Foundation
- [ ] Project setup (go.mod, directory structure)
- [ ] XDG storage initialization
- [ ] Basic MCP server with stdio transport
- [ ] Bridge Block storage (JSON files)
- [ ] SQLite facts database setup

### Phase 2: Core Components
- [x] ChunkEngine (hierarchical chunking)
- [x] Governor (routing logic + 4 scenarios)
- [x] FactScrubber (fact extraction)
- [x] ContextHydrator (prompt assembly)
- [x] LatticeCrawler (vector search)

### Phase 3: Background Agents
- [ ] Scribe agent (user profile updates)
- [ ] Goroutine orchestration

### Phase 4: MCP Tools
- [ ] store_conversation
- [ ] retrieve_memory
- [ ] list_active_topics
- [ ] get_topic_history
- [ ] get_user_profile

### Phase 5: Testing & Polish
- [ ] Unit tests for each component
- [ ] Integration tests with Claude Code
- [ ] Performance optimization
- [ ] Documentation

## Testing Strategy

1. **Unit Tests**: Test each component in isolation
2. **Integration Tests**: Test full flow with mock LLM
3. **RAGAS Benchmarks**: Port Python HMLR test suite to validate 1.00 faithfulness/recall
4. **MCP Integration Tests**: Test with actual Claude Code client

## Success Metrics

- Achieves same 1.00 faithfulness/recall as Python HMLR
- <100ms response time for memory retrieval
- Handles 50+ turn conversations with 30-day gaps
- Seamless integration with Claude Code via MCP

## References

- [HMLR Python Implementation](https://github.com/Sean-V-Dev/HMLR-Agentic-AI-Memory-System)
- [MCP Specification](https://modelcontextprotocol.io)
- [Go MCP SDK](https://github.com/mark3labs/mcp-go)
