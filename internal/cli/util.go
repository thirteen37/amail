package cli

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

// generateID creates a short random ID for messages
func generateID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// SafeShortID returns the first 8 characters of an ID, or the full ID if shorter
func SafeShortID(id string) string {
	if len(id) <= 8 {
		return id
	}
	return id[:8]
}

// formatTimeAgo formats a time as a relative time string
func formatTimeAgo(t time.Time) string {
	duration := time.Since(t)

	switch {
	case duration < time.Minute:
		return "just now"
	case duration < time.Hour:
		mins := int(duration.Minutes())
		if mins == 1 {
			return "1 min ago"
		}
		return fmt.Sprintf("%d min ago", mins)
	case duration < 24*time.Hour:
		hours := int(duration.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	case duration < 7*24*time.Hour:
		days := int(duration.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	default:
		return t.Format("Jan 2")
	}
}

// formatDuration formats a duration in a human-readable way
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return "just now"
	}
	if d < time.Hour {
		mins := int(d.Minutes())
		if mins == 1 {
			return "1 min"
		}
		return fmt.Sprintf("%d min", mins)
	}
	hours := int(d.Hours())
	if hours == 1 {
		return "1 hour"
	}
	return fmt.Sprintf("%d hours", hours)
}

// truncate truncates a string to maxLen runes and adds "..." if truncated
// Uses rune count instead of byte count for proper UTF-8 handling
func truncate(s string, maxLen int) string {
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

// parseRecipients parses a comma-separated list of recipients
func parseRecipients(input string) []string {
	var recipients []string
	for _, r := range strings.Split(input, ",") {
		r = strings.TrimSpace(r)
		if r != "" {
			recipients = append(recipients, r)
		}
	}
	return recipients
}

// Valid priority and message type values
var (
	validPriorities = map[string]bool{"low": true, "normal": true, "high": true, "urgent": true}
	validMsgTypes   = map[string]bool{"message": true, "request": true, "response": true, "notification": true}
)

// validatePriority checks if a priority value is valid
func validatePriority(priority string) error {
	if !validPriorities[priority] {
		return fmt.Errorf("invalid priority: %s (must be low, normal, high, or urgent)", priority)
	}
	return nil
}

// validateMsgType checks if a message type value is valid
func validateMsgType(msgType string) error {
	if !validMsgTypes[msgType] {
		return fmt.Errorf("invalid type: %s (must be message, request, response, or notification)", msgType)
	}
	return nil
}
