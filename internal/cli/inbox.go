package cli

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/thirteen37/amail/internal/config"
	"github.com/thirteen37/amail/internal/db"
	"github.com/thirteen37/amail/internal/identity"
)

var inboxCmd = &cobra.Command{
	Use:   "inbox",
	Short: "List messages in inbox",
	Long: `List messages in your inbox.

By default shows only unread messages.

Examples:
  amail inbox
  amail inbox -a         # Show all messages
  amail inbox --from dev # Filter by sender`,
	RunE: runInbox,
}

var (
	inboxAll  bool
	inboxFrom string
)

func init() {
	inboxCmd.Flags().BoolVarP(&inboxAll, "all", "a", false, "Show all messages (including read)")
	inboxCmd.Flags().StringVar(&inboxFrom, "from", "", "Filter by sender")
	rootCmd.AddCommand(inboxCmd)
}

func runInbox(cmd *cobra.Command, args []string) error {
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

	// Resolve identity
	res, err := identity.MustResolve(cfg)
	if err != nil {
		return err
	}
	toID := res.Identity

	// Get messages
	messages, err := database.GetInbox(toID, inboxAll)
	if err != nil {
		return fmt.Errorf("failed to get inbox: %w", err)
	}

	// Filter by sender if specified
	if inboxFrom != "" {
		var filtered []db.InboxMessage
		for _, m := range messages {
			if m.FromID == inboxFrom {
				filtered = append(filtered, m)
			}
		}
		messages = filtered
	}

	if len(messages) == 0 {
		if inboxAll {
			fmt.Println("No messages.")
		} else {
			fmt.Println("No unread messages.")
		}
		return nil
	}

	// Print table
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tFROM\tSUBJECT\tTO\tPRIORITY\tTIME")
	fmt.Fprintln(w, "--\t----\t-------\t--\t--------\t----")

	for _, m := range messages {
		// Format recipients
		toStr := strings.Join(m.ToIDs, ",")
		if len([]rune(toStr)) > 20 {
			toStr = string([]rune(toStr)[:17]) + "..."
		}

		// Format subject
		subject := m.Subject
		if subject == "" {
			subject = "(no subject)"
		}
		subject = truncate(subject, 30)

		// Add status indicator
		statusIndicator := ""
		if m.Status == "unread" {
			statusIndicator = "*"
		}

		// Priority indicator
		priorityStr := m.Priority
		if m.Priority == "urgent" {
			priorityStr = "🚨 urgent"
		} else if m.Priority == "high" {
			priorityStr = "! high"
		}

		fmt.Fprintf(w, "%s%s\t%s\t%s\t%s\t%s\t%s\n",
			statusIndicator, SafeShortID(m.ID), m.FromID, subject, toStr, priorityStr, formatTimeAgo(m.CreatedAt))
	}

	w.Flush()

	return nil
}
