package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "amail",
	Short: "Multi-agent mailbox system",
	Long: `amail is a CLI-first mailbox system for multi-agent coordination.

It provides async communication between AI agents (and humans) working
on the same project. Each project has its own mailbox database.
Agents identify by role (pm, dev, qa, etc.), not by session.`,
}

// Execute runs the root command
func Execute() error {
	// Configure Cobra to not print errors when JSON output is active
	// We'll handle error output ourselves
	rootCmd.SilenceErrors = IsJSONOutput()
	rootCmd.SilenceUsage = IsJSONOutput()

	err := rootCmd.Execute()
	if err != nil && IsJSONOutput() {
		PrintJSONError(err, "")
	}
	return err
}

func init() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.PersistentFlags().BoolVar(&forceJSON, "json", false, "Force JSON output")
	rootCmd.PersistentFlags().BoolVar(&forceText, "text", false, "Force human-readable text output")
}

// exitWithError prints an error message and exits
func exitWithError(msg string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "Error: "+msg+"\n", args...)
	os.Exit(1)
}
