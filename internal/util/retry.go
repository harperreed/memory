// ABOUTME: Retry utilities for API calls with exponential backoff
// ABOUTME: Shared by LLM client and Scribe for consistent retry behavior
package util

import (
	"math/rand/v2"
	"time"
)

// CalculateBackoff returns exponential backoff with jitter
// Base delay is doubled each attempt, with random jitter up to 25%
func CalculateBackoff(baseDelay time.Duration, attempt int) time.Duration {
	if attempt <= 0 {
		return 0
	}
	// Cap attempt to avoid overflow in bit shift (max 30 for safety)
	if attempt > 30 {
		attempt = 30
	}
	// Exponential: 2^attempt * base
	backoff := baseDelay * time.Duration(1<<uint(attempt))
	// Cap at 30 seconds
	if backoff > 30*time.Second {
		backoff = 30 * time.Second
	}
	// Add jitter: -25% to +25% using auto-seeded math/rand/v2
	jitter := time.Duration(rand.Int64N(int64(backoff)/2)) - backoff/4
	return backoff + jitter
}
