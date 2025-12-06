// ABOUTME: Routing decision types and structures for Governor
// ABOUTME: Defines the 4 routing scenarios for topic management
package models

// RoutingScenario represents the routing decision type
type RoutingScenario string

const (
	// TopicContinuation - Same topic as last active block → append turn
	TopicContinuation RoutingScenario = "topic_continuation"

	// TopicResumption - Match old paused block → reactivate it, pause current
	TopicResumption RoutingScenario = "topic_resumption"

	// NewTopicFirst - No active blocks → create new block
	NewTopicFirst RoutingScenario = "new_topic_first"

	// TopicShift - New topic while one is active → pause old, create new
	TopicShift RoutingScenario = "topic_shift"
)

// RoutingDecision contains the routing decision and relevant metadata
type RoutingDecision struct {
	Scenario       RoutingScenario `json:"scenario"`
	MatchedBlockID string          `json:"matched_block_id,omitempty"`
	ActiveBlockID  string          `json:"active_block_id,omitempty"`
}
