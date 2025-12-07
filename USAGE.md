# When Should an Agent Use Memory?

## Overview

The Memory (HMLR) system provides **persistent, semantic memory** for LLM agents across conversations. Think of it as long-term memory that persists between sessions and allows agents to recall context from hours, days, or weeks ago.

## Key Use Cases

### ✅ When to Use Memory

#### 1. **Multi-Session Projects**
**Use when:** Working on projects that span multiple conversations
```
User: "Yesterday we discussed building a Python API"
Agent: [Uses memory to recall: discussed Flask vs FastAPI, decided on FastAPI, needs PostgreSQL]
```

**Without Memory:** Agent has no context, starts from scratch
**With Memory:** Agent continues where you left off

---

#### 2. **User Preferences & Context**
**Use when:** User shares information about themselves, their preferences, or their environment

Examples:
- "I prefer Python over JavaScript"
- "My API key for OpenAI is sk-..."
- "I'm working on a macOS system"
- "I'm vegetarian"
- "My team uses React and TypeScript"

**Why:** Memory extracts facts and builds a user profile that persists across all future conversations

---

#### 3. **Code Review & Iteration**
**Use when:** Reviewing or updating code from previous sessions

```
User: "Can you update the authentication code we wrote last week?"
Agent: [Searches memory for "authentication code"]
Agent: "I found our previous implementation using JWT tokens. Would you like me to..."
```

---

#### 4. **Learning User Patterns**
**Use when:** User repeatedly asks similar questions or works in similar domains

Memory tracks:
- Topics the user is interested in
- Programming languages they use
- Tools and frameworks they prefer
- Common problems they encounter

**Example:** After 5 conversations about Go concurrency, Memory knows to prioritize Go-specific solutions

---

#### 5. **Complex Research Tasks**
**Use when:** Gathering information over multiple sessions

```
Session 1: "Research database options for my project"
Session 2: "What were those databases we looked at yesterday?"
Agent: [Retrieves: PostgreSQL, MongoDB, Redis with pros/cons]
```

---

### ❌ When NOT to Use Memory

#### 1. **Single-Turn Questions**
**Don't use for:** Simple, standalone questions with no context needed
```
❌ "What's the weather?"
❌ "Convert 100 USD to EUR"
❌ "What's 2+2?"
```

**Why:** No need to persist these - they're stateless queries

---

#### 2. **Sensitive/Temporary Information**
**Don't use for:** Passwords, temporary tokens, or highly sensitive data you don't want persisted

**Note:** While Memory *can* store API keys and credentials (and does so for user convenience), consider whether you want this data in long-term storage.

---

#### 3. **Real-Time Data**
**Don't use for:** Information that changes frequently or needs real-time lookup
```
❌ "What's the current stock price?"
❌ "What's trending on Twitter right now?"
```

**Why:** Memory is for historical context, not live data

---

#### 4. **Every Single Message**
**Don't use for:** Storing trivial conversational turns

**Good practice:** Store meaningful exchanges, decisions, and information. Skip:
- Acknowledgments ("OK", "Thanks", "Got it")
- Simple clarifications
- Routine confirmations

---

## Memory Architecture Decision Tree

```
┌─────────────────────────────────────┐
│ Does this information need to       │
│ persist beyond this conversation?   │
└─────────────┬───────────────────────┘
              │
      ┌───────┴───────┐
     YES             NO
      │               │
      ▼               ▼
┌─────────────┐  [Don't use Memory]
│ Use Memory  │  [Use normal chat]
└─────────────┘
```

**Ask yourself:**
1. Will I need this information tomorrow/next week?
2. Does this build on previous conversations?
3. Is this about user preferences or context?
4. Am I gathering information over multiple sessions?

**If YES to any → Use Memory**

---

## How Memory Routes Context

Memory uses the **Governor** component to intelligently route conversations into topics:

### Scenario 1: Topic Continuation
**When:** Talking about the same topic in sequence
```
Turn 1: "Let's discuss Python async programming"
Turn 2: "Show me an example with asyncio"  ← Continues "Python async" topic
```

### Scenario 2: Topic Resumption
**When:** Returning to a topic from earlier
```
Turn 1: "Let's discuss database design"
Turn 5: "Back to databases - should I use foreign keys?" ← Resumes "database" topic
```

