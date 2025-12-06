// ABOUTME: Scribe agent for async user profile learning
// ABOUTME: Runs in background to extract and update user information from conversations
package core

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/harper/remember-standalone/internal/models"
	"github.com/harper/remember-standalone/internal/storage"
	openai "github.com/sashabaranov/go-openai"
)

// Scribe is an async background agent that learns about the user from conversations
type Scribe struct {
	client     *openai.Client
	maxRetries int
	retryDelay time.Duration
	mu         sync.Mutex // Protects concurrent profile updates
}

// OpenAIClient interface for LLM operations
type OpenAIClient interface {
	CreateChatCompletion(ctx context.Context, req openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error)
}

// NewScribe creates a new Scribe agent
func NewScribe(client interface{}) *Scribe {
	// Extract OpenAI client
	type clientWrapper interface {
		GetClient() *openai.Client
	}

	var oaiClient *openai.Client
	if wrapper, ok := client.(clientWrapper); ok {
		oaiClient = wrapper.GetClient()
	} else {
		// Assume it's already an OpenAI client
		oaiClient = client.(*openai.Client)
	}

	return &Scribe{
		client:     oaiClient,
		maxRetries: 3,
		retryDelay: time.Second * 2,
	}
}

// UpdateProfileAsync runs Scribe in a goroutine (fire-and-forget)
// Analyzes user message and updates profile asynchronously
func (s *Scribe) UpdateProfileAsync(userMessage string, profile *models.UserProfile, store *storage.Storage) {
	// This method is typically called with `go scribe.UpdateProfileAsync(...)`
	// Run the actual update logic
	if err := s.updateProfile(userMessage, profile, store); err != nil {
		// Log error but don't crash - this is async background work
		log.Printf("[Scribe] Error updating profile: %v", err)
	}
}

// updateProfile is the internal sync implementation
func (s *Scribe) updateProfile(userMessage string, profile *models.UserProfile, store *storage.Storage) error {
	// Skip empty messages
	if strings.TrimSpace(userMessage) == "" {
		return nil
	}

	// Extract user info using LLM
	userInfo, err := s.extractUserInfo(userMessage)
	if err != nil {
		return fmt.Errorf("failed to extract user info: %w", err)
	}

	// Lock for concurrent updates
	s.mu.Lock()
	defer s.mu.Unlock()

	// Reload profile from disk (in case it was updated by another goroutine)
	currentProfile, err := store.GetUserProfile()
	if err != nil {
		return fmt.Errorf("failed to load current profile: %w", err)
	}

	// If no profile exists, use the one passed in
	if currentProfile == nil {
		currentProfile = profile
	}

	// Merge new info into profile
	currentProfile.Merge(userInfo)

	// Save updated profile
	if err := store.SaveUserProfile(currentProfile); err != nil {
		return fmt.Errorf("failed to save profile: %w", err)
	}

	log.Printf("[Scribe] Profile updated successfully")
	return nil
}

// extractUserInfo uses GPT-4o-mini to extract user information from message
func (s *Scribe) extractUserInfo(userMessage string) (map[string]interface{}, error) {
	systemPrompt := `You are a user profile learning assistant. Analyze the user's message and extract information about them.

Extract the following if present:
- name: user's name (string)
- preferences: list of preferences, habits, or ways they like to work (array of strings)
- topics_of_interest: subjects, technologies, or areas they're interested in (array of strings)

Return ONLY a JSON object with these fields. Only include fields that are actually mentioned.
Example: {"name": "Alice", "preferences": ["TDD", "simple solutions"], "topics_of_interest": ["AI", "distributed systems"]}

If nothing is found, return an empty object: {}`

	userPrompt := fmt.Sprintf("Extract user information from this message:\n\n%s", userMessage)

	var lastErr error

	for attempt := 0; attempt <= s.maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(s.retryDelay * time.Duration(attempt))
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		resp, err := s.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
			Model: openai.GPT4oMini,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: systemPrompt,
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: userPrompt,
				},
			},
			Temperature: 0.2, // Low temperature for consistent extraction
		})

		if err != nil {
			lastErr = fmt.Errorf("attempt %d: %w", attempt+1, err)
			continue
		}

		if len(resp.Choices) == 0 {
			lastErr = fmt.Errorf("attempt %d: no completion choices returned", attempt+1)
			continue
		}

		content := resp.Choices[0].Message.Content

		// Parse JSON response
		var userInfo map[string]interface{}
		if err := json.Unmarshal([]byte(content), &userInfo); err != nil {
			lastErr = fmt.Errorf("attempt %d: failed to parse JSON: %w", attempt+1, err)
			continue
		}

		return userInfo, nil
	}

	return nil, fmt.Errorf("failed to extract user info after %d attempts: %w", s.maxRetries+1, lastErr)
}
