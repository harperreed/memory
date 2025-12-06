// ABOUTME: Test scenario data structures for RAGAS benchmarks
// ABOUTME: Defines conversation turns, expected outcomes, and ground truth for each test

package ragas

import "time"

// TestScenario represents a complete RAGAS benchmark test
type TestScenario struct {
	ID          string
	Name        string
	Description string
	Turns       []ConversationTurn
	GroundTruth GroundTruth
	Setup       *TestSetup // Optional pre-test setup (e.g., user profile)
}

// ConversationTurn represents a single turn in a test conversation
type ConversationTurn struct {
	TurnNumber  int
	UserMessage string
	Delay       time.Duration // Delay before this turn (for temporal tests)
}

// GroundTruth defines expected outcomes for RAGAS evaluation
type GroundTruth struct {
	// Expected facts to be stored
	ExpectedFacts []ExpectedFact

	// Expected response for final query turn
	FinalQueryTurn      int
	ExpectedInResponse  []string // Strings that MUST appear in response
	ForbiddenInResponse []string // Strings that MUST NOT appear in response

	// Context retrieval expectations
	ExpectedContextItems []string // Facts/context that should be retrieved
}

// ExpectedFact represents a fact that should be extracted
type ExpectedFact struct {
	Key       string
	Value     string
	TurnStored int
	Supersedes *ExpectedFact // For temporal conflicts (Test 7A)
}

// TestSetup defines pre-test environment setup
type TestSetup struct {
	UserProfile *UserProfileSetup
}

// UserProfileSetup defines pre-populated user profile data
type UserProfileSetup struct {
	Name        string
	Preferences []string
	Constraints []ProfileConstraint
}

// ProfileConstraint represents a user constraint in profile
type ProfileConstraint struct {
	Key         string
	Type        string
	Description string
	Severity    string
}

// TestResult represents the outcome of a benchmark test
type TestResult struct {
	TestID          string
	TestName        string
	FaithfulnessScore float64
	ContextRecallScore float64
	OverallScore    float64
	Status          string // "PASS" or "FAIL"
	Details         map[string]interface{}
	ErrorMessage    string
}

// GetTest7A returns Test 7A: API Key Rotation scenario
func GetTest7A() TestScenario {
	return TestScenario{
		ID:          "test_7a",
		Name:        "API Key Rotation (Temporal Conflict)",
		Description: "Tests that system prefers recent truths over past truths",
		Turns: []ConversationTurn{
			{
				TurnNumber:  1,
				UserMessage: "My API Key for the weather service is ABC123.",
				Delay:       0,
			},
			{
				TurnNumber:  2,
				UserMessage: "I rotated my keys. The new API Key is XYZ789.",
				Delay:       200 * time.Millisecond, // Ensure unique timestamps
			},
			{
				TurnNumber:  3,
				UserMessage: "What is my API key?",
				Delay:       200 * time.Millisecond,
			},
		},
		GroundTruth: GroundTruth{
			ExpectedFacts: []ExpectedFact{
				{
					Key:        "weather_api_key",
					Value:      "ABC123",
					TurnStored: 1,
				},
				{
					Key:        "weather_api_key",
					Value:      "XYZ789",
					TurnStored: 2,
					Supersedes: &ExpectedFact{
						Key:   "weather_api_key",
						Value: "ABC123",
					},
				},
			},
			FinalQueryTurn:      3,
			ExpectedInResponse:  []string{"XYZ789"},
			ForbiddenInResponse: []string{"ABC123"}, // Old key must NOT appear
			ExpectedContextItems: []string{
				"XYZ789", // Most recent key must be in context
			},
		},
	}
}

