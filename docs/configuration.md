# Configuration Guide

The memory MCP server uses a centralized configuration system that loads settings from environment variables. All configuration is handled through the `internal/config` package.

## Configuration Options

### Charm Settings

| Variable | Default | Description |
|----------|---------|-------------|
| `CHARM_HOST` | `cloud.charm.sh` | Charm cloud host for data sync |
| `CHARM_DB` | `memory` | Database name in Charm KV |
| `CHARM_AUTO_SYNC` | `true` | Enable automatic sync to cloud after writes |

### OpenAI Settings

| Variable | Default | Description |
|----------|---------|-------------|
| `OPENAI_API_KEY` | (required) | OpenAI API key for embeddings and chat |
| `MEMORY_OPENAI_MODEL` | `gpt-4o-mini` | Chat model for topic analysis |
| `MEMORY_EMBEDDING_MODEL` | `text-embedding-3-small` | Embedding model for vector search |
| `OPENAI_TIMEOUT` | `30s` | Request timeout (e.g., `30s`, `1m`) |
| `OPENAI_MAX_RETRIES` | `3` | Max retry attempts (0-10) |
| `OPENAI_RETRY_DELAY` | `2s` | Delay between retries (e.g., `2s`, `5s`) |

### Memory System Settings

| Variable | Default | Description |
|----------|---------|-------------|
| `TOPIC_MATCH_THRESHOLD` | `0.3` | Similarity threshold for topic matching (0.0-1.0) |
| `VECTOR_DIMENSION` | `1536` | Vector dimension (1536 for text-embedding-3-small, 3072 for large) |

## Usage Examples

### Basic Setup

```bash
# Minimal required configuration
export OPENAI_API_KEY="sk-..."

# Start the server
./memory server
```

### Custom Charm Host

```bash
export CHARM_HOST="custom.charm.sh"
export CHARM_DB="my_memory_db"
export CHARM_AUTO_SYNC="true"
```

### Production Settings

```bash
# Use larger models for better accuracy
export MEMORY_OPENAI_MODEL="gpt-4"
export MEMORY_EMBEDDING_MODEL="text-embedding-3-large"
export VECTOR_DIMENSION="3072"

# Increase timeouts for reliability
export OPENAI_TIMEOUT="60s"
export OPENAI_MAX_RETRIES="5"
export OPENAI_RETRY_DELAY="3s"

# Stricter topic matching
export TOPIC_MATCH_THRESHOLD="0.5"
```

### Development Settings

```bash
# Use smaller, faster models
export MEMORY_OPENAI_MODEL="gpt-4o-mini"
export MEMORY_EMBEDDING_MODEL="text-embedding-3-small"

# Faster timeouts for quick iteration
export OPENAI_TIMEOUT="15s"
export OPENAI_MAX_RETRIES="2"
export OPENAI_RETRY_DELAY="1s"

# Disable auto-sync for local testing
export CHARM_AUTO_SYNC="false"
```

## Validation

The configuration system validates settings on load:

- `TOPIC_MATCH_THRESHOLD` must be between 0.0 and 1.0
- `OPENAI_MAX_RETRIES` must be between 0 and 10
- Invalid values will cause the server to fail at startup with a clear error message

## Loading Configuration

Configuration is automatically loaded when the Charm client initializes. The system:

1. Reads environment variables
2. Applies defaults for missing values
3. Validates all settings
4. Returns errors for invalid configuration

You can also load configuration manually in code:

```go
import "github.com/harper/remember-standalone/internal/config"

cfg, err := config.Load()
if err != nil {
    log.Fatal(err)
}
```

## Environment File

For convenience, you can use a `.env` file (already in `.gitignore`):

```bash
# .env
OPENAI_API_KEY=sk-...
CHARM_HOST=cloud.charm.sh
MEMORY_OPENAI_MODEL=gpt-4o-mini
```

Then load it in your application using `godotenv`:

```go
import "github.com/joho/godotenv"

func main() {
    _ = godotenv.Load() // Ignore error if .env doesn't exist
    // ... rest of application
}
```
