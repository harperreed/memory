// ABOUTME: Version command to display build information
// ABOUTME: Shows version, commit hash, and build date
package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	versionInfo = VersionInfo{
		Version: "dev",
		Commit:  "none",
		Date:    "unknown",
	}
)

// VersionInfo contains build information
type VersionInfo struct {
	Version string
	Commit  string
	Date    string
}

// SetVersion sets the version information (called from main)
func SetVersion(version, commit, date string) {
	versionInfo.Version = version
	versionInfo.Commit = commit
	versionInfo.Date = date
}

// NewVersionCmd creates the version command
func NewVersionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Long:  `Display version, commit hash, and build date for the Memory CLI.`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintf(cmd.OutOrStdout(), "Memory (HMLR) %s\n", versionInfo.Version)
			fmt.Fprintf(cmd.OutOrStdout(), "Commit: %s\n", versionInfo.Commit)
			fmt.Fprintf(cmd.OutOrStdout(), "Built:  %s\n", versionInfo.Date)
		},
	}

	return cmd
}
