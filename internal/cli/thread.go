package cli

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/thirteen37/amail/internal/db"
)

var threadCmd = &cobra.Command{
	Use:   "thread <message-id>",
	Short: "View all messages in a thread",
	Long: `View all messages in a thread.

Given any message ID in the thread, shows all messages from the
thread root to the latest reply.

Examples:
  amail thread abc123`,
	Args: cobra.ExactArgs(1),
	RunE: runThread,
}

func init() {
	rootCmd.AddCommand(threadCmd)
}

func runThread(cmd *cobra.Command, args []string) error {
	messageIDArg := args[0]

	// Open project
	database, _, err := db.OpenProject()
	if err != nil {
		return err
	}
	defer database.Close()

	// Find the message to get thread ID (try exact match first, then prefix)
	msg, err := database.GetMessage(messageIDArg)
	if err != nil {
		return fmt.Errorf("failed to get message: %w", err)
	}
	if msg == nil {
		// Try prefix match
		msg, err = database.FindMessageByPrefix(messageIDArg)
		if err != nil {
			return fmt.Errorf("failed to find message: %w", err)
		}
	}
	if msg == nil {
		return fmt.Errorf("message not found: %s", messageIDArg)
	}

	// Determine thread root
	var threadRootID string
	if msg.ThreadID != nil {
		threadRootID = *msg.ThreadID
	} else {
		// This message might be the root
		threadRootID = msg.ID
	}

	// Get all messages in thread
	messages, err := database.GetThread(threadRootID)
	if err != nil {
		return fmt.Errorf("failed to get thread: %w", err)
	}

	if len(messages) == 0 {
		fmt.Println("No messages in thread.")
		return nil
	}

	// Get subject from first message
	subject := messages[0].Subject
	if subject == "" {
		subject = "(no subject)"
	}

	fmt.Printf("Thread: %s (%d messages)\n", subject, len(messages))
	fmt.Println()

	// Print table
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tFROM\tTO\tTIME")
	fmt.Fprintln(w, "--\t----\t--\t----")

	for _, m := range messages {
		// Format recipients
		toStr := strings.Join(m.ToIDs, ",")
		if len([]rune(toStr)) > 25 {
			toStr = string([]rune(toStr)[:22]) + "..."
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			SafeShortID(m.ID), m.FromID, toStr, m.CreatedAt.Format("15:04:05"))
	}
	w.Flush()

	fmt.Println()
	fmt.Println("Messages:")
	fmt.Println()

	// Print each message body
	for i, m := range messages {
		if i > 0 {
			fmt.Println(strings.Repeat("-", 40))
		}
		fmt.Printf("[%s] %s → %s (%s)\n", SafeShortID(m.ID), m.FromID, strings.Join(m.ToIDs, ","), m.CreatedAt.Format("15:04"))
		fmt.Println()
		fmt.Println(m.Body)
		fmt.Println()
	}

	return nil
}
