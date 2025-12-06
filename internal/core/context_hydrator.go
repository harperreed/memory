// ABOUTME: ContextHydrator assembles intelligent prompts with conversation history, memories, and facts
// ABOUTME: Uses semantic search to retrieve relevant context and enforces token limits
package core

import (
	"fmt"
	"strings"

	"github.com/harper/remember-standalone/internal/models"
	"github.com/harper/remember-standalone/internal/storage"
)

// ContextHydrator assembles context-aware prompts for LLM interactions
type ContextHydrator struct {
	storage       *storage.Storage
	vectorStorage interface {
		GenerateEmbedding(text string) ([]float64, error)
	}
}

// NewContextHydrator creates a new ContextHydrator
func NewContextHydrator(store *storage.Storage, embeddingClient interface {
	GenerateEmbedding(text string) ([]float64, error)
}) *ContextHydrator {
	return &ContextHydrator{
		storage:       store,
		vectorStorage: embeddingClient,
	}
}

// HydrateBridgeBlock assembles a complete prompt for a Bridge Block conversation
// Includes: system prompt, user profile, block history, retrieved memories, relevant facts, and current message
func (ch *ContextHydrator) HydrateBridgeBlock(blockID string, userMessage string, maxTokens int) (string, error) {
	var sections []string

	// 1. System prompt (always included)
	systemPrompt := "You are a helpful AI assistant with access to conversation history and context."
	sections = append(sections, "SYSTEM:\n"+systemPrompt+"\n")

	// 2. User profile (if available)
	profile, err := ch.storage.GetUserProfile()
	if err == nil && profile != nil {
		profileSection := ch.formatUserProfile(profile)
		sections = append(sections, profileSection)
	}

	// 3. Bridge Block conversation history
	block, err := ch.storage.GetBridgeBlock(blockID)
	if err != nil {
		return "", fmt.Errorf("failed to get bridge block: %w", err)
	}

	blockHistory := ch.formatBlockHistory(block)
	sections = append(sections, blockHistory)

	// 4. Retrieved memories from other blocks (via semantic search)
	if ch.vectorStorage != nil {
		memories, err := ch.storage.SearchMemory(userMessage, 3)
		if err == nil && len(memories) > 0 {
			// Filter out current block from memories
			var relevantMemories []models.MemorySearchResult
			for _, mem := range memories {
				if mem.BlockID != blockID {
					relevantMemories = append(relevantMemories, mem)
				}
			}

			if len(relevantMemories) > 0 {
				memoriesSection := ch.formatRetrievedMemories(relevantMemories)
				sections = append(sections, memoriesSection)
			}
		}
	}

	// 5. Relevant facts (search by keywords from user message)
	facts, err := ch.storage.SearchFacts(userMessage, 5)
	if err == nil && len(facts) > 0 {
		factsSection := ch.formatRelevantFacts(facts)
		sections = append(sections, factsSection)
	}

	// 6. Current user message (always included at the end)
	sections = append(sections, "CURRENT USER MESSAGE:\n"+userMessage+"\n")

	// Assemble full prompt
	fullPrompt := strings.Join(sections, "\n")

	// Token limiting (4 chars ≈ 1 token)
	fullPrompt = ch.limitTokens(fullPrompt, userMessage, maxTokens)

	return fullPrompt, nil
}

// formatUserProfile formats user profile for prompt
func (ch *ContextHydrator) formatUserProfile(profile *models.UserProfile) string {
	var sb strings.Builder
	sb.WriteString("USER PROFILE:\n")
	sb.WriteString(fmt.Sprintf("Name: %s\n", profile.Name))

	if len(profile.Preferences) > 0 {
		sb.WriteString(fmt.Sprintf("Preferences: %s\n", strings.Join(profile.Preferences, ", ")))
	}

	if len(profile.TopicsOfInterest) > 0 {
		sb.WriteString(fmt.Sprintf("Topics of Interest: %s\n", strings.Join(profile.TopicsOfInterest, ", ")))
	}

	sb.WriteString("\n")
	return sb.String()
}

// formatBlockHistory formats Bridge Block conversation history
func (ch *ContextHydrator) formatBlockHistory(block *models.BridgeBlock) string {
	var sb strings.Builder
	sb.WriteString("CONVERSATION HISTORY:\n")
	sb.WriteString(fmt.Sprintf("Topic: %s\n\n", block.TopicLabel))

	for i, turn := range block.Turns {
		sb.WriteString(fmt.Sprintf("Turn %d:\n", i+1))
		sb.WriteString(fmt.Sprintf("User: %s\n", turn.UserMessage))
		sb.WriteString(fmt.Sprintf("AI: %s\n\n", turn.AIResponse))
	}

	return sb.String()
}

