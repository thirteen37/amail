package cli

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/thirteen37/amail/internal/config"
	"github.com/thirteen37/amail/internal/db"
	"github.com/thirteen37/amail/internal/identity"
	"github.com/thirteen37/amail/internal/notify"
)

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Watch inbox for new messages",
	Long: `Watch your inbox and trigger notifications for new messages.

Polls the database at a configurable interval and executes notification
commands when new messages arrive.

Configure notifications in .amail/config.toml:
  [notify.default]
  commands = ["echo '📬 {from}: {subject}'"]

  [notify.urgent]
  commands = ["terminal-notifier -title '🚨 {from}' -message '{body}'"]

Examples:
  amail watch
  amail watch --interval 5`,
	RunE: runWatch,
}

var watchInterval int

func init() {
	watchCmd.Flags().IntVar(&watchInterval, "interval", 0, "Polling interval in seconds (default from config)")
	rootCmd.AddCommand(watchCmd)
}

func runWatch(cmd *cobra.Command, args []string) error {
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

	// Determine interval
	interval := cfg.Watch.Interval
	if watchInterval > 0 {
		interval = watchInterval
	}
	if interval < 1 {
		interval = 2
	}

	fmt.Printf("Watching inbox for %s (interval: %ds)\n", toID, interval)
	fmt.Println("Press Ctrl+C to stop")
	fmt.Println()

	// Track last check time
	lastCheck := time.Now()

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	// Initial check
	if err := checkAndNotify(database, cfg, toID, &lastCheck); err != nil {
		fmt.Fprintf(os.Stderr, "Error checking inbox: %v\n", err)
	}

	for {
		select {
		case <-ticker.C:
			if err := checkAndNotify(database, cfg, toID, &lastCheck); err != nil {
				fmt.Fprintf(os.Stderr, "Error checking inbox: %v\n", err)
			}
		case <-sigChan:
			fmt.Println("\nStopping watch...")
			return nil
		}
	}
}

func checkAndNotify(database *db.DB, cfg *config.Config, toID string, lastCheck *time.Time) error {
	// Get unread messages
	messages, err := database.GetInbox(toID, false)
	if err != nil {
		return err
	}

	// Filter to messages created since last check
	var newMessages []db.InboxMessage
	for _, msg := range messages {
		if msg.CreatedAt.After(*lastCheck) {
			newMessages = append(newMessages, msg)
		}
	}

	// Update last check time
	*lastCheck = time.Now()

	// Notify for each new message
	for _, msg := range newMessages {
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

		// Mark as notified (update notified_at in database)
		// For now just log
		fmt.Printf("[%s] New message from %s: %s\n",
			time.Now().Format("15:04:05"), msg.FromID, msg.Subject)
	}

	return nil
}
