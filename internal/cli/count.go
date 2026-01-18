package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/thirteen37/amail/internal/config"
	"github.com/thirteen37/amail/internal/db"
	"github.com/thirteen37/amail/internal/identity"
)

// CountOutput is the JSON output structure for the count command
type CountOutput struct {
	Count int `json:"count"`
}

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
	// Helper to output count
	outputCount := func(count int) error {
		if IsJSONOutput() {
			return PrintJSON(CountOutput{Count: count})
		}
		fmt.Println(count)
		return nil
	}

	// Open project
	database, root, err := db.OpenProject()
	if err != nil {
		// For count, just print 0 if not in project (for status bars)
		return outputCount(0)
	}
	defer database.Close()

	// Load config
	cfg, err := config.LoadProject(root)
	if err != nil {
		return outputCount(0)
	}

	// Resolve identity
	res, err := identity.Resolve(cfg)
	if err != nil || res == nil {
		return outputCount(0)
	}

	// Get count
	count, err := database.CountUnread(res.Identity)
	if err != nil {
		return outputCount(0)
	}

	return outputCount(count)
}
