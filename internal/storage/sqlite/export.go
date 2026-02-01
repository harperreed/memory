// ABOUTME: Export functionality for memory data
// ABOUTME: Supports YAML and Markdown export formats
package sqlite

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// ExportData represents the complete exportable data structure
type ExportData struct {
	Version    string              `yaml:"version" json:"version"`
	ExportedAt string              `yaml:"exported_at" json:"exported_at"`
	Tool       string              `yaml:"tool" json:"tool"`
	Profile    *ExportProfile      `yaml:"profile,omitempty" json:"profile,omitempty"`
	Blocks     []ExportBlock       `yaml:"blocks,omitempty" json:"blocks,omitempty"`
	Facts      []ExportFact        `yaml:"facts,omitempty" json:"facts,omitempty"`
	Embeddings string              `yaml:"embeddings_file,omitempty" json:"embeddings_file,omitempty"`
}

// ExportProfile represents user profile for export
type ExportProfile struct {
	Name             string   `yaml:"name" json:"name"`
	Preferences      []string `yaml:"preferences" json:"preferences"`
	TopicsOfInterest []string `yaml:"topics_of_interest" json:"topics_of_interest"`
}

// ExportBlock represents a bridge block for export
type ExportBlock struct {
	BlockID    string       `yaml:"block_id" json:"block_id"`
	TopicLabel string       `yaml:"topic_label" json:"topic_label"`
	Status     string       `yaml:"status" json:"status"`
	Keywords   []string     `yaml:"keywords,omitempty" json:"keywords,omitempty"`
	Summary    string       `yaml:"summary,omitempty" json:"summary,omitempty"`
	CreatedAt  string       `yaml:"created_at" json:"created_at"`
	Turns      []ExportTurn `yaml:"turns" json:"turns"`
}

// ExportTurn represents a turn for export
type ExportTurn struct {
	TurnID      string `yaml:"turn_id" json:"turn_id"`
	UserMessage string `yaml:"user_message" json:"user_message"`
	AIResponse  string `yaml:"ai_response" json:"ai_response"`
	Timestamp   string `yaml:"timestamp" json:"timestamp"`
}

// ExportFact represents a fact for export
type ExportFact struct {
	FactID     string  `yaml:"fact_id" json:"fact_id"`
	Key        string  `yaml:"key" json:"key"`
	Value      string  `yaml:"value" json:"value"`
	Confidence float64 `yaml:"confidence" json:"confidence"`
	CreatedAt  string  `yaml:"created_at" json:"created_at"`
}

