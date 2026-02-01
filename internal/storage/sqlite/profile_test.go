// ABOUTME: Tests for user profile storage operations
// ABOUTME: Verifies CRUD operations for user profile data
package sqlite

import (
	"testing"
	"time"

	"github.com/harper/remember-standalone/internal/models"
)

func TestProfileCRUD(t *testing.T) {
	db, err := OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory() error = %v", err)
	}
	defer func() { _ = db.Close() }()

	store := NewProfileStore(db)

	// Test GetUserProfile returns nil when no profile exists
	profile, err := store.Get()
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if profile != nil {
		t.Error("Get() should return nil when no profile exists")
	}

	// Test SaveUserProfile
	newProfile := &models.UserProfile{
		Name:             "Harper",
		Preferences:      []string{"dark mode", "vim keybindings"},
		TopicsOfInterest: []string{"Go programming", "distributed systems"},
		LastUpdated:      time.Now(),
	}

	err = store.Save(newProfile)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Test GetUserProfile returns saved profile
	retrieved, err := store.Get()
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if retrieved == nil {
		t.Fatal("Get() returned nil after Save()")
	}

	if retrieved.Name != "Harper" {
		t.Errorf("Name = %v, want Harper", retrieved.Name)
	}
	if len(retrieved.Preferences) != 2 {
		t.Errorf("Preferences length = %v, want 2", len(retrieved.Preferences))
	}
	if len(retrieved.TopicsOfInterest) != 2 {
		t.Errorf("TopicsOfInterest length = %v, want 2", len(retrieved.TopicsOfInterest))
	}

	// Test update profile
	retrieved.Name = "Doctor Biz"
	retrieved.Preferences = append(retrieved.Preferences, "TDD")
	err = store.Save(retrieved)
	if err != nil {
		t.Fatalf("Save() update error = %v", err)
	}

	updated, err := store.Get()
	if err != nil {
		t.Fatalf("Get() after update error = %v", err)
	}
	if updated.Name != "Doctor Biz" {
		t.Errorf("Name = %v, want Doctor Biz", updated.Name)
	}
	if len(updated.Preferences) != 3 {
		t.Errorf("Preferences length = %v, want 3", len(updated.Preferences))
	}
}

func TestProfileEmptyArrays(t *testing.T) {
	db, err := OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory() error = %v", err)
	}
	defer func() { _ = db.Close() }()

	store := NewProfileStore(db)

	profile := &models.UserProfile{
		Name:             "Test",
		Preferences:      []string{},
		TopicsOfInterest: []string{},
		LastUpdated:      time.Now(),
	}

	err = store.Save(profile)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	retrieved, err := store.Get()
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if retrieved.Name != "Test" {
		t.Errorf("Name = %v, want Test", retrieved.Name)
	}
	// Empty arrays should be preserved (not nil)
	if retrieved.Preferences == nil {
		t.Error("Preferences should not be nil")
	}
	if retrieved.TopicsOfInterest == nil {
		t.Error("TopicsOfInterest should not be nil")
	}
}

func TestProfileNilArrays(t *testing.T) {
	db, err := OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory() error = %v", err)
	}
	defer func() { _ = db.Close() }()

	store := NewProfileStore(db)

	profile := &models.UserProfile{
		Name:        "NilTest",
		LastUpdated: time.Now(),
		// Preferences and TopicsOfInterest are nil
	}

	err = store.Save(profile)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	retrieved, err := store.Get()
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if retrieved.Name != "NilTest" {
		t.Errorf("Name = %v, want NilTest", retrieved.Name)
	}
}
