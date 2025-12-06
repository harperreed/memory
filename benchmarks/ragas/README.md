# RAGAS Benchmarks for HMLR

ABOUTME: Documentation for RAGAS benchmark tests ported from Python HMLR
ABOUTME: Explains test scenarios, metrics, and how to run benchmarks

## Overview

This directory contains RAGAS (Retrieval Augmented Generation Assessment) benchmark tests ported from the Python HMLR implementation. These tests validate that the Go HMLR system achieves 1.00 faithfulness and recall scores for critical scenarios.

## What is RAGAS?

RAGAS is a framework for evaluating RAG (Retrieval Augmented Generation) systems using two key metrics:

1. **Faithfulness** (0.0-1.0): Does the LLM response accurately reflect the retrieved context? No hallucinations?
2. **Context Recall** (0.0-1.0): Was the correct context retrieved from memory? Did we find what we needed?

Target: **1.00 on both metrics** for production-grade memory systems.

## Benchmark Tests

### Test 7A: API Key Rotation (Temporal Conflict)

**Scenario**: User updates information over time - system must prefer most recent truth.

```
Turn 1: "My API Key for the weather service is ABC123."
Turn 2: "I rotated my keys. The new API Key is XYZ789." (weeks later)
Turn 3: "What is my API key?"
```

**Expected**: LLM responds with `XYZ789` (new key), NOT `ABC123` (old key)

**Tests**:
- Temporal conflict resolution
- Timestamp-based fact prioritization
- Bridge Block conversation context vs fact store

**Success Criteria**:
- Faithfulness: 1.00 (response matches most recent context)
- Context Recall: 1.00 (retrieved XYZ789, not ABC123)

---

### Test 7B: Vegetarian Trap (User Profile vs Override)

**Scenario**: User profile constraints must be honored even when context suggests otherwise.

```
Setup: User profile contains "strictly vegetarian, does not eat meat or fish"
Turn 1: "I'm going to a steakhouse tonight. Can you recommend a dish?"
```

**Expected**: LLM acknowledges vegetarian preference, suggests vegetarian options (NOT steak)

**Tests**:
- Cross-topic user profile persistence
- Profile constraints override situational context
- Scribe extraction + ContextHydrator inclusion

**Success Criteria**:
- Faithfulness: 1.00 (response respects vegetarian constraint)
- Context Recall: 1.00 (user profile constraint retrieved)

---

### Test 2A: 10-Turn Vague Secret Retrieval (Zero-Keyword Recall)

**Scenario**: LLM must interpret vague queries semantically, not just match keywords.

```
Turn 1: "My API key for the weather service is ABC123XYZ"
Turns 2-9: Conversation about weather dashboard (NO mention of API key)
Turn 10: "Remind me what credential I need for the weather service?"
```

**Expected**: LLM interprets "credential" → API key, retrieves `ABC123XYZ`

**Tests**:
- Fact extraction from Turn 1
- Fact persistence across 10 turns in same block
- Vague query interpretation (semantic understanding, not keyword matching)
- Precise retrieval despite ambiguity

**Success Criteria**:
- Faithfulness: 1.00 (response contains correct API key)
- Context Recall: 1.00 (retrieved API key fact despite vague query)

## Metrics Explained

### Faithfulness Score

Measures whether the LLM response is grounded in retrieved context (no hallucinations).

- **1.00**: Response perfectly reflects retrieved context
- **0.90-0.99**: Minor deviations or extra context
- **0.80-0.89**: Some unsupported statements
- **< 0.80**: Hallucinations or incorrect information

**Calculation**:
```
faithfulness = supported_statements / total_statements
```

### Context Recall Score

Measures whether the correct context was retrieved from memory.

- **1.00**: All relevant context retrieved
- **0.90-0.99**: Most relevant context retrieved
- **0.80-0.89**: Some relevant context missing
- **< 0.80**: Critical context not retrieved

**Calculation**:
```
recall = ground_truth_items_in_context / total_ground_truth_items
```

## Running Benchmarks

### Prerequisites

```bash
# Build the project
cd /Users/harper/Public/src/2389/remember-standalone
go build -o bin/hmlr-benchmark ./cmd/benchmark
```

