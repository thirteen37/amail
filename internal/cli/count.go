package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/thirteen37/amail/internal/config"
	"github.com/thirteen37/amail/internal/db"
	"github.com/thirteen37/amail/internal/identity"
)

var countCmd = &cobra.Command{
	Use:   "count",
	Short: "Count unread messages",
	Long: `Count unread messages in your inbox.

Useful for status bars and scripts.

Examples:
  amail count
  # In tmux status bar: #(amail count)`,
	RunE: runCount,
}

func init() {
	rootCmd.AddCommand(countCmd)
}

func runCount(cmd *cobra.Command, args []string) error {
	// Open project
	database, root, err := db.OpenProject()
	if err != nil {
		// For count, just print 0 if not in project (for status bars)
		fmt.Println("0")
		return nil
	}
	defer database.Close()

	// Load config
	cfg, err := config.LoadProject(root)
	if err != nil {
		fmt.Println("0")
		return nil
	}

	// Resolve identity
	res, err := identity.Resolve(cfg)
	if err != nil || res == nil {
		fmt.Println("0")
		return nil
	}

	// Get count
	count, err := database.CountUnread(res.Identity)
	if err != nil {
		fmt.Println("0")
		return nil
	}

	fmt.Println(count)
	return nil
}