// formatRetrievedMemories formats retrieved memories from other blocks
func (ch *ContextHydrator) formatRetrievedMemories(memories []models.MemorySearchResult) string {
	var sb strings.Builder
	sb.WriteString("RETRIEVED MEMORIES (from other conversations):\n")

	for i, mem := range memories {
		sb.WriteString(fmt.Sprintf("\nMemory %d (Relevance: %.2f):\n", i+1, mem.RelevanceScore))
		sb.WriteString(fmt.Sprintf("Topic: %s\n", mem.TopicLabel))

		// Include summary if available, otherwise include first turn
		if mem.Summary != "" {
			sb.WriteString(fmt.Sprintf("Summary: %s\n", mem.Summary))
		} else if len(mem.Turns) > 0 {
			turn := mem.Turns[0]
			sb.WriteString(fmt.Sprintf("User: %s\n", turn.UserMessage))
			sb.WriteString(fmt.Sprintf("AI: %s\n", turn.AIResponse))
		}
	}

	sb.WriteString("\n")
	return sb.String()
}

// formatRelevantFacts formats relevant facts
func (ch *ContextHydrator) formatRelevantFacts(facts []models.Fact) string {
	var sb strings.Builder
	sb.WriteString("RELEVANT FACTS:\n")

	for _, fact := range facts {
		sb.WriteString(fmt.Sprintf("- %s: %s (confidence: %.2f)\n", fact.Key, fact.Value, fact.Confidence))
	}

	sb.WriteString("\n")
	return sb.String()
}

// limitTokens enforces token limit by trimming sections
// Prioritizes: system prompt > current message > block history > memories > facts > profile
func (ch *ContextHydrator) limitTokens(fullPrompt string, userMessage string, maxTokens int) string {
	// Token approximation: 4 chars ≈ 1 token
	maxChars := maxTokens * 4

	if len(fullPrompt) <= maxChars {
		return fullPrompt
	}

	// If we're over limit, rebuild with priorities
	// Essential sections (always include):
	systemPrompt := "SYSTEM:\nYou are a helpful AI assistant with access to conversation history and context.\n\n"
	currentMessage := "CURRENT USER MESSAGE:\n" + userMessage + "\n"
	essentialChars := len(systemPrompt) + len(currentMessage)

	if essentialChars > maxChars {
		// Even essential content is too large - just return system + truncated message
		remaining := maxChars - len(systemPrompt)
		if remaining > 100 {
			truncatedMessage := userMessage[:remaining-50] + "... [truncated]"
			return systemPrompt + "CURRENT USER MESSAGE:\n" + truncatedMessage + "\n"
		}
		return systemPrompt
	}

	// Build prompt section by section until we hit the limit
	availableChars := maxChars - essentialChars
	var optionalSections []string

	// Extract sections from original prompt
	if strings.Contains(fullPrompt, "CONVERSATION HISTORY:") {
		start := strings.Index(fullPrompt, "CONVERSATION HISTORY:")
		end := strings.Index(fullPrompt[start:], "\nRETRIEVED MEMORIES")
		if end == -1 {
			end = strings.Index(fullPrompt[start:], "\nRELEVANT FACTS")
		}
		if end == -1 {
			end = strings.Index(fullPrompt[start:], "\nCURRENT USER MESSAGE")
		}
		if end != -1 {
			section := fullPrompt[start : start+end+1]
			if len(section) <= availableChars {
				optionalSections = append(optionalSections, section)
				availableChars -= len(section)
			}
		}
	}

	if strings.Contains(fullPrompt, "RETRIEVED MEMORIES") && availableChars > 0 {
		start := strings.Index(fullPrompt, "RETRIEVED MEMORIES")
		end := strings.Index(fullPrompt[start:], "\nRELEVANT FACTS")
		if end == -1 {
			end = strings.Index(fullPrompt[start:], "\nCURRENT USER MESSAGE")
		}
		if end != -1 {
			section := fullPrompt[start : start+end+1]
			if len(section) <= availableChars {
				optionalSections = append(optionalSections, section)
				availableChars -= len(section)
			}
		}
	}

	if strings.Contains(fullPrompt, "RELEVANT FACTS") && availableChars > 0 {
		start := strings.Index(fullPrompt, "RELEVANT FACTS")
		end := strings.Index(fullPrompt[start:], "\nCURRENT USER MESSAGE")
		if end != -1 {
			section := fullPrompt[start : start+end+1]
			if len(section) <= availableChars {
				optionalSections = append(optionalSections, section)
				availableChars -= len(section)
			}
		}
	}

	if strings.Contains(fullPrompt, "USER PROFILE") && availableChars > 0 {
		start := strings.Index(fullPrompt, "USER PROFILE")
		end := strings.Index(fullPrompt[start:], "\nCONVERSATION HISTORY")
		if end == -1 {
			end = strings.Index(fullPrompt[start:], "\nRETRIEVED MEMORIES")
		}
		if end != -1 {
			section := fullPrompt[start : start+end+1]
			if len(section) <= availableChars {
				optionalSections = append(optionalSections, section)
			}
		}
	}

	// Assemble final prompt
	result := systemPrompt
	for _, section := range optionalSections {
		result += section
	}
	result += currentMessage

	return result
}