### Run All Benchmarks

```bash
# Run all RAGAS benchmarks with JSON output
./bin/hmlr-benchmark --output=json

# Run specific test
./bin/hmlr-benchmark --test=7a

# Run with verbose output
./bin/hmlr-benchmark --verbose
```

### Example Output

```json
{
  "test_7a_api_key_rotation": {
    "scenario": "API Key Rotation (Temporal Conflict)",
    "faithfulness": 1.00,
    "context_recall": 1.00,
    "overall_score": 1.00,
    "status": "PASS",
    "details": {
      "turn_1_stored": "ABC123",
      "turn_2_stored": "XYZ789",
      "turn_3_response_contains": "XYZ789",
      "turn_3_response_excludes": "ABC123"
    }
  },
  "test_7b_vegetarian": {
    "scenario": "Vegetarian Trap (User Profile vs Override)",
    "faithfulness": 1.00,
    "context_recall": 1.00,
    "overall_score": 1.00,
    "status": "PASS"
  },
  "test_2a_vague_retrieval": {
    "scenario": "10-Turn Vague Secret Retrieval",
    "faithfulness": 1.00,
    "context_recall": 1.00,
    "overall_score": 1.00,
    "status": "PASS"
  }
}
```

## Interpreting Results

### Production Readiness Benchmarks

Based on RAGAS research and production RAG systems:

- **0.90 - 1.00**: Exceptional (production-grade)
- **0.80 - 0.90**: Excellent (acceptable for most use cases)
- **0.70 - 0.80**: Good (needs improvement for critical applications)
- **0.60 - 0.70**: Fair (baseline, not production-ready)
- **< 0.60**: Poor (needs significant work)

### HMLR Target

HMLR targets **1.00 on both metrics** for all benchmark tests because:

1. Memory systems MUST be faithful (no hallucinated memories)
2. Retrieval MUST be precise (correct context, not just similar)
3. User trust depends on consistent accuracy

## Implementation Details

### File Structure

```
benchmarks/ragas/
├── README.md              # This file
├── test_data.go           # Test scenario definitions
├── runner.go              # Test execution logic
├── metrics.go             # Faithfulness and recall calculations
└── scenarios/
    ├── test_7a.go         # API Key Rotation test
    ├── test_7b.go         # Vegetarian Trap test
    └── test_2a.go         # Vague Retrieval test
```

### Metrics Implementation

The Go implementation uses simplified RAGAS metrics:

1. **Faithfulness**: Compare LLM response against expected ground truth
   - Extract key facts from response
   - Verify all facts match retrieved context
   - No unsupported claims allowed

2. **Context Recall**: Verify correct context was retrieved
   - Check that expected facts are in retrieved context
   - Verify temporal ordering (for Test 7A)
   - Confirm profile constraints loaded (for Test 7B)

### Differences from Python RAGAS

The Python HMLR uses the full RAGAS library with LLM-based evaluation. This Go implementation uses:

- **Deterministic evaluation**: Rule-based checks (faster, consistent)
- **Ground truth comparison**: Direct string/fact matching
- **Simplified scoring**: Binary pass/fail converted to 1.00/0.00 scores

This is appropriate because:
- HMLR tests have clear ground truth (specific API keys, preferences)
- Binary outcomes are what matter (did it retrieve the right key or not?)
- Deterministic tests are more reliable for CI/CD

## Future Enhancements

1. **Additional Tests**:
   - Test 8: Multi-hop reasoning
   - Test 9: Long conversation (50+ turns)
   - Test 12: Hydra E2E validation

2. **Advanced Metrics**:
   - Context Precision (relevance of retrieved context)
   - Answer Relevancy (query-answer alignment)

3. **Integration**:
   - CI/CD pipeline integration
   - Automated regression testing
   - Performance benchmarking (latency, memory usage)

## References

- Python HMLR: `/tmp/hmlr-reference/tests/`
- RAGAS Framework: https://github.com/explodinggradients/ragas
- RAGAS Metrics Paper: https://arxiv.org/abs/2309.15217