// GetTest7B returns Test 7B: Vegetarian Trap scenario
func GetTest7B() TestScenario {
	return TestScenario{
		ID:          "test_7b",
		Name:        "Vegetarian Trap (User Profile vs Override)",
		Description: "Tests that user profile constraints are honored in context",
		Setup: &TestSetup{
			UserProfile: &UserProfileSetup{
				Name:        "",
				Preferences: []string{},
				Constraints: []ProfileConstraint{
					{
						Key:         "diet_vegetarian",
						Type:        "Dietary Restriction",
						Description: "User is strictly vegetarian, does not eat meat or fish",
						Severity:    "strict",
					},
				},
			},
		},
		Turns: []ConversationTurn{
			{
				TurnNumber:  1,
				UserMessage: "I'm going to a steakhouse tonight. Can you recommend a dish for me to eat?",
				Delay:       0,
			},
		},
		GroundTruth: GroundTruth{
			ExpectedFacts: []ExpectedFact{
				// User profile constraint (pre-populated)
				{
					Key:        "diet_vegetarian",
					Value:      "User is strictly vegetarian, does not eat meat or fish",
					TurnStored: 0, // Pre-populated
				},
			},
			FinalQueryTurn: 1,
			ExpectedInResponse: []string{
				// Must acknowledge vegetarian preference
				// (flexible - any of these keywords indicates awareness)
			},
			ForbiddenInResponse: []string{
				// Must NOT blindly recommend meat
				"try the ribeye",
				"order a filet",
				"steak is great",
				"recommend the steak",
			},
			ExpectedContextItems: []string{
				"vegetarian", // User profile constraint must be in context
			},
		},
	}
}

// GetTest2A returns Test 2A: 10-Turn Vague Secret Retrieval scenario
func GetTest2A() TestScenario {
	return TestScenario{
		ID:          "test_2a",
		Name:        "10-Turn Vague Secret Retrieval (Zero-Keyword Recall)",
		Description: "Tests vague query interpretation - LLM must understand 'credential' refers to API key",
		Turns: []ConversationTurn{
			{
				TurnNumber:  1,
				UserMessage: "My API key for the weather service is ABC123XYZ. Can you help me set up a weather dashboard?",
				Delay:       0,
			},
			{
				TurnNumber:  2,
				UserMessage: "I want to display temperature and humidity",
				Delay:       100 * time.Millisecond,
			},
			{
				TurnNumber:  3,
				UserMessage: "Should I use Celsius or Fahrenheit?",
				Delay:       100 * time.Millisecond,
			},
			{
				TurnNumber:  4,
				UserMessage: "Let's go with Fahrenheit",
				Delay:       100 * time.Millisecond,
			},
			{
				TurnNumber:  5,
				UserMessage: "How do I structure the HTML layout?",
				Delay:       100 * time.Millisecond,
			},
			{
				TurnNumber:  6,
				UserMessage: "What about styling with CSS?",
				Delay:       100 * time.Millisecond,
			},
			{
				TurnNumber:  7,
				UserMessage: "I need to make API calls from JavaScript",
				Delay:       100 * time.Millisecond,
			},
			{
				TurnNumber:  8,
				UserMessage: "What's the best way to handle errors?",
				Delay:       100 * time.Millisecond,
			},
			{
				TurnNumber:  9,
				UserMessage: "Should I cache the weather data?",
				Delay:       100 * time.Millisecond,
			},
			{
				TurnNumber:  10,
				UserMessage: "Remind me what credential I need for the weather service? Please respond with only the credential value.",
				Delay:       100 * time.Millisecond,
			},
		},
		GroundTruth: GroundTruth{
			ExpectedFacts: []ExpectedFact{
				{
					Key:        "weather_api_key",
					Value:      "ABC123XYZ",
					TurnStored: 1,
				},
			},
			FinalQueryTurn:      10,
			ExpectedInResponse:  []string{"ABC123XYZ"},
			ForbiddenInResponse: []string{}, // No forbidden strings
			ExpectedContextItems: []string{
				"ABC123XYZ", // API key must be in context
			},
		},
	}
}

// GetAllTests returns all RAGAS benchmark tests
func GetAllTests() []TestScenario {
	return []TestScenario{
		GetTest7A(),
		GetTest7B(),
		GetTest2A(),
	}
}
