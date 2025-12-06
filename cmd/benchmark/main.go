// ABOUTME: Command-line benchmark runner for RAGAS tests
// ABOUTME: Executes RAGAS benchmarks and outputs JSON results

package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/harper/remember-standalone/benchmarks/ragas"
	"github.com/joho/godotenv"
)

func main() {
	// Command-line flags
	testID := flag.String("test", "", "Run specific test (7a, 7b, 2a). If empty, runs all tests.")
	outputPath := flag.String("output", "benchmark_results.json", "Output path for JSON results")
	verbose := flag.Bool("verbose", false, "Enable verbose output")
	flag.Parse()

	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Printf("No .env file found (continuing anyway): %v", err)
	}

	// Verify OpenAI API key
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable is required for benchmarks")
	}

	// Print header
	fmt.Println("========================================")
	fmt.Println("HMLR RAGAS Benchmarks")
	fmt.Println("========================================")
	fmt.Println()

	// Create benchmark runner
	runner, err := ragas.NewBenchmarkRunner(apiKey, *verbose)
	if err != nil {
		log.Fatalf("Failed to create benchmark runner: %v", err)
	}
	defer runner.Close()

	// Run tests
	var results []ragas.TestResult

	if *testID == "" {
		// Run all tests
		fmt.Println("Running all RAGAS benchmark tests...")
		fmt.Println()

		results, err = runner.RunAllTests()
		if err != nil {
			log.Fatalf("Benchmark failed: %v", err)
		}
	} else {
		// Run specific test
		var scenario ragas.TestScenario

		switch *testID {
		case "7a":
			scenario = ragas.GetTest7A()
		case "7b":
			scenario = ragas.GetTest7B()
		case "2a":
			scenario = ragas.GetTest2A()
		default:
			log.Fatalf("Unknown test ID: %s (valid options: 7a, 7b, 2a)", *testID)
		}

		fmt.Printf("Running test: %s\n\n", scenario.Name)

		result, err := runner.RunTest(scenario)
		if err != nil {
			log.Fatalf("Test failed: %v", err)
		}

		results = []ragas.TestResult{result}
	}

	// Print summary
	fmt.Println("\n========================================")
	fmt.Println("BENCHMARK SUMMARY")
	fmt.Println("========================================")

	passed := 0
	failed := 0

	for _, result := range results {
		fmt.Printf("\n%s: %s\n", result.TestID, result.TestName)
		fmt.Printf("  Faithfulness: %.2f\n", result.FaithfulnessScore)
		fmt.Printf("  Context Recall: %.2f\n", result.ContextRecallScore)
		fmt.Printf("  Overall: %.2f\n", result.OverallScore)
		fmt.Printf("  Status: %s\n", result.Status)

		if result.Status == "PASS" {
			passed++
		} else {
			failed++
		}
	}

	fmt.Println("\n========================================")
	fmt.Printf("Total Tests: %d\n", len(results))
	fmt.Printf("Passed: %d\n", passed)
	fmt.Printf("Failed: %d\n", failed)
	fmt.Println("========================================")

	// Export results
	if err := runner.ExportResults(results, *outputPath); err != nil {
		log.Fatalf("Failed to export results: %v", err)
	}

	// Exit with error code if any tests failed
	if failed > 0 {
		os.Exit(1)
	}
}
