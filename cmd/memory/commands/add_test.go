// ABOUTME: Tests for add command and extractStringArray helper
// ABOUTME: Verifies memory creation and metadata extraction

package commands

import (
	"testing"
)

func TestNewAddCmd(t *testing.T) {
	cmd := NewAddCmd()

	if cmd.Use != "add [text]" {
		t.Errorf("Use = %q, want %q", cmd.Use, "add [text]")
	}

	if cmd.Short == "" {
		t.Error("Short description should not be empty")
	}

	if cmd.Long == "" {
		t.Error("Long description should not be empty")
	}
}

func TestAddCmd_Flags(t *testing.T) {
	cmd := NewAddCmd()

	tests := []struct {
		flagName string
		defValue string
	}{
		{"file", ""},
		{"tags", "[]"},
	}

	for _, tt := range tests {
		t.Run(tt.flagName, func(t *testing.T) {
			flag := cmd.Flags().Lookup(tt.flagName)
			if flag == nil {
				t.Fatalf("--%s flag not found", tt.flagName)
			}
			if flag.DefValue != tt.defValue {
				t.Errorf("--%s default = %q, want %q", tt.flagName, flag.DefValue, tt.defValue)
			}
		})
	}
}

func TestExtractStringArray(t *testing.T) {
	tests := []struct {
		name     string
		metadata map[string]interface{}
		key      string
		want     []string
	}{
		{
			name: "string array present",
			metadata: map[string]interface{}{
				"keywords": []interface{}{"go", "testing", "cli"},
			},
			key:  "keywords",
			want: []string{"go", "testing", "cli"},
		},
		{
			name: "key not present",
			metadata: map[string]interface{}{
				"other": "value",
			},
			key:  "keywords",
			want: []string{},
		},
		{
			name:     "nil metadata",
			metadata: nil,
			key:      "keywords",
			want:     []string{},
		},
		{
			name: "empty array",
			metadata: map[string]interface{}{
				"keywords": []interface{}{},
			},
			key:  "keywords",
			want: []string{},
		},
		{
			name: "mixed types in array (only strings extracted)",
			metadata: map[string]interface{}{
				"keywords": []interface{}{"string1", 123, "string2", true},
			},
			key:  "keywords",
			want: []string{"string1", "string2"},
		},
		{
			name: "value is not array",
			metadata: map[string]interface{}{
				"keywords": "not an array",
			},
			key:  "keywords",
			want: []string{},
		},
		{
			name: "topics extraction",
			metadata: map[string]interface{}{
				"topics": []interface{}{"programming", "memory"},
			},
			key:  "topics",
			want: []string{"programming", "memory"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractStringArray(tt.metadata, tt.key)
			if len(got) != len(tt.want) {
				t.Errorf("extractStringArray() length = %d, want %d", len(got), len(tt.want))
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("extractStringArray()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestAddCmd_ArgsValidation(t *testing.T) {
	cmd := NewAddCmd()

	// Args should allow 0 or 1 arguments
	if cmd.Args == nil {
		t.Error("Args validator should be set")
	}
}

func TestAddCmd_Examples(t *testing.T) {
	cmd := NewAddCmd()

	// Long description should contain examples
	if cmd.Long == "" {
		t.Fatal("Long description should not be empty")
	}

	// Should mention file flag
	if !findSubstring(cmd.Long, "--file") {
		t.Error("Long description should mention --file flag")
	}

	// Should mention tags flag
	if !findSubstring(cmd.Long, "--tags") {
		t.Error("Long description should mention --tags flag")
	}
}
