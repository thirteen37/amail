package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/thirteen37/amail/internal/config"
	"github.com/thirteen37/amail/internal/db"
	"github.com/thirteen37/amail/internal/identity"
)

var markReadCmd = &cobra.Command{
	Use:   "mark-read [message-id]",
	Short: "Mark message(s) as read",
	Long: `Mark one or all messages as read.

Examples:
  amail mark-read abc123
  amail mark-read --all`,
	Args: cobra.MaximumNArgs(1),
	RunE: runMarkRead,
}

var markReadAll bool

var archiveCmd = &cobra.Command{
	Use:   "archive <message-id>",
	Short: "Archive a message",
	Long: `Archive a message (removes from inbox but keeps in database).

Examples:
  amail archive abc123`,
	Args: cobra.ExactArgs(1),
	RunE: runArchive,
}

var deleteCmd = &cobra.Command{
	Use:   "delete <message-id>",
	Short: "Delete a message from your inbox",
	Long: `Delete a message from your inbox.

This only removes the message from your view; other recipients still have it.

Examples:
  amail delete abc123`,
	Args: cobra.ExactArgs(1),
	RunE: runDelete,
}

func init() {
	markReadCmd.Flags().BoolVar(&markReadAll, "all", false, "Mark all unread messages as read")
	rootCmd.AddCommand(markReadCmd)
	rootCmd.AddCommand(archiveCmd)
	rootCmd.AddCommand(deleteCmd)
}

func runMarkRead(cmd *cobra.Command, args []string) error {
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

	if markReadAll {
		// Mark all as read
		count, err := database.MarkAllRead(toID)
		if err != nil {
			return fmt.Errorf("failed to mark all as read: %w", err)
		}
		fmt.Printf("✓ Marked %d messages as read\n", count)
		return nil
	}

	// Need message ID
	if len(args) == 0 {
		return fmt.Errorf("message ID required (or use --all)")
	}

	messageID := args[0]

	// Find message by prefix
	msg, err := findMessageByPrefix(database, messageID, toID)
	if err != nil {
		return err
	}
	if msg == nil {
		return fmt.Errorf("message not found: %s", messageID)
	}

	if err := database.MarkRead(msg.ID, toID); err != nil {
		return fmt.Errorf("failed to mark as read: %w", err)
	}

	fmt.Printf("✓ Marked %s as read\n", SafeShortID(msg.ID))
	return nil
}

func runArchive(cmd *cobra.Command, args []string) error {
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

	messageID := args[0]

	// Find message by prefix
	msg, err := findMessageByPrefix(database, messageID, toID)
	if err != nil {
		return err
	}
	if msg == nil {
		return fmt.Errorf("message not found: %s", messageID)
	}

	if err := database.Archive(msg.ID, toID); err != nil {
		return fmt.Errorf("failed to archive: %w", err)
	}

	fmt.Printf("✓ Archived %s\n", SafeShortID(msg.ID))
	return nil
}

func runDelete(cmd *cobra.Command, args []string) error {
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

	messageID := args[0]

	// Find message by prefix
	msg, err := findMessageByPrefix(database, messageID, toID)
	if err != nil {
		return err
	}
	if msg == nil {
		return fmt.Errorf("message not found: %s", messageID)
	}

	if err := database.Delete(msg.ID, toID); err != nil {
		return fmt.Errorf("failed to delete: %w", err)
	}

	fmt.Printf("✓ Deleted %s\n", SafeShortID(msg.ID))
	return nil
}
