// ABOUTME: Memory search result structures for retrieval operations
// ABOUTME: Used by MCP tools to return search results
package models

// MemorySearchResult represents a memory retrieval result
type MemorySearchResult struct {
	BlockID        string  `json:"block_id"`
	TopicLabel     string  `json:"topic_label"`
	RelevanceScore float64 `json:"relevance_score"`
	Summary        string  `json:"summary"`
	Turns          []Turn  `json:"turns,omitempty"`
}
