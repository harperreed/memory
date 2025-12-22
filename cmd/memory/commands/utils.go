// ABOUTME: Shared utility functions for CLI commands
// ABOUTME: Consolidates duplicate code from list, search, profile commands
package commands

import (
	"fmt"
	"time"
)

// truncate shortens a string to maxLen, adding "..." if truncated
func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return string(runes[:maxLen-3]) + "..."
}

// formatTime formats a time for display
func formatTime(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

	if diff < time.Minute {
		return "just now"
	} else if diff < time.Hour {
		mins := int(diff.Minutes())
		return fmt.Sprintf("%dm ago", mins)
	} else if diff < 24*time.Hour {
		hours := int(diff.Hours())
		return fmt.Sprintf("%dh ago", hours)
	} else if diff < 7*24*time.Hour {
		days := int(diff.Hours() / 24)
		return fmt.Sprintf("%dd ago", days)
	}
	return t.Format("2006-01-02")
}

// containsString checks if a slice contains a string
func containsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// validatePositiveInt returns error if n is not positive
func validatePositiveInt(n int, name string) error {
	if n <= 0 {
		return fmt.Errorf("%s must be positive, got %d", name, n)
	}
	return nil
}
