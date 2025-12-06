# Scribe Agent - Async User Profile Learning

## Overview

Scribe is a background agent that learns about the user from their conversations. It runs asynchronously (fire-and-forget) and updates the user profile with extracted information.

## How It Works

1. **Triggered on Every Conversation**: When `store_conversation` is called, Scribe is triggered in the background
2. **LLM Extraction**: Uses GPT-4o-mini to extract user information from the message
3. **Intelligent Merge**: Updates profile without duplicating existing information
4. **Async Operation**: Runs in a goroutine, doesn't block the main conversation flow
5. **Error Handling**: Logs errors but never crashes - profile learning is optional

## What Scribe Learns

Scribe extracts three types of information:

- **Name**: User's name
- **Preferences**: How they like to work, their habits, methodologies they prefer
- **Topics of Interest**: Technologies, subjects, areas they're interested in

## Example

```go
// User says:
"Hi! My name is Harper and I love using Go for backend development.
I prefer TDD and keeping things simple."

// Scribe extracts:
{
  "name": "Harper",
  "preferences": ["using Go for backend development", "TDD", "keeping things simple"],
  "topics_of_interest": ["backend development", "Go"]
}

// Profile is updated asynchronously
```

## Integration

Scribe is integrated into the MCP handler:

```go
// In handleStoreConversation:
if h.scribe != nil {
    profile, _ := h.storage.GetUserProfile()
    if profile == nil {
        profile = &models.UserProfile{...}
    }
    // Fire-and-forget async update
    go h.scribe.UpdateProfileAsync(message, profile, h.storage)
}
```

## Concurrency Safety

- Uses mutex to protect concurrent profile updates
- Reloads profile from disk before each update
- Handles race conditions gracefully
- Multiple async updates won't corrupt data

## Testing

See `.scratch/scenario_08_scribe_test.go` for comprehensive tests:

- ✓ Learn user's name
- ✓ Learn preferences
- ✓ Learn topics of interest
- ✓ Merge without duplicates
- ✓ Error handling (empty messages, etc.)
- ✓ Multiple concurrent async updates

All tests use REAL LLM calls (GPT-4o-mini) - no mocks!

## Files

- `internal/core/scribe.go` - Scribe agent implementation
- `internal/models/user_profile.go` - UserProfile model with Merge method
- `internal/mcp/handlers.go` - Integration with store_conversation handler
- `.scratch/scenario_08_scribe_test.go` - End-to-end scenario tests
