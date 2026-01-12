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
	return rootCmd.Execute()
}

func init() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
}

// exitWithError prints an error message and exits
func exitWithError(msg string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "Error: "+msg+"\n", args...)
	os.Exit(1)
}
