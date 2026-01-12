package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version information (can be set via ldflags at build time)
var (
	Version   = "0.1.0"
	BuildDate = "unknown"
	GitCommit = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("amail %s\n", Version)
		if GitCommit != "unknown" {
			fmt.Printf("  commit: %s\n", GitCommit)
		}
		if BuildDate != "unknown" {
			fmt.Printf("  built:  %s\n", BuildDate)
		}
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
