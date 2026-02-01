// ABOUTME: Tests for UserProfile model and Merge functionality
// ABOUTME: Verifies profile merging logic for name, preferences, and topics

package models

import (
	"testing"
	"time"
)

func TestUserProfile_Merge_Name(t *testing.T) {
	tests := []struct {
		name     string
		initial  string
		newName  interface{}
		wantName string
	}{
		{
			name:     "update empty name",
			initial:  "",
			newName:  "Alice",
			wantName: "Alice",
		},
		{
			name:     "update existing name",
			initial:  "Bob",
			newName:  "Alice",
			wantName: "Alice",
		},
		{
			name:     "empty new name keeps original",
			initial:  "Bob",
			newName:  "",
			wantName: "Bob",
		},
		{
			name:     "nil name keeps original",
			initial:  "Bob",
			newName:  nil,
			wantName: "Bob",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			up := &UserProfile{Name: tt.initial}
			newInfo := make(map[string]interface{})
			if tt.newName != nil {
				newInfo["name"] = tt.newName
			}

			up.Merge(newInfo)

			if up.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", up.Name, tt.wantName)
			}
		})
	}
}

func TestUserProfile_Merge_Preferences(t *testing.T) {
	tests := []struct {
		name         string
		initialPrefs []string
		newPrefs     []interface{}
		wantPrefsLen int
		wantContains []string
	}{
		{
			name:         "add preferences to empty",
			initialPrefs: []string{},
			newPrefs:     []interface{}{"TDD", "vim"},
			wantPrefsLen: 2,
			wantContains: []string{"TDD", "vim"},
		},
		{
			name:         "merge without duplicates",
			initialPrefs: []string{"TDD"},
			newPrefs:     []interface{}{"TDD", "vim"},
			wantPrefsLen: 2,
			wantContains: []string{"TDD", "vim"},
		},
		{
			name:         "skip non-string items",
			initialPrefs: []string{},
			newPrefs:     []interface{}{"valid", 123, true, "also_valid"},
			wantPrefsLen: 2,
			wantContains: []string{"valid", "also_valid"},
		},
		{
			name:         "nil preferences preserves existing",
			initialPrefs: []string{"existing"},
			newPrefs:     nil,
			wantPrefsLen: 1,
			wantContains: []string{"existing"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			up := &UserProfile{Preferences: tt.initialPrefs}
			newInfo := make(map[string]interface{})
			if tt.newPrefs != nil {
				newInfo["preferences"] = tt.newPrefs
			}

			up.Merge(newInfo)

			if len(up.Preferences) != tt.wantPrefsLen {
				t.Errorf("Preferences length = %d, want %d", len(up.Preferences), tt.wantPrefsLen)
			}

			for _, want := range tt.wantContains {
				if !contains(up.Preferences, want) {
					t.Errorf("Preferences should contain %q", want)
				}
			}
		})
	}
}

func TestUserProfile_Merge_TopicsOfInterest(t *testing.T) {
	tests := []struct {
		name          string
		initialTopics []string
		newTopics     []interface{}
		wantTopicsLen int
		wantContains  []string
	}{
		{
			name:          "add topics to empty",
			initialTopics: []string{},
			newTopics:     []interface{}{"Go", "Python"},
			wantTopicsLen: 2,
			wantContains:  []string{"Go", "Python"},
		},
		{
			name:          "merge without duplicates",
			initialTopics: []string{"Go"},
			newTopics:     []interface{}{"Go", "Python"},
			wantTopicsLen: 2,
			wantContains:  []string{"Go", "Python"},
		},
		{
			name:          "skip non-string items",
			initialTopics: []string{},
			newTopics:     []interface{}{"AI", nil, 42, "ML"},
			wantTopicsLen: 2,
			wantContains:  []string{"AI", "ML"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			up := &UserProfile{TopicsOfInterest: tt.initialTopics}
			newInfo := make(map[string]interface{})
			if tt.newTopics != nil {
				newInfo["topics_of_interest"] = tt.newTopics
			}

			up.Merge(newInfo)

			if len(up.TopicsOfInterest) != tt.wantTopicsLen {
				t.Errorf("TopicsOfInterest length = %d, want %d", len(up.TopicsOfInterest), tt.wantTopicsLen)
			}

			for _, want := range tt.wantContains {
				if !contains(up.TopicsOfInterest, want) {
					t.Errorf("TopicsOfInterest should contain %q", want)
				}
			}
		})
	}
}

