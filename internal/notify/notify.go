package notify

import (
	"os"
	"os/exec"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/thirteen37/amail/internal/db"
)

// Message holds the data for a notification
type Message struct {
	ID        string
	From      string
	To        string
	Subject   string
	Body      string
	Priority  string
	Type      string
	Timestamp time.Time
}

// FromInboxMessage converts an InboxMessage to a notification Message
func FromInboxMessage(msg *db.InboxMessage) *Message {
	return &Message{
		ID:        msg.ID,
		From:      msg.FromID,
		To:        strings.Join(msg.ToIDs, ","),
		Subject:   msg.Subject,
		Body:      msg.Body,
		Priority:  msg.Priority,
		Type:      msg.MsgType,
		Timestamp: msg.CreatedAt,
	}
}

// Execute runs a notification command with template substitution
// Uses environment variables to safely pass message data, avoiding shell injection
func Execute(command string, msg *Message) error {
	// Create environment variables for template values
	env := os.Environ()
	env = append(env,
		"AMAIL_ID="+msg.ID,
		"AMAIL_FROM="+msg.From,
		"AMAIL_TO="+msg.To,
		"AMAIL_SUBJECT="+msg.Subject,
		"AMAIL_BODY="+truncateForNotification(msg.Body, 100),
		"AMAIL_PRIORITY="+msg.Priority,
		"AMAIL_TYPE="+msg.Type,
		"AMAIL_TIMESTAMP="+msg.Timestamp.Format("15:04:05"),
	)

	// Substitute template variables with shell variable references
	cmd := substituteTemplateVars(command)

	// Execute command via shell with safe environment variables
	c := exec.Command("sh", "-c", cmd)
	c.Env = env
	return c.Run()
}

// ExecuteAll runs all notification commands for a message
func ExecuteAll(commands []string, msg *Message) []error {
	var errors []error
	for _, cmd := range commands {
		if err := Execute(cmd, msg); err != nil {
			errors = append(errors, err)
		}
	}
	return errors
}

// substituteTemplateVars replaces {var} with shell variable references
// This allows the shell to safely expand the values from environment variables
func substituteTemplateVars(template string) string {
	replacements := map[string]string{
		"{id}":        `"$AMAIL_ID"`,
		"{from}":      `"$AMAIL_FROM"`,
		"{to}":        `"$AMAIL_TO"`,
		"{subject}":   `"$AMAIL_SUBJECT"`,
		"{body}":      `"$AMAIL_BODY"`,
		"{priority}":  `"$AMAIL_PRIORITY"`,
		"{type}":      `"$AMAIL_TYPE"`,
		"{timestamp}": `"$AMAIL_TIMESTAMP"`,
	}

	result := template
	for key, value := range replacements {
		result = strings.ReplaceAll(result, key, value)
	}

	return result
}

// truncateForNotification truncates a string for notification display
// Uses rune count instead of byte count for proper UTF-8 handling
func truncateForNotification(s string, maxLen int) string {
	// Replace newlines with spaces
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", "")

	if utf8.RuneCountInString(s) <= maxLen {
		return s
	}

	// Truncate by runes, not bytes
	runes := []rune(s)
	if maxLen <= 3 {
		return string(runes[:maxLen])
	}
	return string(runes[:maxLen-3]) + "..."
}
