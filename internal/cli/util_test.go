package cli

import (
	"testing"
)

func TestSafeShortID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty string", "", ""},
		{"length 1", "a", "a"},
		{"length 7", "1234567", "1234567"},
		{"length 8 exactly", "12345678", "12345678"},
		{"length 9", "123456789", "12345678"},
		{"length 16 (typical ID)", "1234567890abcdef", "12345678"},
		{"length 32", "1234567890abcdef1234567890abcdef", "12345678"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SafeShortID(tt.input)
			if result != tt.expected {
				t.Errorf("SafeShortID(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		// Basic cases
		{"empty string", "", 10, ""},
		{"short string", "hello", 10, "hello"},
		{"exact length", "hello", 5, "hello"},
		{"truncate needed", "hello world", 8, "hello..."},
		{"very short max", "hello", 3, "hel"},
		{"max 0", "hello", 0, ""},

		// UTF-8 handling - truncate by runes, not bytes
		{"japanese short", "日本語", 5, "日本語"},
		{"japanese truncate", "日本語テスト", 5, "日本..."},
		{"mixed ascii unicode", "Hello世界", 8, "Hello世界"},
		{"mixed truncate", "Hello世界Test", 8, "Hello..."},
		{"emojis short", "🚀🎉🎊", 5, "🚀🎉🎊"},
		{"emojis truncate", "🚀🎉🎊🎁🎄", 4, "🚀..."},
		{"single emoji", "🚀", 1, "🚀"},

		// Edge cases around ellipsis
		{"max 1", "hello", 1, "h"},
		{"max 2", "hello", 2, "he"},
		{"max 3", "hello", 3, "hel"},
		{"max 4 truncate", "hello", 4, "h..."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncate(tt.input, tt.maxLen)
			if result != tt.expected {
				t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, result, tt.expected)
			}
		})
	}
}

func TestGenerateID(t *testing.T) {
	// Test that generateID returns a 16-character hex string
	id := generateID()
	if len(id) != 16 {
		t.Errorf("generateID() returned %q with length %d, want 16", id, len(id))
	}

	// Test uniqueness (generate multiple IDs)
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := generateID()
		if ids[id] {
			t.Errorf("generateID() returned duplicate ID: %s", id)
		}
		ids[id] = true
	}
}

func TestParseRecipients(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{"single recipient", "dev", []string{"dev"}},
		{"multiple recipients", "dev,qa,pm", []string{"dev", "qa", "pm"}},
		{"with spaces", "dev, qa, pm", []string{"dev", "qa", "pm"}},
		{"empty parts", "dev,,qa", []string{"dev", "qa"}},
		{"empty string", "", nil},
		{"only commas", ",,,", nil},
		{"whitespace only", "  ,  ,  ", nil},
		{"leading/trailing spaces", "  dev  ,  qa  ", []string{"dev", "qa"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseRecipients(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("parseRecipients(%q) returned %d items, want %d", tt.input, len(result), len(tt.expected))
				return
			}
			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("parseRecipients(%q)[%d] = %q, want %q", tt.input, i, v, tt.expected[i])
				}
			}
		})
	}
}
