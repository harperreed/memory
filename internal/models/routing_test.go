// ABOUTME: Tests for RoutingScenario and RoutingDecision types
// ABOUTME: Verifies routing decision validation

package models

import "testing"

func TestRoutingScenario_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		scenario RoutingScenario
		want     bool
	}{
		{"TopicContinuation", TopicContinuation, true},
		{"TopicResumption", TopicResumption, true},
		{"NewTopicFirst", NewTopicFirst, true},
		{"TopicShift", TopicShift, true},
		{"empty string", RoutingScenario(""), false},
		{"invalid scenario", RoutingScenario("invalid"), false},
		{"close but wrong", RoutingScenario("topic_continue"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.scenario.IsValid()
			if got != tt.want {
				t.Errorf("IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRoutingScenario_Constants(t *testing.T) {
	// Verify the string values of constants
	if TopicContinuation != "topic_continuation" {
		t.Errorf("TopicContinuation = %q, want %q", TopicContinuation, "topic_continuation")
	}
	if TopicResumption != "topic_resumption" {
		t.Errorf("TopicResumption = %q, want %q", TopicResumption, "topic_resumption")
	}
	if NewTopicFirst != "new_topic_first" {
		t.Errorf("NewTopicFirst = %q, want %q", NewTopicFirst, "new_topic_first")
	}
	if TopicShift != "topic_shift" {
		t.Errorf("TopicShift = %q, want %q", TopicShift, "topic_shift")
	}
}

func TestRoutingDecision_Fields(t *testing.T) {
	decision := RoutingDecision{
		Scenario:       TopicContinuation,
		MatchedBlockID: "block_123",
		ActiveBlockID:  "block_456",
	}

	if decision.Scenario != TopicContinuation {
		t.Errorf("Scenario = %v, want TopicContinuation", decision.Scenario)
	}
	if decision.MatchedBlockID != "block_123" {
		t.Errorf("MatchedBlockID = %q, want %q", decision.MatchedBlockID, "block_123")
	}
	if decision.ActiveBlockID != "block_456" {
		t.Errorf("ActiveBlockID = %q, want %q", decision.ActiveBlockID, "block_456")
	}
}

func TestRoutingDecision_EmptyOptionalFields(t *testing.T) {
	// Test with optional fields empty (e.g., NewTopicFirst scenario)
	decision := RoutingDecision{
		Scenario:       NewTopicFirst,
		MatchedBlockID: "",
		ActiveBlockID:  "",
	}

	if !decision.Scenario.IsValid() {
		t.Error("Scenario should be valid")
	}
	if decision.MatchedBlockID != "" {
		t.Error("MatchedBlockID should be empty for NewTopicFirst")
	}
	if decision.ActiveBlockID != "" {
		t.Error("ActiveBlockID should be empty for NewTopicFirst")
	}
}
