package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/thirteen37/amail/internal/config"
	"github.com/thirteen37/amail/internal/db"
	"github.com/thirteen37/amail/internal/identity"
	"github.com/thirteen37/amail/internal/notify"
)

// CheckOutput is the JSON output structure for the check command
type CheckOutput struct {
	Messages []CheckMessageJSON `json:"messages"`
	Count    int                `json:"count"`
}

// CheckMessageJSON is the JSON representation of a check message
type CheckMessageJSON struct {
	ID        string   `json:"id"`
	ShortID   string   `json:"short_id"`
	From      string   `json:"from"`
	To        []string `json:"to"`
	Subject   string   `json:"subject"`
	Priority  string   `json:"priority"`
	CreatedAt string   `json:"created_at"`
}

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Check for new messages and notify",
	Long: `One-shot check for new messages and trigger notifications.

Useful for cron jobs or scripts.

Examples:
  amail check --notify`,
	RunE: runCheck,
}

var checkNotify bool

func init() {
	checkCmd.Flags().BoolVar(&checkNotify, "notify", false, "Trigger notifications for unread messages")
	rootCmd.AddCommand(checkCmd)
}

func runCheck(cmd *cobra.Command, args []string) error {
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

	// Get unread messages
	messages, err := database.GetInbox(toID, false)
	if err != nil {
		return fmt.Errorf("failed to get inbox: %w", err)
	}

	// Execute notifications if requested (do this before output so it happens regardless of format)
	if checkNotify {
		for _, msg := range messages {
			// Get notification commands based on priority
			commands := cfg.GetNotifyCommands(msg.Priority)
			if len(commands) == 0 {
				continue
			}

			// Execute notifications
			notifyMsg := notify.FromInboxMessage(&msg)
			errors := notify.ExecuteAll(commands, notifyMsg)

			// Log any errors to stderr (not part of JSON output)
			for _, err := range errors {
				fmt.Fprintf(os.Stderr, "Notification error: %v\n", err)
			}
		}
	}

	// JSON output
	if IsJSONOutput() {
		output := CheckOutput{
			Messages: make([]CheckMessageJSON, len(messages)),
			Count:    len(messages),
		}
		for i, m := range messages {
			output.Messages[i] = CheckMessageJSON{
				ID:        m.ID,
				ShortID:   SafeShortID(m.ID),
				From:      m.FromID,
				To:        m.ToIDs,
				Subject:   m.Subject,
				Priority:  m.Priority,
				CreatedAt: m.CreatedAt.Format(time.RFC3339),
			}
		}
		return PrintJSON(output)
	}

	// Text output
	if len(messages) == 0 {
		fmt.Println("No unread messages.")
		return nil
	}

	fmt.Printf("%d unread message(s)\n", len(messages))

	if checkNotify {
		fmt.Println()
		for _, msg := range messages {
			fmt.Printf("Notified: [%s] %s - %s\n",
				SafeShortID(msg.ID), msg.FromID, msg.Subject)
		}
	} else {
		fmt.Println()
		for _, msg := range messages {
			fmt.Printf("  [%s] %s: %s (%s)\n",
				SafeShortID(msg.ID), msg.FromID, msg.Subject, formatTimeAgo(msg.CreatedAt))
		}
		fmt.Println()
		fmt.Println("Use --notify to trigger notifications")
	}

	return nil
}
