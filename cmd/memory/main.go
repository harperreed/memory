// ABOUTME: Main entry point for Memory CLI
// ABOUTME: Sets up Cobra root command and executes CLI
package main

import (
	"fmt"
	"os"

	"github.com/harper/remember-standalone/cmd/memory/commands"
)

// Version information (set by goreleaser)
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	// Set version info for commands
	commands.SetVersion(version, commit, date)

	if err := commands.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
