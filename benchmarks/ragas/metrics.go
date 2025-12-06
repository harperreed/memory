// ABOUTME: RAGAS metrics implementation for faithfulness and context recall
// ABOUTME: Simplified deterministic evaluation based on ground truth comparison

package ragas

import (
	"fmt"
	"strings"
)

// MetricsCalculator computes RAGAS scores for benchmark tests
type MetricsCalculator struct{}

// NewMetricsCalculator creates a new metrics calculator
func NewMetricsCalculator() *MetricsCalculator {
	return &MetricsCalculator{}
}

// CalculateFaithfulness computes faithfulness score (0.0-1.0)
// Faithfulness = Does the response match retrieved context? No hallucinations?
func (m *MetricsCalculator) CalculateFaithfulness(
	response string,
	expectedInResponse []string,
	forbiddenInResponse []string,
) (float64, string) {
	responseUpper := strings.ToUpper(response)

	// Check all expected items are present
	missingItems := []string{}
	for _, expected := range expectedInResponse {
		if !strings.Contains(responseUpper, strings.ToUpper(expected)) {
			missingItems = append(missingItems, expected)
		}
	}

	// Check no forbidden items are present
	forbiddenFound := []string{}
	for _, forbidden := range forbiddenInResponse {
		if strings.Contains(responseUpper, strings.ToUpper(forbidden)) {
			forbiddenFound = append(forbiddenFound, forbidden)
		}
	}

	// Calculate score
	// Perfect score (1.0) requires all expected items AND no forbidden items
	if len(missingItems) == 0 && len(forbiddenFound) == 0 {
		return 1.0, "Perfect faithfulness - response matches expected ground truth"
	}

	// Partial failure
	if len(missingItems) > 0 && len(forbiddenFound) > 0 {
		return 0.0, fmt.Sprintf(
			"Faithfulness failure - missing expected items: %v, forbidden items found: %v",
			missingItems, forbiddenFound,
		)
	}

	if len(missingItems) > 0 {
		return 0.5, fmt.Sprintf(
			"Partial faithfulness - missing expected items: %v",
			missingItems,
		)
	}

	if len(forbiddenFound) > 0 {
		return 0.5, fmt.Sprintf(
			"Partial faithfulness - forbidden items found: %v",
			forbiddenFound,
		)
	}

	return 1.0, "Faithfulness verified"
}

// CalculateContextRecall computes context recall score (0.0-1.0)
// Context Recall = Was the correct context retrieved from memory?
func (m *MetricsCalculator) CalculateContextRecall(
	retrievedContext []string,
	expectedContextItems []string,
) (float64, string) {
	if len(expectedContextItems) == 0 {
		return 1.0, "No context retrieval required"
	}

	// Join all retrieved context for searching
	allContext := strings.ToUpper(strings.Join(retrievedContext, " "))

	// Check how many expected items were retrieved
	foundCount := 0
	missingItems := []string{}

	for _, expectedItem := range expectedContextItems {
		if strings.Contains(allContext, strings.ToUpper(expectedItem)) {
			foundCount++
		} else {
			missingItems = append(missingItems, expectedItem)
		}
	}

	// Calculate recall as proportion of expected items found
	recall := float64(foundCount) / float64(len(expectedContextItems))

	if recall == 1.0 {
		return 1.0, "Perfect context recall - all expected items retrieved"
	}

	return recall, fmt.Sprintf(
		"Partial context recall (%.2f) - missing items: %v",
		recall, missingItems,
	)
}

// EvaluateTest runs full RAGAS evaluation for a test
func (m *MetricsCalculator) EvaluateTest(
	scenario TestScenario,
	finalResponse string,
	retrievedContext []string,
) TestResult {
	// Calculate faithfulness
	faithfulness, faithfulnessDetail := m.CalculateFaithfulness(
		finalResponse,
		scenario.GroundTruth.ExpectedInResponse,
		scenario.GroundTruth.ForbiddenInResponse,
	)

	// Calculate context recall
	recall, recallDetail := m.CalculateContextRecall(
		retrievedContext,
		scenario.GroundTruth.ExpectedContextItems,
	)

	// Calculate overall score
	overallScore := (faithfulness + recall) / 2.0

	// Determine pass/fail status
	// For production memory system, we require >= 0.9 on both metrics
	status := "FAIL"
	if faithfulness >= 0.9 && recall >= 0.9 {
		status = "PASS"
	}

	return TestResult{
		TestID:             scenario.ID,
		TestName:           scenario.Name,
		FaithfulnessScore:  faithfulness,
		ContextRecallScore: recall,
		OverallScore:       overallScore,
		Status:             status,
		Details: map[string]interface{}{
			"faithfulness_detail": faithfulnessDetail,
			"recall_detail":       recallDetail,
			"final_response":      finalResponse[:min(200, len(finalResponse))],
			"context_items":       len(retrievedContext),
		},
	}
}

// CalculateVegetarianAwareness is a special metric for Test 7B
// Checks if response acknowledges vegetarian preference
func (m *MetricsCalculator) CalculateVegetarianAwareness(response string) (bool, string) {
	responseLower := strings.ToLower(response)

	// Keywords indicating vegetarian awareness
	vegetarianKeywords := []string{
		"vegetarian",
		"plant-based",
		"meat-free",
		"vegetables",
		"salad",
		"vegan",
		"veggie",
	}

	for _, keyword := range vegetarianKeywords {
		if strings.Contains(responseLower, keyword) {
			return true, fmt.Sprintf("Vegetarian-aware response (contains '%s')", keyword)
		}
	}

	return false, "No vegetarian awareness detected in response"
}

// CalculateTemporalCorrectness is a special metric for Test 7A
// Checks if response contains most recent fact, not superseded fact
func (m *MetricsCalculator) CalculateTemporalCorrectness(
	response string,
	currentValue string,
	supersededValue string,
) (bool, string) {
	responseUpper := strings.ToUpper(response)
	containsCurrent := strings.Contains(responseUpper, strings.ToUpper(currentValue))
	containsSuperseded := strings.Contains(responseUpper, strings.ToUpper(supersededValue))

	if containsCurrent && !containsSuperseded {
		return true, "Response contains only current value (temporal ordering correct)"
	}

	if containsCurrent && containsSuperseded {
		return false, "Response contains both current and superseded values (ambiguous)"
	}

	if !containsCurrent && containsSuperseded {
		return false, "Response contains only superseded value (temporal ordering FAILED)"
	}

	return false, "Response contains neither value (fact not retrieved)"
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
