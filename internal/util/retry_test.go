// ABOUTME: Tests for retry utilities including exponential backoff
// ABOUTME: Validates backoff calculation, bounds, and jitter behavior
package util

import (
	"testing"
	"time"
)

func TestCalculateBackoff_ZeroAttempt(t *testing.T) {
	result := CalculateBackoff(time.Second, 0)
	if result != 0 {
		t.Errorf("expected 0 for attempt 0, got %v", result)
	}
}

func TestCalculateBackoff_FirstAttempt(t *testing.T) {
	baseDelay := 100 * time.Millisecond
	result := CalculateBackoff(baseDelay, 1)

	// First attempt: 2^1 * 100ms = 200ms, with ±25% jitter = 150ms to 250ms
	minExpected := 150 * time.Millisecond
	maxExpected := 250 * time.Millisecond

	if result < minExpected || result > maxExpected {
		t.Errorf("expected backoff between %v and %v, got %v", minExpected, maxExpected, result)
	}
}

func TestCalculateBackoff_ExponentialGrowth(t *testing.T) {
	baseDelay := 100 * time.Millisecond

	// Run multiple times to account for jitter, check median values grow
	for attempt := 1; attempt <= 5; attempt++ {
		// Expected base: 2^attempt * 100ms
		expectedBase := baseDelay * time.Duration(1<<uint(attempt))
		minExpected := expectedBase * 3 / 4 // -25%
		maxExpected := expectedBase * 5 / 4 // +25%

		result := CalculateBackoff(baseDelay, attempt)

		if result < minExpected || result > maxExpected {
			t.Errorf("attempt %d: expected backoff between %v and %v, got %v",
				attempt, minExpected, maxExpected, result)
		}
	}
}

func TestCalculateBackoff_CapsAt30Seconds(t *testing.T) {
	baseDelay := time.Second

	// Attempt 10 would give 2^10 * 1s = 1024s without cap
	result := CalculateBackoff(baseDelay, 10)

	// Should be capped at 30s with ±25% jitter = 22.5s to 37.5s
	maxAllowed := 37500 * time.Millisecond

	if result > maxAllowed {
		t.Errorf("expected backoff <= %v (30s + 25%% jitter), got %v", maxAllowed, result)
	}
}

func TestCalculateBackoff_AttemptCappedAt30(t *testing.T) {
	baseDelay := time.Millisecond

	// Very high attempt values should not overflow or panic
	result := CalculateBackoff(baseDelay, 100)

	// Should be capped at 30s with jitter
	maxAllowed := 37500 * time.Millisecond

	if result > maxAllowed {
		t.Errorf("expected backoff <= %v for high attempt, got %v", maxAllowed, result)
	}
	if result < 0 {
		t.Error("backoff should never be negative")
	}
}

func TestCalculateBackoff_JitterDistribution(t *testing.T) {
	baseDelay := time.Second
	attempt := 2 // 2^2 * 1s = 4s base

	// Run many iterations to verify jitter is applied
	var results []time.Duration
	for i := 0; i < 100; i++ {
		results = append(results, CalculateBackoff(baseDelay, attempt))
	}

	// Check that we get some variation (not all the same value)
	allSame := true
	for i := 1; i < len(results); i++ {
		if results[i] != results[0] {
			allSame = false
			break
		}
	}

	if allSame {
		t.Error("jitter should produce varying results, but all 100 samples were identical")
	}

	// Check all results are within expected bounds (4s ± 25% = 3s to 5s)
	for i, r := range results {
		if r < 3*time.Second || r > 5*time.Second {
			t.Errorf("sample %d: expected between 3s and 5s, got %v", i, r)
		}
	}
}

func TestCalculateBackoff_NegativeAttemptReturnsZero(t *testing.T) {
	// Negative attempts should return 0 (same as attempt 0)
	result := CalculateBackoff(time.Second, -1)

	if result != 0 {
		t.Errorf("expected 0 for negative attempt, got %v", result)
	}

	// Also test very negative values
	result = CalculateBackoff(time.Second, -100)
	if result != 0 {
		t.Errorf("expected 0 for very negative attempt, got %v", result)
	}
}