// Export exports all data from storage
func (s *Storage) Export() (*ExportData, error) {
	data := &ExportData{
		Version:    "1.0",
		ExportedAt: time.Now().Format(time.RFC3339),
		Tool:       "memory",
	}

	// Export profile
	profile, err := s.GetUserProfile()
	if err != nil {
		return nil, fmt.Errorf("failed to get profile: %w", err)
	}
	if profile != nil {
		data.Profile = &ExportProfile{
			Name:             profile.Name,
			Preferences:      profile.Preferences,
			TopicsOfInterest: profile.TopicsOfInterest,
		}
	}

	// Export blocks with turns
	blocks, err := s.blocks.ListAll()
	if err != nil {
		return nil, fmt.Errorf("failed to list blocks: %w", err)
	}

	for _, block := range blocks {
		fullBlock, err := s.blocks.GetWithTurns(block.BlockID)
		if err != nil {
			continue
		}

		exportBlock := ExportBlock{
			BlockID:    fullBlock.BlockID,
			TopicLabel: fullBlock.TopicLabel,
			Status:     string(fullBlock.Status),
			Keywords:   fullBlock.Keywords,
			Summary:    fullBlock.Summary,
			CreatedAt:  fullBlock.CreatedAt.Format(time.RFC3339),
			Turns:      make([]ExportTurn, 0, len(fullBlock.Turns)),
		}

		for _, turn := range fullBlock.Turns {
			exportBlock.Turns = append(exportBlock.Turns, ExportTurn{
				TurnID:      turn.TurnID,
				UserMessage: turn.UserMessage,
				AIResponse:  turn.AIResponse,
				Timestamp:   turn.Timestamp.Format(time.RFC3339),
			})
		}

		data.Blocks = append(data.Blocks, exportBlock)
	}

	// Export facts (without block reference for orphaned facts)
	allFacts := []ExportFact{}
	rows, err := s.db.Query(`
		SELECT id, key, value, confidence, created_at
		FROM facts
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query facts: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var fact ExportFact
		var createdAt time.Time
		if err := rows.Scan(&fact.FactID, &fact.Key, &fact.Value, &fact.Confidence, &createdAt); err != nil {
			continue
		}
		fact.CreatedAt = createdAt.Format(time.RFC3339)
		allFacts = append(allFacts, fact)
	}
	data.Facts = allFacts

	return data, nil
}

// ExportToYAML exports data to a YAML file
func (s *Storage) ExportToYAML(outputPath string) error {
	data, err := s.Export()
	if err != nil {
		return err
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	file, err := os.Create(outputPath) // #nosec G304
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer func() { _ = file.Close() }()

	encoder := yaml.NewEncoder(file)
	encoder.SetIndent(2)
	if err := encoder.Encode(data); err != nil {
		return fmt.Errorf("failed to encode YAML: %w", err)
	}

	return nil
}

// ExportToMarkdown exports data to a Markdown file
func (s *Storage) ExportToMarkdown(outputPath string) error {
	data, err := s.Export()
	if err != nil {
		return err
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	file, err := os.Create(outputPath) // #nosec G304
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer func() { _ = file.Close() }()

	// Write header
	_, _ = fmt.Fprintf(file, "# Memory Export - %s\n\n", time.Now().Format("2006-01-02"))
	_, _ = fmt.Fprintf(file, "Generated: %s\n\n", data.ExportedAt)

	// Write profile
	if data.Profile != nil {
		_, _ = fmt.Fprintln(file, "## User Profile")
		_, _ = fmt.Fprintln(file)
		_, _ = fmt.Fprintf(file, "- **Name:** %s\n", data.Profile.Name)
		if len(data.Profile.Preferences) > 0 {
			_, _ = fmt.Fprintln(file, "- **Preferences:**")
			for _, pref := range data.Profile.Preferences {
				_, _ = fmt.Fprintf(file, "  - %s\n", pref)
			}
		}
		if len(data.Profile.TopicsOfInterest) > 0 {
			_, _ = fmt.Fprintln(file, "- **Topics of Interest:**")
			for _, topic := range data.Profile.TopicsOfInterest {
				_, _ = fmt.Fprintf(file, "  - %s\n", topic)
			}
		}
		_, _ = fmt.Fprintln(file)
	}

	// Write facts
	if len(data.Facts) > 0 {
		_, _ = fmt.Fprintln(file, "## Facts")
		_, _ = fmt.Fprintln(file)
		_, _ = fmt.Fprintln(file, "| Key | Value | Confidence |")
		_, _ = fmt.Fprintln(file, "|-----|-------|------------|")
		for _, fact := range data.Facts {
			_, _ = fmt.Fprintf(file, "| %s | %s | %.2f |\n", fact.Key, fact.Value, fact.Confidence)
		}
		_, _ = fmt.Fprintln(file)
	}

	// Write conversations
	if len(data.Blocks) > 0 {
		_, _ = fmt.Fprintln(file, "## Conversations")
		_, _ = fmt.Fprintln(file)
		for _, block := range data.Blocks {
			_, _ = fmt.Fprintf(file, "### %s (%s)\n\n", block.TopicLabel, block.Status)
			if len(block.Keywords) > 0 {
				_, _ = fmt.Fprintf(file, "*Keywords: %s*\n\n", formatKeywords(block.Keywords))
			}
			for _, turn := range block.Turns {
				_, _ = fmt.Fprintf(file, "**User:** %s\n\n", turn.UserMessage)
				if turn.AIResponse != "" {
					_, _ = fmt.Fprintf(file, "**AI:** %s\n\n", turn.AIResponse)
				}
			}
			_, _ = fmt.Fprintln(file, "---")
			_, _ = fmt.Fprintln(file)
		}
	}

	return nil
}

// ExportEmbeddingsToJSON exports embeddings to a separate JSON file
func (s *Storage) ExportEmbeddingsToJSON(outputPath string) error {
	rows, err := s.db.Query(`
		SELECT chunk_id, turn_id, block_id, vector, created_at
		FROM embeddings
	`)
	if err != nil {
		return fmt.Errorf("failed to query embeddings: %w", err)
	}
	defer func() { _ = rows.Close() }()

	type EmbeddingExport struct {
		ChunkID   string    `json:"chunk_id"`
		TurnID    string    `json:"turn_id"`
		BlockID   string    `json:"block_id"`
		Vector    []float64 `json:"vector"`
		CreatedAt string    `json:"created_at"`
	}

	var embeddings []EmbeddingExport
	for rows.Next() {
		var (
			emb       EmbeddingExport
			blob      []byte
			createdAt time.Time
		)
		if err := rows.Scan(&emb.ChunkID, &emb.TurnID, &emb.BlockID, &blob, &createdAt); err != nil {
			continue
		}
		emb.Vector = blobToVector(blob)
		emb.CreatedAt = createdAt.Format(time.RFC3339)
		embeddings = append(embeddings, emb)
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	file, err := os.Create(outputPath) // #nosec G304
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer func() { _ = file.Close() }()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(embeddings); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}

	return nil
}

func formatKeywords(keywords []string) string {
	result := ""
	for i, kw := range keywords {
		if i > 0 {
			result += ", "
		}
		result += kw
	}
	return result
}
