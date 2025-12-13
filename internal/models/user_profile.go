// ABOUTME: UserProfile represents user context and preferences
// ABOUTME: Stored in JSON for easy loading and saving
package models

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/adrg/xdg"
)

// UserProfile represents user context and preferences
type UserProfile struct {
	Name             string    `json:"name"`
	Preferences      []string  `json:"preferences,omitempty"`
	TopicsOfInterest []string  `json:"topics_of_interest,omitempty"`
	LastUpdated      time.Time `json:"last_updated"`
}

// LoadUserProfile loads user profile from XDG data directory
func LoadUserProfile() (*UserProfile, error) {
	// Use XDG data directory: ~/.local/share/memory/user_profile.json
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		dataHome = xdg.DataHome
	}
	profilePath := filepath.Join(dataHome, "memory", "user_profile.json")

	data, err := os.ReadFile(profilePath)
	if err != nil {
		if os.IsNotExist(err) {
			// No profile exists yet - return nil without error
			return nil, nil
		}
		return nil, err
	}

	var profile UserProfile
	if err := json.Unmarshal(data, &profile); err != nil {
		return nil, err
	}

	return &profile, nil
}

// Save saves the user profile to XDG data directory
func (up *UserProfile) Save() error {
	// Use XDG data directory
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		dataHome = xdg.DataHome
	}
	basePath := filepath.Join(dataHome, "memory")

	// Ensure directory exists
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return err
	}

	profilePath := filepath.Join(basePath, "user_profile.json")

	// Update last_updated timestamp
	up.LastUpdated = time.Now()

	// Marshal to JSON
	data, err := json.MarshalIndent(up, "", "  ")
	if err != nil {
		return err
	}

	// Write to file
	if err := os.WriteFile(profilePath, data, 0644); err != nil {
		return err
	}

	return nil
}

// Merge intelligently merges new user info into the profile
// Handles updating name, adding preferences without duplicates, and adding topics without duplicates
func (up *UserProfile) Merge(newInfo map[string]interface{}) {
	// Update name if provided
	if name, ok := newInfo["name"].(string); ok && name != "" {
		up.Name = name
	}

	// Merge preferences without duplicates
	if prefs, ok := newInfo["preferences"].([]interface{}); ok {
		for _, pref := range prefs {
			if prefStr, ok := pref.(string); ok {
				if !contains(up.Preferences, prefStr) {
					up.Preferences = append(up.Preferences, prefStr)
				}
			}
		}
	}

	// Merge topics of interest without duplicates
	if topics, ok := newInfo["topics_of_interest"].([]interface{}); ok {
		for _, topic := range topics {
			if topicStr, ok := topic.(string); ok {
				if !contains(up.TopicsOfInterest, topicStr) {
					up.TopicsOfInterest = append(up.TopicsOfInterest, topicStr)
				}
			}
		}
	}

	// Update last_updated timestamp
	up.LastUpdated = time.Now()
}

// contains checks if a string slice contains a specific string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