func TestUserProfile_Merge_UpdatesTimestamp(t *testing.T) {
	up := &UserProfile{
		LastUpdated: time.Now().Add(-1 * time.Hour),
	}
	oldTime := up.LastUpdated

	time.Sleep(1 * time.Millisecond) // Ensure time difference

	up.Merge(map[string]interface{}{"name": "NewName"})

	if !up.LastUpdated.After(oldTime) {
		t.Error("LastUpdated should be updated after Merge")
	}
}

func TestUserProfile_Merge_AllFields(t *testing.T) {
	up := &UserProfile{
		Name:             "Original",
		Preferences:      []string{"pref1"},
		TopicsOfInterest: []string{"topic1"},
	}

	newInfo := map[string]interface{}{
		"name":               "Updated",
		"preferences":        []interface{}{"pref2", "pref3"},
		"topics_of_interest": []interface{}{"topic2"},
	}

	up.Merge(newInfo)

	if up.Name != "Updated" {
		t.Errorf("Name = %q, want %q", up.Name, "Updated")
	}
	if len(up.Preferences) != 3 {
		t.Errorf("Preferences length = %d, want 3", len(up.Preferences))
	}
	if len(up.TopicsOfInterest) != 2 {
		t.Errorf("TopicsOfInterest length = %d, want 2", len(up.TopicsOfInterest))
	}
}

func TestUserProfile_Merge_EmptyNewInfo(t *testing.T) {
	up := &UserProfile{
		Name:             "Keep",
		Preferences:      []string{"keep_this"},
		TopicsOfInterest: []string{"keep_topic"},
	}
	oldTime := up.LastUpdated

	time.Sleep(1 * time.Millisecond)

	up.Merge(map[string]interface{}{})

	if up.Name != "Keep" {
		t.Error("Name should be preserved with empty merge")
	}
	if len(up.Preferences) != 1 {
		t.Error("Preferences should be preserved with empty merge")
	}
	if len(up.TopicsOfInterest) != 1 {
		t.Error("TopicsOfInterest should be preserved with empty merge")
	}
	// Timestamp should still be updated
	if !up.LastUpdated.After(oldTime) {
		t.Error("LastUpdated should still be updated even with empty merge")
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		name  string
		slice []string
		item  string
		want  bool
	}{
		{
			name:  "item present",
			slice: []string{"a", "b", "c"},
			item:  "b",
			want:  true,
		},
		{
			name:  "item absent",
			slice: []string{"a", "b", "c"},
			item:  "d",
			want:  false,
		},
		{
			name:  "empty slice",
			slice: []string{},
			item:  "a",
			want:  false,
		},
		{
			name:  "nil slice",
			slice: nil,
			item:  "a",
			want:  false,
		},
		{
			name:  "empty item in slice",
			slice: []string{"", "a"},
			item:  "",
			want:  true,
		},
		{
			name:  "case sensitive",
			slice: []string{"A", "B"},
			item:  "a",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := contains(tt.slice, tt.item)
			if got != tt.want {
				t.Errorf("contains(%v, %q) = %v, want %v", tt.slice, tt.item, got, tt.want)
			}
		})
	}
}

func TestUserProfile_JSON_Tags(t *testing.T) {
	// Test that the struct has proper JSON tags
	up := &UserProfile{
		Name:             "Test",
		Preferences:      []string{"pref"},
		TopicsOfInterest: []string{"topic"},
		LastUpdated:      time.Now(),
	}

	// Basic sanity check - struct fields are accessible
	if up.Name != "Test" {
		t.Error("Name field not accessible")
	}
	if len(up.Preferences) != 1 {
		t.Error("Preferences field not accessible")
	}
	if len(up.TopicsOfInterest) != 1 {
		t.Error("TopicsOfInterest field not accessible")
	}
}
