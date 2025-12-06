# RAGAS Benchmarks - Quick Start

ABOUTME: Quick start guide for running RAGAS benchmarks
ABOUTME: Step-by-step instructions with examples and troubleshooting

## Prerequisites

1. **OpenAI API Key**: Required for LLM responses
   ```bash
   export OPENAI_API_KEY="sk-..."
   ```

   Or add to `.env` file:
   ```
   OPENAI_API_KEY=sk-...
   ```

2. **Go 1.21+**: Check version
   ```bash
   go version
   ```

3. **Build the benchmark**:
   ```bash
   cd /Users/harper/Public/src/2389/remember-standalone
   go build -o bin/hmlr-benchmark ./cmd/benchmark
   ```

## Running Benchmarks

### Run All Tests

```bash
./bin/hmlr-benchmark --output=results.json
```

**Output**:
```
========================================
HMLR RAGAS Benchmarks
========================================

Running all RAGAS benchmark tests...

========================================
RUNNING: API Key Rotation (Temporal Conflict)
========================================
...

========================================
BENCHMARK SUMMARY
========================================

test_7a: API Key Rotation (Temporal Conflict)
  Faithfulness: 1.00
  Context Recall: 1.00
  Overall: 1.00
  Status: PASS

test_7b: Vegetarian Trap (User Profile vs Override)
  Faithfulness: 1.00
  Context Recall: 1.00
  Overall: 1.00
  Status: PASS

test_2a: 10-Turn Vague Secret Retrieval
  Faithfulness: 1.00
  Context Recall: 1.00
  Overall: 1.00
  Status: PASS

========================================
Total Tests: 3
Passed: 3
Failed: 0
========================================
✓ Results exported to: results.json
```

### Run Specific Test

```bash
# Test 7A: API Key Rotation
./bin/hmlr-benchmark --test=7a --verbose

# Test 7B: Vegetarian Trap
./bin/hmlr-benchmark --test=7b --verbose

# Test 2A: Vague Retrieval
./bin/hmlr-benchmark --test=2a --verbose
```

### Verbose Mode

```bash
./bin/hmlr-benchmark --verbose
```

Shows detailed turn-by-turn output:
```
[Turn 1] User: My API Key for the weather service is ABC123.
[Turn 1] AI: I'll remember that your API key for the weather service is ABC123...

[Turn 2] User: I rotated my keys. The new API Key is XYZ789.
[Turn 2] AI: Got it, I've updated your weather service API key to XYZ789...

[Turn 3] User: What is my API key?
[Turn 3] AI: Your current API key is XYZ789.
```

## Understanding Results

### JSON Output Format

```json
{
  "timestamp": "2025-12-06T12:00:00Z",
  "total_tests": 3,
  "passed": 3,
  "failed": 0,
  "results": [
    {
      "TestID": "test_7a",
      "TestName": "API Key Rotation (Temporal Conflict)",
      "FaithfulnessScore": 1.0,
      "ContextRecallScore": 1.0,
      "OverallScore": 1.0,
      "Status": "PASS",
      "Details": {
        "faithfulness_detail": "Perfect faithfulness - response matches expected ground truth",
        "recall_detail": "Perfect context recall - all expected items retrieved",
        "final_response": "Your current API key is XYZ789.",
        "context_items": 5
      }
    }
  ]
}
```

### Metrics Explained

#### Faithfulness (0.0-1.0)
- **1.0**: Response perfectly reflects retrieved context (no hallucinations)
- **0.5**: Partial match (missing expected items OR forbidden items present)
- **0.0**: Complete failure (both missing and forbidden items)

**Example**:
- Expected: "XYZ789"
- Forbidden: "ABC123"
- Response: "Your current API key is XYZ789" → **1.0** ✅
- Response: "Your API key is ABC123" → **0.0** ❌

#### Context Recall (0.0-1.0)
- **1.0**: All expected context items retrieved
- **0.5**: 50% of expected items retrieved
- **0.0**: No expected items retrieved

**Example**:
- Expected context: ["XYZ789", "weather_api_key"]
- Retrieved: ["XYZ789", "weather_api_key"] → **1.0** ✅
- Retrieved: ["XYZ789"] → **0.5** ⚠️
- Retrieved: [] → **0.0** ❌

