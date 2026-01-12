package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/thirteen37/amail/internal/config"
	"github.com/thirteen37/amail/internal/db"
	"github.com/thirteen37/amail/internal/identity"
)

var whoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Show current identity",
	Long: `Show the current identity and how it was resolved.

Identity is resolved in this order:
  1. $AMAIL_IDENTITY environment variable
  2. tmux session name mapping from config
  3. Not set (prompts to register)

Examples:
  amail whoami`,
	RunE: runWhoami,
}

func init() {
	rootCmd.AddCommand(whoamiCmd)
}

func runWhoami(cmd *cobra.Command, args []string) error {
	// Find project root
	root, err := db.FindProjectRoot()
	if err != nil {
		return err
	}

	// Load config
	cfg, err := config.LoadProject(root)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Resolve identity
	res, err := identity.Resolve(cfg)
	if err != nil {
		return err
	}

	if res == nil {
		fmt.Println("Identity not set.")
		fmt.Println()
		fmt.Println("Resolution attempted:")
		fmt.Println("  - $AMAIL_IDENTITY: not set")

		if identity.IsInTmux() {
			tmuxSession := identity.GetTmuxSession()
			fmt.Printf("  - tmux session: %s (no mapping in config)\n", tmuxSession)
		} else {
			fmt.Println("  - tmux: not running in tmux")
		}

		fmt.Println()
		fmt.Println("To set identity:")
		fmt.Println("  source <(amail use <role>)")
		fmt.Println()
		fmt.Println("Available roles:", formatRoles(cfg))

		return nil
	}

	fmt.Printf("%s\n", res.Identity)
	fmt.Printf("  (from %s)\n", res.Source)

	// Check if valid role
	if !cfg.IsValidRole(res.Identity) {
		fmt.Println()
		fmt.Printf("  Warning: '%s' is not a defined role\n", res.Identity)
		fmt.Println("  Available roles:", formatRoles(cfg))
	}

	return nil
}

func formatRoles(cfg *config.Config) string {
	roles := cfg.AllRoles()
	if len(roles) == 0 {
		return "(none defined)"
	}
	result := ""
	for i, r := range roles {
		if i > 0 {
			result += ", "
		}
		if r == "user" {
			result += "user (reserved)"
		} else {
			result += r
		}
	}
	return result
}
