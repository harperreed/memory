// ABOUTME: UserProfile represents user context and preferences
// ABOUTME: Stored in SQLite via the Storage layer
package models

import (
	"time"
)

// UserProfile represents user context and preferences
type UserProfile struct {
	Name             string    `json:"name"`
	Preferences      []string  `json:"preferences,omitempty"`
	TopicsOfInterest []string  `json:"topics_of_interest,omitempty"`
	LastUpdated      time.Time `json:"last_updated"`
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
