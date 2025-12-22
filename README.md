# HMLR Go MCP Server

A Go implementation of the [HMLR (Hierarchical Memory Lookup & Routing)](https://github.com/Sean-V-Dev/HMLR-Agentic-AI-Memory-System) memory system, exposed as an MCP (Model Context Protocol) server for integration with Claude Code and other MCP clients.

## What is HMLR?

HMLR replaces traditional vector-only RAG systems with a structured, state-aware memory architecture that achieves **perfect (1.00) faithfulness and context recall** across adversarial benchmarks.

**Key Features:**
- 🧠 **Temporal Resolution**: Automatically handles conflicting facts across time
- 📋 **Bridge Blocks**: Organizes conversations by topic with smart routing
- 🎯 **4 Routing Scenarios**: Continuation, Resumption, New Topic, Topic Shift
- 🔍 **Semantic Recall**: Retrieves relevant memories even with zero keyword overlap
- 💾 **XDG Compliant**: Stores data in `~/.local/share/remember/`

## Quick Start

### Prerequisites

- Go 1.23+ installed
- OpenAI API key (for embeddings and LLM features)

### Installation

1. **Clone the repository:**
```bash
git clone <your-repo-url>
cd remember-standalone
```

2. **Create `.env` file with your API keys:**
```bash
cp .env.example .env
# Edit .env and add your OPENAI_API_KEY
```

Example `.env`:
```bash
OPENAI_API_KEY=sk-proj-...
ANTHROPIC_API_KEY=sk-ant-...  # Optional, for future features
MEMORY_OPENAI_MODEL=gpt-4o-mini  # Optional, defaults to gpt-4o-mini
```

3. **Build the server:**
```bash
go build -o bin/hmlr-server ./cmd/server
```

4. **Run the server:**
```bash
./bin/hmlr-server
```

The server will start on stdio and wait for MCP protocol messages.

## Configuration

### Environment Variables

**Required:**
- `OPENAI_API_KEY` - Your OpenAI API key for embeddings and LLM features

**Optional:**
- `MEMORY_OPENAI_MODEL` - Chat model to use (default: `gpt-4o-mini`)
  - Options: `gpt-4o-mini`, `gpt-4o`, `o1-mini`, etc.
  - Affects metadata extraction, fact extraction, and user profile learning
  - gpt-4o-mini is recommended for best balance of speed and quality

**Model Selection Guide:**
- `gpt-4o-mini`: **Recommended** - Good balance of speed, quality, and cost (~$0.15/1M input tokens)
- `gpt-4o`: Highest quality, slower, more expensive (~$2.50/1M input tokens)
- `o1-mini`: Advanced reasoning capabilities (~$3/1M input tokens)

## Usage with Claude Code

Add to your Claude Code MCP settings (`~/.config/claude-code/mcp_settings.json`):

```json
{
  "mcpServers": {
    "remember": {
      "command": "/path/to/remember-standalone/bin/hmlr-server",
      "args": [],
      "env": {
        "OPENAI_API_KEY": "your-key-here",
        "MEMORY_OPENAI_MODEL": "gpt-4o-mini"
      }
    }
  }
}
```

## MCP Tools

The server exposes 5 MCP tools:

### 1. `store_conversation`
Store a conversation turn in HMLR memory.

**Input:**
```json
{
  "message": "What's the capital of France?",
  "context": "Optional additional context"
}
```

**Output:**
```json
{
  "block_id": "block_20251206_143022",
  "turn_id": "turn_20251206_143022_abc123",
  "routing_scenario": "topic_continuation",
  "facts_extracted": 1
}
```

### 2. `retrieve_memory`
Search for relevant memories.

**Input:**
```json
{
  "query": "What did we discuss about France?",
  "max_results": 5
}
```

**Output:**
```json
{
  "memories": [
    {
      "block_id": "block_20251206_143022",
      "topic_label": "Geography",
      "relevance_score": 0.95,
      "summary": "Discussion about European capitals",
      "turns": [...]
    }
  ],
  "facts": [
    {
      "key": "capital_of_France",
      "value": "Paris",
      "confidence": 1.0
    }
  ]
}
```

### 3. `list_active_topics`
List all active conversation topics.

**Output:**
```json
{
  "topics": [
    {
      "block_id": "block_20251206_143022",
      "topic_label": "Geography",
      "status": "ACTIVE",
      "turn_count": 3,
      "created_at": "2025-12-06T14:30:22Z"
    }
  ]
}
```

### 4. `get_topic_history`
Get full conversation history for a topic.

**Input:**
```json
{
  "block_id": "block_20251206_143022"
}
```

### 5. `get_user_profile`
Get the user profile summary.

**Output:**
```json
{
  "profile": {
    "preferences": {},
    "topics_of_interest": [],
    "last_updated": "2025-12-06T14:30:22Z"
  }
}
```

## Architecture

```
HMLR Memory System
├── Governor         # Smart routing (4 scenarios)
├── ChunkEngine      # Hierarchical chunking (turn → paragraph → sentence)
├── Storage          # XDG-compliant file + SQLite storage
├── Bridge Blocks    # Topic-based conversation organization
└── MCP Server       # Stdio transport for Claude Code integration
```

### Storage Layout

```
~/.local/share/remember/
├── bridge_blocks/
│   └── 2025-12-06/
│       ├── block_*.json         # Conversation topics
│       └── day_metadata.json
├── facts.db                      # SQLite fact database
├── embeddings/
│   └── *.json                    # Vector embeddings
└── user_profile.json             # Long-term user profile
```

## Development

### Running Tests

```bash
# Run all scenario tests (with REAL storage, no mocks!)
go test -v ./.scratch/

# Run specific scenario
go test -v ./.scratch/ -run TestScenario01
```

### Project Structure

```
remember-standalone/
├── cmd/
│   └── server/           # Main entry point
├── internal/
│   ├── core/            # Governor, ChunkEngine
│   ├── storage/         # Storage implementation
│   ├── models/          # Data structures
│   └── mcp/             # MCP tools and handlers
├── .scratch/            # Scenario tests (not committed)
├── scenarios.jsonl      # Documented test scenarios
└── DESIGN.md           # Full architecture design
```

### Scenario Testing

This project follows **scenario-driven testing** with zero mocks:

```bash
# All tests use REAL storage, REAL SQLite, REAL files
.scratch/scenario_01_store_retrieve_test.go  # Basic storage
.scratch/scenario_02_routing_test.go         # Governor routing
.scratch/scenario_03_chunking_test.go        # Hierarchical chunking
```

See `scenarios.jsonl` for documented test scenarios.

## Implementation Status

✅ **Phase 1: Foundation**
- XDG storage initialization
- Bridge Block JSON storage
- SQLite facts database

✅ **Phase 2: Core Components**
- Governor with 4 routing scenarios
- ChunkEngine for hierarchical chunking
- FactScrubber with LLM-based extraction
- ContextHydrator for intelligent prompt assembly
- LatticeCrawler for vector-based candidate retrieval

✅ **Phase 3: Background Agents**
- Scribe agent for async user profile learning
- Goroutine-based async processing

✅ **Phase 4: MCP Server**
- Stdio transport
- All 5 MCP tools implemented
- .env loading for API keys

✅ **Phase 5: Advanced Features**
- Vector embeddings with OpenAI (text-embedding-3-small)
- LLM-based fact extraction (GPT-4o-mini)
- Semantic memory search with cosine similarity
- Dynamic user profiles with intelligent merging

## Roadmap

- [x] Integrate OpenAI embeddings (text-embedding-3-small)
- [x] Implement FactScrubber with LLM
- [x] Add vector-based semantic search
- [x] Implement Scribe agent for user profiles
- [x] Add ContextHydrator for prompt assembly
- [x] Implement LatticeCrawler for memory retrieval
- [x] RAGAS benchmark tests (3/3 passing at 1.00)
- [ ] Performance optimizations
- [ ] Port remaining RAGAS tests (7C, 8, 9, 12)

## Contributing

This project uses:
- **TDD** - Write tests first, then implement
- **Scenario Testing** - Real dependencies, no mocks
- **Subagent-Driven Development** - Fresh context per task

See `DESIGN.md` for full architecture details.

## License

MIT License - See LICENSE file

## References

- [HMLR Python Implementation](https://github.com/Sean-V-Dev/HMLR-Agentic-AI-Memory-System)
- [MCP Specification](https://modelcontextprotocol.io)
- [Go MCP SDK](https://github.com/mark3labs/mcp-go)
