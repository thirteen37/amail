package tui

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

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

func timeNow() time.Time {
	return time.Now()
}

func formatTimeAgo(t time.Time) string {
	duration := time.Since(t)

	switch {
	case duration < time.Minute:
		return "now"
	case duration < time.Hour:
		mins := int(duration.Minutes())
		if mins == 1 {
			return "1m"
		}
		return string(rune(mins/10+'0')) + string(rune(mins%10+'0')) + "m"
	case duration < 24*time.Hour:
		hours := int(duration.Hours())
		if hours == 1 {
			return "1h"
		}
		return string(rune(hours/10+'0')) + string(rune(hours%10+'0')) + "h"
	case duration < 7*24*time.Hour:
		days := int(duration.Hours() / 24)
		if days == 1 {
			return "1d"
		}
		return string(rune(days+'0')) + "d"
	default:
		return t.Format("Jan 2")
	}
}
