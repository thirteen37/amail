package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/thirteen37/amail/internal/config"
	"github.com/thirteen37/amail/internal/db"
	"github.com/thirteen37/amail/internal/identity"
)

var replyCmd = &cobra.Command{
	Use:   "reply <message-id> <body>",
	Short: "Reply to a message",
	Long: `Reply to a message, optionally including all recipients.

By default, replies only to the sender.
Use --all to reply to sender + all original recipients (minus yourself).

Examples:
  amail reply abc123 "Got it, working on it"
  amail reply abc123 --all "Acknowledged by all"
  amail reply abc123 -p high "Urgent response"`,
	Args: cobra.ExactArgs(2),
	RunE: runReply,
}

var (
	replyAll      bool
	replyPriority string
	replyType     string
)

func init() {
	replyCmd.Flags().BoolVar(&replyAll, "all", false, "Reply to sender + all recipients")
	replyCmd.Flags().StringVarP(&replyPriority, "priority", "p", "normal", "Priority: low, normal, high, urgent")
	replyCmd.Flags().StringVarP(&replyType, "type", "t", "response", "Type: message, request, response, notification")
	rootCmd.AddCommand(replyCmd)
}

func runReply(cmd *cobra.Command, args []string) error {
	messageIDArg := args[0]
	body := args[1]

	if err := validatePriority(replyPriority); err != nil {
		return err
	}
	if err := validateMsgType(replyType); err != nil {
		return err
	}

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

	// Resolve sender identity
	res, err := identity.MustResolve(cfg)
	if err != nil {
		return err
	}
	fromID := res.Identity

	// Find the original message
	originalMsg, err := findMessageByPrefix(database, messageIDArg, fromID)
	if err != nil {
		return err
	}
	if originalMsg == nil {
		// Try to find by global search (in case user is replying to a message they sent)
		originalMsg, err = findMessageGlobally(database, messageIDArg)
		if err != nil {
			return err
		}
		if originalMsg == nil {
			return fmt.Errorf("message not found: %s", messageIDArg)
		}
	}

	// Determine recipients
	var recipients []string
	if replyAll {
		// Include original sender + all original recipients (minus self)
		recipients = append(recipients, originalMsg.FromID)
		recipients = append(recipients, originalMsg.ToIDs...)
		recipients = filterOut(recipients, fromID)
		recipients = dedupe(recipients)
	} else {
		// Just reply to sender
		if originalMsg.FromID == fromID {
			return fmt.Errorf("cannot reply to your own message without --all")
		}
		recipients = []string{originalMsg.FromID}
	}

	if len(recipients) == 0 {
		return fmt.Errorf("no recipients for reply")
	}

	// Determine thread ID
	var threadID string
	if originalMsg.ThreadID != nil {
		// Continue existing thread
		threadID = *originalMsg.ThreadID
	} else {
		// Start new thread with original message as root
		threadID = originalMsg.ID
	}

	// Generate subject
	subject := originalMsg.Subject
	if !strings.HasPrefix(strings.ToLower(subject), "re:") {
		subject = "RE: " + subject
	}

	// Create reply message
	msg := &db.Message{
		ID:        generateID(),
		FromID:    fromID,
		Subject:   subject,
		Body:      body,
		Priority:  replyPriority,
		MsgType:   replyType,
		ThreadID:  &threadID,
		ReplyToID: &originalMsg.ID,
		CreatedAt: time.Now(),
	}

	// Send
	if err := database.SendMessage(msg, recipients); err != nil {
		return fmt.Errorf("failed to send reply: %w", err)
	}

	fmt.Printf("✓ Sent %s to: %s (thread: %s)\n", SafeShortID(msg.ID), strings.Join(recipients, ", "), SafeShortID(threadID))

	return nil
}

// findMessageGlobally finds a message by ID prefix without recipient filter
func findMessageGlobally(database *db.DB, prefix string) (*db.InboxMessage, error) {
	msg, err := database.GetMessage(prefix)
	if err != nil {
		return nil, err
	}
	if msg != nil && strings.HasPrefix(msg.ID, prefix) {
		return msg, nil
	}
	return nil, nil
}

// dedupe removes duplicates from a slice
func dedupe(slice []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, s := range slice {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}
