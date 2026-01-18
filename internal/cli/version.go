package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version information (can be set via ldflags at build time)
var (
	Version   = "0.2.0"
	BuildDate = "unknown"
	GitCommit = "unknown"
)

// VersionOutput is the JSON output structure for the version command
type VersionOutput struct {
	Version   string `json:"version"`
	Commit    string `json:"commit,omitempty"`
	BuildDate string `json:"build_date,omitempty"`
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	RunE:  runVersion,
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

func runVersion(cmd *cobra.Command, args []string) error {
	// JSON output
	if IsJSONOutput() {
		output := VersionOutput{
			Version: Version,
		}
		if GitCommit != "unknown" {
			output.Commit = GitCommit
		}
		if BuildDate != "unknown" {
			output.BuildDate = BuildDate
		}
		return PrintJSON(output)
	}

	// Text output
	fmt.Printf("amail %s\n", Version)
	if GitCommit != "unknown" {
		fmt.Printf("  commit: %s\n", GitCommit)
	}
	if BuildDate != "unknown" {
		fmt.Printf("  built:  %s\n", BuildDate)
	}
	return nil
}
