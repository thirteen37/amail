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

var sendCmd = &cobra.Command{
	Use:   "send <to> <subject> <body>",
	Short: "Send a message",
	Long: `Send a message to one or more recipients.

Recipients can be:
  - Single: dev
  - Multiple: dev,qa,pm
  - Groups: @all, @agents, @others, or custom groups from config

Examples:
  amail send dev "API ready" "GET /users endpoint at routes/users.ts:45"
  amail send dev,qa "Ready for review" "Feature complete"
  amail send @all "Announcement" "Deploy at 3pm"
  amail send dev -p urgent "Bug found" "Production issue"
  amail send pm -t request "Need spec" "Please clarify requirements"`,
	Args: cobra.ExactArgs(3),
	RunE: runSend,
}

var (
	sendPriority string
	sendType     string
)

func init() {
	sendCmd.Flags().StringVarP(&sendPriority, "priority", "p", "normal", "Priority: low, normal, high, urgent")
	sendCmd.Flags().StringVarP(&sendType, "type", "t", "message", "Type: message, request, response, notification")
	rootCmd.AddCommand(sendCmd)
}

func runSend(cmd *cobra.Command, args []string) error {
	toArg := args[0]
	subject := args[1]
	body := args[2]

	if err := validatePriority(sendPriority); err != nil {
		return err
	}
	if err := validateMsgType(sendType); err != nil {
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

	// Resolve recipients
	recipients, err := resolveRecipients(toArg, fromID, cfg)
	if err != nil {
		return err
	}

	if len(recipients) == 0 {
		return fmt.Errorf("no recipients resolved")
	}

	// Remove sender from recipients (can't send to self)
	recipients = filterOut(recipients, fromID)
	if len(recipients) == 0 {
		return fmt.Errorf("cannot send to self only")
	}

	// Create message
	msg := &db.Message{
		ID:        generateID(),
		FromID:    fromID,
		Subject:   subject,
		Body:      body,
		Priority:  sendPriority,
		MsgType:   sendType,
		CreatedAt: time.Now(),
	}

	// Send
	if err := database.SendMessage(msg, recipients); err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	fmt.Printf("✓ Sent %s to: %s\n", msg.ID, strings.Join(recipients, ", "))

	return nil
}

// resolveRecipients resolves a recipient string to a list of role IDs
func resolveRecipients(toArg, fromID string, cfg *config.Config) ([]string, error) {
	var allRecipients []string
	seen := make(map[string]bool)

	for _, part := range parseRecipients(toArg) {
		var resolved []string

		if strings.HasPrefix(part, "@") {
			// Group
			members := cfg.ResolveGroup(part, fromID)
			if members == nil {
				return nil, fmt.Errorf("unknown group: %s", part)
			}
			resolved = members
		} else {
			// Individual recipient - validate it exists
			if !cfg.IsValidRole(part) {
				return nil, fmt.Errorf("unknown recipient: %s (valid roles: %v)", part, cfg.AllRoles())
			}
			resolved = []string{part}
		}

		// Add to list, avoiding duplicates
		for _, r := range resolved {
			if !seen[r] {
				seen[r] = true
				allRecipients = append(allRecipients, r)
			}
		}
	}

	return allRecipients, nil
}

// filterOut removes a value from a slice
func filterOut(slice []string, value string) []string {
	var result []string
	for _, s := range slice {
		if s != value {
			result = append(result, s)
		}
	}
	return result
}
