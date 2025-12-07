// ABOUTME: Root Cobra command for Memory CLI
// ABOUTME: Sets up global flags and subcommands
package commands

import (
	"github.com/spf13/cobra"
)

var (
	verbose      bool
	quiet        bool
	outputFormat string
)

// NewRootCmd creates the root command
func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "memory",
		Short: "HMLR - Hierarchical Memory Lookup & Routing",
		Long: `
███╗   ███╗███████╗███╗   ███╗ ██████╗ ██████╗ ██╗   ██╗
████╗ ████║██╔════╝████╗ ████║██╔═══██╗██╔══██╗╚██╗ ██╔╝
██╔████╔██║█████╗  ██╔████╔██║██║   ██║██████╔╝ ╚████╔╝
██║╚██╔╝██║██╔══╝  ██║╚██╔╝██║██║   ██║██╔══██╗  ╚██╔╝
██║ ╚═╝ ██║███████╗██║ ╚═╝ ██║╚██████╔╝██║  ██║   ██║
╚═╝     ╚═╝╚══════╝╚═╝     ╚═╝ ╚═════╝ ╚═╝  ╚═╝   ╚═╝

    🧠 Hierarchical Memory for LLM Agents 🤖

Memory is a CLI and MCP server that provides intelligent memory management
for LLM agents using semantic search, fact extraction, and smart routing.

HMLR Architecture Components:
  • Governor       - Smart topic routing with 4 scenarios
  • ChunkEngine    - Hierarchical text chunking (turn → paragraph → sentence)
  • FactScrubber   - LLM-based fact extraction
  • LatticeCrawler - Vector similarity search
  • Scribe         - Async user profile learning
  • ContextHydrator - Intelligent prompt assembly`,
		SilenceUsage: true,
	}

	// Global flags
	cmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")
	cmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "Quiet output")
	cmd.PersistentFlags().StringVar(&outputFormat, "format", "auto", "Output format (auto|json|table)")

	// Add subcommands
	cmd.AddCommand(NewMCPCmd())
	cmd.AddCommand(NewAddCmd())
	cmd.AddCommand(NewSearchCmd())
	cmd.AddCommand(NewListCmd())
	cmd.AddCommand(NewVersionCmd())

	return cmd
}

// Execute runs the root command
func Execute() error {
	return NewRootCmd().Execute()
}
