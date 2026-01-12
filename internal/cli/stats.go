package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/thirteen37/amail/internal/config"
	"github.com/thirteen37/amail/internal/db"
)

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show message statistics",
	Long: `Show statistics about messages in the project.

Examples:
  amail stats`,
	RunE: runStats,
}

func init() {
	rootCmd.AddCommand(statsCmd)
}

func runStats(cmd *cobra.Command, args []string) error {
	// Open project
	database, root, err := db.OpenProject()
	if err != nil {
		return err
	}
	defer database.Close()

	// Load config
	cfg, err := config.LoadProject(root)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	fmt.Println("Message Statistics")
	fmt.Println("==================")
	fmt.Println()

	// Get stats for each role
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ROLE\tUNREAD\tTOTAL")
	fmt.Fprintln(w, "----\t------\t-----")

	allRoles := cfg.AllRoles()
	var totalUnread, totalAll int

	for _, role := range allRoles {
		unread, err := database.CountUnread(role)
		if err != nil {
			continue
		}

		all, err := countAll(database, role)
		if err != nil {
			continue
		}

		if all > 0 {
			fmt.Fprintf(w, "%s\t%d\t%d\n", role, unread, all)
			totalUnread += unread
			totalAll += all
		}
	}

	fmt.Fprintln(w, "----\t------\t-----")
	fmt.Fprintf(w, "TOTAL\t%d\t%d\n", totalUnread, totalAll)
	w.Flush()

	return nil
}

func countAll(database *db.DB, toID string) (int, error) {
	messages, err := database.GetInbox(toID, true)
	if err != nil {
		return 0, err
	}
	return len(messages), nil
}
