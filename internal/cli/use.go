package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/thirteen37/amail/internal/config"
	"github.com/thirteen37/amail/internal/db"
	"github.com/thirteen37/amail/internal/identity"
)

var useCmd = &cobra.Command{
	Use:   "use <role>",
	Short: "Set identity for current shell",
	Long: `Output a shell command to set the identity for the current shell.

Use with source to apply:
  source <(amail use dev)

This sets the AMAIL_IDENTITY environment variable.

Examples:
  source <(amail use dev)
  source <(amail use pm)
  source <(amail use user)`,
	Args: cobra.ExactArgs(1),
	RunE: runUse,
}

func init() {
	rootCmd.AddCommand(useCmd)
}

func runUse(cmd *cobra.Command, args []string) error {
	role := args[0]

	// Find project root (optional - we still allow use outside project)
	root, err := db.FindProjectRoot()
	if err == nil {
		// Load config to validate role
		cfg, err := config.LoadProject(root)
		if err == nil && !cfg.IsValidRole(role) {
			// Print warning to stderr so it doesn't interfere with source
			fmt.Fprintf(cmd.ErrOrStderr(), "# Warning: '%s' is not a defined role\n", role)
			fmt.Fprintf(cmd.ErrOrStderr(), "# Available roles: %s\n", formatRoles(cfg))
		}
	}

	// Output export command for shell to source
	fmt.Println(identity.ExportCommand(role))

	return nil
}
