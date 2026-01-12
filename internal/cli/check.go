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

	if len(messages) == 0 {
		fmt.Println("No unread messages.")
		return nil
	}

	fmt.Printf("%d unread message(s)\n", len(messages))

	if checkNotify {
		fmt.Println()
		for _, msg := range messages {
			// Get notification commands based on priority
			commands := cfg.GetNotifyCommands(msg.Priority)
			if len(commands) == 0 {
				continue
			}

			// Execute notifications
			notifyMsg := notify.FromInboxMessage(&msg)
			errors := notify.ExecuteAll(commands, notifyMsg)

			// Log any errors
			for _, err := range errors {
				fmt.Fprintf(os.Stderr, "Notification error: %v\n", err)
			}

			fmt.Printf("Notified: [%s] %s - %s\n",
				msg.ID[:8], msg.FromID, msg.Subject)
		}
	} else {
		fmt.Println()
		for _, msg := range messages {
			fmt.Printf("  [%s] %s: %s (%s)\n",
				msg.ID[:8], msg.FromID, msg.Subject, formatTimeAgo(msg.CreatedAt))
		}
		fmt.Println()
		fmt.Println("Use --notify to trigger notifications")
	}

	return nil
}

// Used for rate limiting notifications
var lastNotificationTime = make(map[string]time.Time)
