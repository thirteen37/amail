package notify

import (
	"os/exec"
	"strings"
	"time"

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
func Execute(command string, msg *Message) error {
	// Substitute template variables
	cmd := substituteTemplateVars(command, msg)

	// Execute command via shell
	return exec.Command("sh", "-c", cmd).Run()
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

// substituteTemplateVars replaces {var} with message values
func substituteTemplateVars(template string, msg *Message) string {
	replacements := map[string]string{
		"{id}":        msg.ID,
		"{from}":      msg.From,
		"{to}":        msg.To,
		"{subject}":   msg.Subject,
		"{body}":      truncateForNotification(msg.Body, 100),
		"{priority}":  msg.Priority,
		"{type}":      msg.Type,
		"{timestamp}": msg.Timestamp.Format("15:04:05"),
	}

	result := template
	for key, value := range replacements {
		// Escape single quotes in values for shell safety
		safeValue := strings.ReplaceAll(value, "'", "'\\''")
		result = strings.ReplaceAll(result, key, safeValue)
	}

	return result
}

// truncateForNotification truncates a string for notification display
func truncateForNotification(s string, maxLen int) string {
	// Replace newlines with spaces
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", "")

	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
