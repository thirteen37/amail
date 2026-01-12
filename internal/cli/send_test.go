package cli

import (
	"testing"

	"github.com/thirteen37/amail/internal/config"
)

func TestResolveRecipients(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Agents.Roles = []string{"pm", "dev", "qa", "research"}
	cfg.Groups = map[string][]string{
		"engineers": {"dev", "qa"},
		"leads":     {"pm", "dev"},
	}

	tests := []struct {
		name      string
		toArg     string
		fromID    string
		wantErr   bool
		wantCount int
		wantRoles []string
	}{
		// Valid single recipients
		{"valid single dev", "dev", "pm", false, 1, []string{"dev"}},
		{"valid single pm", "pm", "dev", false, 1, []string{"pm"}},
		{"valid single qa", "qa", "pm", false, 1, []string{"qa"}},
		{"valid user", "user", "pm", false, 1, []string{"user"}},

		// Valid multiple recipients
		{"valid multiple", "dev,qa", "pm", false, 2, []string{"dev", "qa"}},
		{"valid multiple with spaces", "dev, qa, pm", "research", false, 3, nil},
		{"valid all roles", "pm,dev,qa,research", "user", false, 4, nil},

		// Invalid recipients
		{"invalid single", "invalid", "pm", true, 0, nil},
		{"invalid mixed", "dev,invalid,qa", "pm", true, 0, nil},
		{"invalid empty role", "", "pm", false, 0, nil},

		// Built-in groups
		{"group @all", "@all", "pm", false, 5, nil},  // pm, dev, qa, research, user
		{"group @agents", "@agents", "pm", false, 4, nil},  // pm, dev, qa, research
		{"group @others", "@others", "pm", false, 4, nil},  // dev, qa, research, user (excludes pm)

		// Custom groups
		{"custom group engineers", "@engineers", "pm", false, 2, []string{"dev", "qa"}},
		{"custom group leads", "@leads", "qa", false, 2, []string{"pm", "dev"}},

		// Invalid groups
		{"invalid group", "@unknown", "pm", true, 0, nil},
		{"not a group", "engineers", "pm", true, 0, nil},  // missing @ prefix

		// Deduplication
		{"duplicate recipients", "dev,dev,dev", "pm", false, 1, []string{"dev"}},
		{"overlapping groups", "@engineers,dev", "pm", false, 2, []string{"dev", "qa"}},

		// Edge cases
		{"empty string", "", "pm", false, 0, nil},
		{"only commas", ",,,", "pm", false, 0, nil},
		{"whitespace", "  ,  ,  ", "pm", false, 0, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := resolveRecipients(tt.toArg, tt.fromID, cfg)

			// Check error expectation
			if (err != nil) != tt.wantErr {
				t.Errorf("resolveRecipients(%q, %q) error = %v, wantErr %v", tt.toArg, tt.fromID, err, tt.wantErr)
				return
			}

			// Check count
			if len(result) != tt.wantCount {
				t.Errorf("resolveRecipients(%q, %q) returned %d recipients, want %d: %v", tt.toArg, tt.fromID, len(result), tt.wantCount, result)
				return
			}

			// Check specific roles if specified
			if tt.wantRoles != nil {
				for i, want := range tt.wantRoles {
					if i >= len(result) || result[i] != want {
						t.Errorf("resolveRecipients(%q, %q)[%d] = %q, want %q", tt.toArg, tt.fromID, i, result[i], want)
					}
				}
			}
		})
	}
}

func TestResolveRecipientsValidation(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Agents.Roles = []string{"dev", "qa"}

	// Test that invalid recipients are properly rejected
	invalidRecipients := []string{
		"admin",           // Not a defined role
		"root",            // Not a defined role
		"pm",              // Not in this config
		"Dev",             // Case sensitive - "Dev" != "dev"
		"dev@example.com", // Email-like string
		"dev;qa",          // Semicolon instead of comma
		"@",               // Just @ symbol
		"@@all",           // Double @
	}

	for _, recipient := range invalidRecipients {
		t.Run(recipient, func(t *testing.T) {
			_, err := resolveRecipients(recipient, "dev", cfg)
			if err == nil {
				t.Errorf("resolveRecipients(%q) should have returned error for invalid recipient", recipient)
			}
		})
	}
}

func TestFilterOut(t *testing.T) {
	tests := []struct {
		name     string
		slice    []string
		value    string
		expected []string
	}{
		{"remove existing", []string{"a", "b", "c"}, "b", []string{"a", "c"}},
		{"remove first", []string{"a", "b", "c"}, "a", []string{"b", "c"}},
		{"remove last", []string{"a", "b", "c"}, "c", []string{"a", "b"}},
		{"remove nonexistent", []string{"a", "b", "c"}, "d", []string{"a", "b", "c"}},
		{"empty slice", []string{}, "a", nil},
		{"single element remove", []string{"a"}, "a", nil},
		{"single element keep", []string{"a"}, "b", []string{"a"}},
		{"multiple occurrences", []string{"a", "b", "a", "c"}, "a", []string{"b", "c"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterOut(tt.slice, tt.value)
			if len(result) != len(tt.expected) {
				t.Errorf("filterOut(%v, %q) = %v, want %v", tt.slice, tt.value, result, tt.expected)
				return
			}
			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("filterOut(%v, %q)[%d] = %q, want %q", tt.slice, tt.value, i, v, tt.expected[i])
				}
			}
		})
	}
}
