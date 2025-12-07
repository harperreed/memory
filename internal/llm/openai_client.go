// ABOUTME: OpenAI client for embeddings and LLM-based extraction
// ABOUTME: Uses text-embedding-3-small for embeddings, gpt-4o-mini for structured extraction
package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/harper/remember-standalone/internal/models"
	openai "github.com/sashabaranov/go-openai"
)

// OpenAIClient wraps the OpenAI API client with retry logic
type OpenAIClient struct {
	client     *openai.Client
	maxRetries int
	retryDelay time.Duration
}

// NewOpenAIClient creates a new OpenAI client with the given API key
func NewOpenAIClient(apiKey string) (*OpenAIClient, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("OpenAI API key is required")
	}

	client := openai.NewClient(apiKey)

	return &OpenAIClient{
		client:     client,
		maxRetries: 3,
		retryDelay: time.Second * 2,
	}, nil
}

// GetClient returns the underlying OpenAI client for direct use
func (c *OpenAIClient) GetClient() *openai.Client {
	return c.client
}

// GenerateEmbedding generates a 1536-dimensional embedding vector using text-embedding-3-small
func (c *OpenAIClient) GenerateEmbedding(text string) ([]float64, error) {
	var lastErr error

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(c.retryDelay * time.Duration(attempt))
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		resp, err := c.client.CreateEmbeddings(ctx, openai.EmbeddingRequestStrings{
			Input: []string{text},
			Model: openai.SmallEmbedding3,
		})

		if err != nil {
			lastErr = fmt.Errorf("attempt %d: %w", attempt+1, err)
			continue
		}

		if len(resp.Data) == 0 {
			lastErr = fmt.Errorf("attempt %d: no embeddings returned", attempt+1)
			continue
		}

		// Convert []float32 to []float64
		embedding32 := resp.Data[0].Embedding
		embedding64 := make([]float64, len(embedding32))
		for i, v := range embedding32 {
			embedding64[i] = float64(v)
		}

		return embedding64, nil
	}

	return nil, fmt.Errorf("failed to generate embedding after %d attempts: %w", c.maxRetries+1, lastErr)
}

// ExtractMetadata uses gpt-4o-mini to extract keywords, topics, and affect from conversation text
func (c *OpenAIClient) ExtractMetadata(text string) (map[string]interface{}, error) {
	systemPrompt := `You are a metadata extraction assistant. Given a conversation, extract:
1. keywords: Important terms and concepts (array of strings)
2. topics: High-level subjects discussed (array of strings)
3. affect: Overall emotional tone/sentiment (string: positive, negative, neutral, mixed)

Return ONLY a JSON object with these three fields. No additional text.`

	userPrompt := fmt.Sprintf("Extract metadata from this conversation:\n\n%s", text)

	var lastErr error

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(c.retryDelay * time.Duration(attempt))
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		resp, err := c.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
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
			Temperature: 0.3,
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
		var metadata map[string]interface{}
		if err := json.Unmarshal([]byte(content), &metadata); err != nil {
			lastErr = fmt.Errorf("attempt %d: failed to parse JSON: %w", attempt+1, err)
			continue
		}

		return metadata, nil
	}

	return nil, fmt.Errorf("failed to extract metadata after %d attempts: %w", c.maxRetries+1, lastErr)
}

// ExtractFacts uses gpt-4o-mini to extract key-value facts from conversation text
func (c *OpenAIClient) ExtractFacts(text string) ([]models.Fact, error) {
	systemPrompt := `You are a fact extraction assistant. Given a conversation, extract ALL factual key-value pairs.

Extract facts like:
- name: user's name
- company: where they work
- project: what they're working on
- favorite_language: programming language preference
- location: city/country
- role: job title
- api_key, weather_api_key, stripe_api_key: API keys and credentials
- email, phone: contact information
- dietary_preference: food preferences
- Any other factual information explicitly stated

For each fact, provide:
- key: descriptive fact name (lowercase, underscores). For API keys, include service name (e.g., "weather_api_key")
- value: the actual value
- confidence: 0.0 to 1.0 (how certain you are)

Return ONLY a JSON array of fact objects. Each object must have: key, value, confidence.
Example: [{"key": "weather_api_key", "value": "ABC123XYZ", "confidence": 1.0}]

Extract EVERY fact explicitly stated. Do not infer or assume.`

	userPrompt := fmt.Sprintf("Extract facts from this conversation:\n\n%s", text)

	var lastErr error

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(c.retryDelay * time.Duration(attempt))
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		resp, err := c.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
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
			Temperature: 0.1, // Low temperature for factual extraction
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

		// Parse JSON response into temporary struct
		type FactResponse struct {
			Key        string  `json:"key"`
			Value      string  `json:"value"`
			Confidence float64 `json:"confidence"`
		}

		var factResponses []FactResponse
		if err := json.Unmarshal([]byte(content), &factResponses); err != nil {
			lastErr = fmt.Errorf("attempt %d: failed to parse JSON: %w", attempt+1, err)
			continue
		}

		// Convert to models.Fact (without IDs and timestamps - those will be added by storage layer)
		facts := make([]models.Fact, len(factResponses))
		for i, fr := range factResponses {
			facts[i] = models.Fact{
				Key:        fr.Key,
				Value:      fr.Value,
				Confidence: fr.Confidence,
			}
		}

		return facts, nil
	}

	return nil, fmt.Errorf("failed to extract facts after %d attempts: %w", c.maxRetries+1, lastErr)
}