### Scenario 3: Topic Shift
**When:** Changing topics mid-conversation
```
Turn 1: "Help me debug this Python code"
Turn 2: "Actually, can we talk about Docker instead?" ← New topic
```

### Scenario 4: New Topic (First Turn)
**When:** Starting a fresh conversation

---

## Practical Examples

### Example 1: Software Development Project

**Session 1 (Monday):**
```
User: "I'm building a todo app in Go with SQLite"
Agent: [Stores: project=todo-app, language=Go, database=SQLite]
```

**Session 2 (Wednesday):**
```
User: "How should I structure the database for my todo app?"
Agent: [Searches memory → finds "todo app in Go with SQLite"]
Agent: "For your Go todo app using SQLite, I recommend..."
```

---

### Example 2: User Preferences

**Session 1:**
```
User: "I prefer minimal dependencies and standard library when possible"
Agent: [FactScrubber extracts: preference="minimal dependencies, standard library"]
```

**Session 3:**
```
User: "What HTTP router should I use?"
Agent: [Checks user profile → sees "minimal dependencies preference"]
Agent: "Given your preference for minimal dependencies, I'd recommend net/http from the standard library..."
```

---

### Example 3: Multi-Day Research

**Monday:**
```
User: "Research vector databases for semantic search"
Agent: [Stores findings about Pinecone, Weaviate, Milvus]
```

**Thursday:**
```
User: "Which vector database did we decide on?"
Agent: [Searches memory → retrieves previous research]
Agent: "We looked at Pinecone, Weaviate, and Milvus. Based on your needs..."
```

---

## Best Practices for Agents

### 1. **Store After Significant Exchanges**
```python
# Good - meaningful exchange
user: "I'm building a REST API for my mobile app"
agent: [Store this - it's project context]

# Skip - routine acknowledgment
user: "Thanks!"
agent: [Don't store - no meaningful context]
```

### 2. **Search Before Answering Context Questions**
```python
user: "What was that library we discussed?"
agent: [First search memory before saying "I don't recall"]
```

### 3. **Extract Facts Proactively**
```python
user: "My email is alice@example.com and I use VSCode"
agent: [FactScrubber should extract: email="alice@example.com", editor="VSCode"]
```

### 4. **Use Retrieved Context in Responses**
```python
# Bad
agent: "Here's how to do X"

# Good
agent: "Since you're using FastAPI (from our discussion last week), here's how to do X with FastAPI-specific patterns..."
```

---

## Memory System Capabilities

| Component | What It Does | When Agent Should Use It |
|-----------|--------------|--------------------------|
| **Governor** | Routes conversations into topics | Automatically happens - agent doesn't control this |
| **FactScrubber** | Extracts key-value facts | When user shares personal info, preferences, credentials |
| **LatticeCrawler** | Semantic search | When user asks "what did we discuss about X?" |
| **Scribe** | Learns user profile | Runs async - agent doesn't control this |
| **ChunkEngine** | Breaks text into searchable chunks | Automatically happens during storage |
| **ContextHydrator** | Assembles relevant context | Automatically provides context for responses |

---

## Integration with Claude Desktop

When configured as an MCP server in Claude Desktop, Memory provides tools:

- `store_conversation` - Store a turn with user message and AI response
- `retrieve_memory` - Search past conversations by query
- `list_active_topics` - See what topics are being tracked
- `get_topic_history` - Get full history for a specific topic
- `get_user_profile` - Retrieve learned user preferences

**Agent should use these tools when:**
- User references past conversations
- User asks about preferences or context
- Working on multi-session projects
- Building up knowledge over time

---

## Summary

**Use Memory when:**
- 🔄 Multi-session projects
- 👤 User preferences & facts
- 📚 Research over time
- 🔍 Need to recall previous context
- 💡 Building long-term understanding

**Don't use Memory when:**
- ⚡ One-off questions
- 🔒 Highly sensitive temporary data
- 📊 Real-time data lookups
- 💬 Trivial acknowledgments

**Golden Rule:** If the information would be valuable tomorrow, next week, or next month → Use Memory.
