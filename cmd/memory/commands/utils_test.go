// ABOUTME: Tests for shared utility functions used by CLI commands
// ABOUTME: Verifies truncate, formatTime, containsString, and validation helpers

package commands

import (
	"testing"
	"time"
)

func TestTruncate(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{
			name:   "short string unchanged",
			input:  "hello",
			maxLen: 10,
			want:   "hello",
		},
		{
			name:   "exact length unchanged",
			input:  "hello",
			maxLen: 5,
			want:   "hello",
		},
		{
			name:   "long string truncated",
			input:  "hello world",
			maxLen: 8,
			want:   "hello...",
		},
		{
			name:   "very short maxLen",
			input:  "hello",
			maxLen: 2,
			want:   "he",
		},
		{
			name:   "maxLen equals 3",
			input:  "hello",
			maxLen: 3,
			want:   "hel",
		},
		{
			name:   "empty string",
			input:  "",
			maxLen: 10,
			want:   "",
		},
		{
			name:   "unicode string",
			input:  "你好世界！",
			maxLen: 3,
			want:   "你好世界！"[:3],
		},
		{
			name:   "unicode truncated with ellipsis",
			input:  "你好世界你好世界",
			maxLen: 5,
			want:   "你好...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncate(tt.input, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
			}
		})
	}
}

func TestFormatTime(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		input    time.Time
		contains string
	}{
		{
			name:     "just now (seconds ago)",
			input:    now.Add(-30 * time.Second),
			contains: "just now",
		},
		{
			name:     "minutes ago",
			input:    now.Add(-5 * time.Minute),
			contains: "m ago",
		},
		{
			name:     "hours ago",
			input:    now.Add(-3 * time.Hour),
			contains: "h ago",
		},
		{
			name:     "days ago",
			input:    now.Add(-2 * 24 * time.Hour),
			contains: "d ago",
		},
		{
			name:     "weeks ago (shows date)",
			input:    now.Add(-14 * 24 * time.Hour),
			contains: "-", // Date format contains hyphens
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatTime(tt.input)
			if !contains(got, tt.contains) {
				t.Errorf("formatTime() = %q, want to contain %q", got, tt.contains)
			}
		})
	}
}

func TestFormatTime_EdgeCases(t *testing.T) {
	now := time.Now()

	// Test boundary conditions
	tests := []struct {
		name     string
		input    time.Time
		notEmpty bool
	}{
		{
			name:     "59 seconds ago",
			input:    now.Add(-59 * time.Second),
			notEmpty: true,
		},
		{
			name:     "60 seconds ago",
			input:    now.Add(-60 * time.Second),
			notEmpty: true,
		},
		{
			name:     "59 minutes ago",
			input:    now.Add(-59 * time.Minute),
			notEmpty: true,
		},
		{
			name:     "23 hours ago",
			input:    now.Add(-23 * time.Hour),
			notEmpty: true,
		},
		{
			name:     "6 days ago",
			input:    now.Add(-6 * 24 * time.Hour),
			notEmpty: true,
		},
		{
			name:     "7 days ago",
			input:    now.Add(-7 * 24 * time.Hour),
			notEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatTime(tt.input)
			if tt.notEmpty && got == "" {
				t.Error("formatTime() returned empty string")
			}
		})
	}
}

func TestContainsString(t *testing.T) {
	tests := []struct {
		name  string
		slice []string
		item  string
		want  bool
	}{
		{
			name:  "item present",
			slice: []string{"apple", "banana", "cherry"},
			item:  "banana",
			want:  true,
		},
		{
			name:  "item absent",
			slice: []string{"apple", "banana", "cherry"},
			item:  "grape",
			want:  false,
		},
		{
			name:  "empty slice",
			slice: []string{},
			item:  "apple",
			want:  false,
		},
		{
			name:  "nil slice",
			slice: nil,
			item:  "apple",
			want:  false,
		},
		{
			name:  "empty item in slice",
			slice: []string{"", "apple"},
			item:  "",
			want:  true,
		},
		{
			name:  "case sensitive match",
			slice: []string{"Apple", "Banana"},
			item:  "apple",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := containsString(tt.slice, tt.item)
			if got != tt.want {
				t.Errorf("containsString(%v, %q) = %v, want %v", tt.slice, tt.item, got, tt.want)
			}
		})
	}
}

func TestValidatePositiveInt(t *testing.T) {
	tests := []struct {
		name      string
		n         int
		fieldName string
		wantErr   bool
	}{
		{
			name:      "positive value",
			n:         5,
			fieldName: "count",
			wantErr:   false,
		},
		{
			name:      "zero value",
			n:         0,
			fieldName: "limit",
			wantErr:   true,
		},
		{
			name:      "negative value",
			n:         -1,
			fieldName: "offset",
			wantErr:   true,
		},
		{
			name:      "large positive value",
			n:         1000000,
			fieldName: "max",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePositiveInt(tt.n, tt.fieldName)
			if (err != nil) != tt.wantErr {
				t.Errorf("validatePositiveInt(%d, %q) error = %v, wantErr %v", tt.n, tt.fieldName, err, tt.wantErr)
			}
			if tt.wantErr && err != nil {
				// Error message should contain the field name
				if !contains(err.Error(), tt.fieldName) {
					t.Errorf("Error message should contain field name %q: %v", tt.fieldName, err)
				}
			}
		})
	}
}

// Helper function for test - checks if s contains substr
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || (len(s) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
