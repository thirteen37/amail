package tui

import (
	"testing"
	"time"
)

func TestSafeShortID(t *testing.T) {
	tests := []struct {
		name     string
		id       string
		expected string
	}{
		{
			name:     "long ID truncated to 8 chars",
			id:       "abcdef1234567890",
			expected: "abcdef12",
		},
		{
			name:     "exactly 8 chars unchanged",
			id:       "abcdef12",
			expected: "abcdef12",
		},
		{
			name:     "short ID unchanged",
			id:       "abc",
			expected: "abc",
		},
		{
			name:     "empty ID unchanged",
			id:       "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SafeShortID(tt.id)
			if result != tt.expected {
				t.Errorf("SafeShortID(%q) = %q, want %q", tt.id, result, tt.expected)
			}
		})
	}
}

func TestFormatTimeAgo(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		time     time.Time
		expected string
	}{
		{
			name:     "just now (seconds ago)",
			time:     now.Add(-30 * time.Second),
			expected: "now",
		},
		{
			name:     "1 minute ago",
			time:     now.Add(-1 * time.Minute),
			expected: "1m",
		},
		{
			name:     "15 minutes ago",
			time:     now.Add(-15 * time.Minute),
			expected: "15m",
		},
		{
			name:     "59 minutes ago",
			time:     now.Add(-59 * time.Minute),
			expected: "59m",
		},
		{
			name:     "1 hour ago",
			time:     now.Add(-1 * time.Hour),
			expected: "1h",
		},
		{
			name:     "5 hours ago",
			time:     now.Add(-5 * time.Hour),
			expected: "5h",
		},
		{
			name:     "23 hours ago",
			time:     now.Add(-23 * time.Hour),
			expected: "23h",
		},
		{
			name:     "1 day ago",
			time:     now.Add(-24 * time.Hour),
			expected: "1d",
		},
		{
			name:     "3 days ago",
			time:     now.Add(-3 * 24 * time.Hour),
			expected: "3d",
		},
		{
			name:     "6 days ago",
			time:     now.Add(-6 * 24 * time.Hour),
			expected: "6d",
		},
		{
			name:     "more than a week ago shows date",
			time:     now.Add(-10 * 24 * time.Hour),
			expected: now.Add(-10 * 24 * time.Hour).Format("Jan 2"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatTimeAgo(tt.time)
			if result != tt.expected {
				t.Errorf("formatTimeAgo() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestGenerateID(t *testing.T) {
	id1 := generateID()
	id2 := generateID()

	// Check length (8 bytes = 16 hex chars)
	if len(id1) != 16 {
		t.Errorf("generateID() length = %d, want 16", len(id1))
	}

	// Check uniqueness
	if id1 == id2 {
		t.Error("generateID() should generate unique IDs")
	}

	// Check it's valid hex
	for _, c := range id1 {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Errorf("generateID() contains invalid hex character: %c", c)
		}
	}
}
