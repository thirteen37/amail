package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/thirteen37/amail/internal/config"
	"github.com/thirteen37/amail/internal/db"
	"github.com/thirteen37/amail/internal/identity"
)

var readCmd = &cobra.Command{
	Use:   "read [message-id]",
	Short: "Read a message",
	Long: `Read a message and mark it as read.

If --latest is specified, reads the most recent unread message.

Examples:
  amail read abc123
  amail read --latest`,
	Args: cobra.MaximumNArgs(1),
	RunE: runRead,
}

var readLatest bool

func init() {
	readCmd.Flags().BoolVar(&readLatest, "latest", false, "Read the most recent unread message")
	rootCmd.AddCommand(readCmd)
}

func runRead(cmd *cobra.Command, args []string) error {
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

	var msg *db.InboxMessage

	if readLatest {
		// Get latest unread
		msg, err = database.GetLatestUnread(toID)
		if err != nil {
			return fmt.Errorf("failed to get latest message: %w", err)
		}
		if msg == nil {
			fmt.Println("No unread messages.")
			return nil
		}
	} else {
		// Need message ID
		if len(args) == 0 {
			return fmt.Errorf("message ID required (or use --latest)")
		}

		messageID := args[0]

		// Find message by ID prefix
		msg, err = findMessageByPrefix(database, messageID, toID)
		if err != nil {
			return err
		}
		if msg == nil {
			return fmt.Errorf("message not found: %s", messageID)
		}
	}

	// Display message
	displayMessage(msg)

	// Mark as read
	if msg.Status == "unread" {
		if err := database.MarkRead(msg.ID, toID); err != nil {
			return fmt.Errorf("failed to mark as read: %w", err)
		}
	}

	return nil
}

// findMessageByPrefix finds a message by ID prefix in the recipient's inbox
func findMessageByPrefix(database *db.DB, prefix, toID string) (*db.InboxMessage, error) {
	// Get all messages for recipient
	messages, err := database.GetInbox(toID, true)
	if err != nil {
		return nil, err
	}

	// Find by prefix
	var matches []*db.InboxMessage
	for i := range messages {
		if strings.HasPrefix(messages[i].ID, prefix) {
			matches = append(matches, &messages[i])
		}
	}

	if len(matches) == 0 {
		return nil, nil
	}
	if len(matches) > 1 {
		return nil, fmt.Errorf("ambiguous ID prefix: %s matches %d messages", prefix, len(matches))
	}

	return matches[0], nil
}

// displayMessage prints a message in a readable format
func displayMessage(msg *db.InboxMessage) {
	fmt.Println(strings.Repeat("-", 60))
	fmt.Printf("ID:       %s\n", msg.ID)
	fmt.Printf("From:     %s\n", msg.FromID)
	fmt.Printf("To:       %s\n", strings.Join(msg.ToIDs, ", "))
	fmt.Printf("Subject:  %s\n", msg.Subject)
	fmt.Printf("Priority: %s\n", msg.Priority)
	fmt.Printf("Type:     %s\n", msg.MsgType)
	fmt.Printf("Time:     %s (%s)\n", msg.CreatedAt.Format("2006-01-02 15:04:05"), formatTimeAgo(msg.CreatedAt))

	if msg.ThreadID != nil {
		fmt.Printf("Thread:   %s\n", *msg.ThreadID)
	}

	fmt.Println(strings.Repeat("-", 60))
	fmt.Println()
	fmt.Println(msg.Body)
	fmt.Println()
}