#### Overall Score
Average of Faithfulness and Context Recall:
```
Overall = (Faithfulness + ContextRecall) / 2
```

### Pass/Fail Criteria

A test **PASSES** if:
- Faithfulness >= 0.9 **AND**
- Context Recall >= 0.9

Target for production: **1.00 on both metrics**

## Test Scenarios

### Test 7A: API Key Rotation
**What it tests**: Temporal conflict resolution

**Scenario**:
1. Turn 1: Store API key "ABC123"
2. Turn 2: Update to "XYZ789" (weeks later)
3. Turn 3: Query for API key

**Expected**: Returns XYZ789 (new key), NOT ABC123 (old key)

**Why it matters**: Memory systems must prefer recent truths over past truths

---

### Test 7B: Vegetarian Trap
**What it tests**: User profile persistence across contexts

**Scenario**:
1. Setup: User profile contains "strictly vegetarian"
2. Turn 1: "I'm going to a steakhouse. What should I order?"

**Expected**: Acknowledges vegetarian preference, suggests vegetarian options

**Why it matters**: User constraints must override situational context

---

### Test 2A: Vague Retrieval
**What it tests**: Semantic query understanding

**Scenario**:
1. Turn 1: "My API key is ABC123XYZ"
2. Turns 2-9: Conversation about weather dashboard
3. Turn 10: "What credential do I need?" (NOT "what API key")

**Expected**: Interprets "credential" → API key, returns ABC123XYZ

**Why it matters**: Real users don't use exact keywords - system must understand semantics

## Troubleshooting

### Error: OPENAI_API_KEY not set

```bash
export OPENAI_API_KEY="sk-..."
```

Or create `.env` file in project root.

### Error: Failed to initialize storage

Check file permissions:
```bash
ls -la /tmp/
```

Benchmark creates temporary database in `/tmp/hmlr_benchmark_*.db`

### All tests failing with low scores

**Possible causes**:
1. LLM model changed behavior (OpenAI API updated)
2. Storage/retrieval not working correctly
3. Governor routing incorrectly

**Debug steps**:
```bash
# Run with verbose to see turn-by-turn
./bin/hmlr-benchmark --test=7a --verbose

# Check final response manually
# Does it contain the expected values?
```

### Test 7B fails: "No vegetarian awareness"

**Cause**: User profile not loaded into context

**Fix**: Check that `GetUserProfile()` returns profile with constraints

### Test 2A fails: "Missing ABC123XYZ"

**Cause**: Fact not persisting across 10 turns OR vague query not understood

**Debug**:
```bash
# Run verbose to see which turn loses the fact
./bin/hmlr-benchmark --test=2a --verbose

# Check if "credential" query retrieves API key fact
```

## Comparing to Python HMLR

### Differences

| Aspect | Python HMLR | Go HMLR Benchmark |
|--------|-------------|-------------------|
| Metrics | LLM-based RAGAS eval | Deterministic string matching |
| Scoring | Probabilistic (0.0-1.0) | Binary (1.0 or 0.0) |
| Response | Real LLM calls | Simplified mock responses |
| Speed | ~5-10s per test | ~2-3s per test |

### Why Simplified Metrics?

1. **Deterministic CI/CD**: Binary pass/fail is reliable for automation
2. **Clear Ground Truth**: These tests have specific expected values (API keys, preferences)
3. **Fast Iteration**: No LLM calls for evaluation = faster feedback

### When to Use Full RAGAS

Use Python RAGAS library for:
- Open-ended questions (no single correct answer)
- Evaluating response quality/style
- Benchmarking against published RAGAS papers

Use this Go benchmark for:
- CI/CD regression testing
- Fact accuracy validation
- Memory persistence verification

## Next Steps

1. **Run benchmarks**: `./bin/hmlr-benchmark`
2. **Check results**: Open `benchmark_results.json`
3. **Investigate failures**: Use `--verbose` mode
4. **Add more tests**: See `benchmarks/ragas/test_data.go`

## Further Reading

- Full documentation: `benchmarks/ragas/README.md`
- Python reference tests: `/tmp/hmlr-reference/tests/`
- RAGAS framework: https://github.com/explodinggradients/ragas
