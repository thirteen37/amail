package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
	if cfg.Watch.Interval != 2 {
		t.Errorf("expected default interval 2, got %d", cfg.Watch.Interval)
	}
	if len(cfg.Notify) == 0 {
		t.Error("expected default notify config")
	}
}

func TestLoadNonExistent(t *testing.T) {
	cfg, err := Load("/nonexistent/path/config.toml")
	if err != nil {
		t.Fatalf("Load should not error on nonexistent file: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected default config")
	}
}

func TestLoadAndSave(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "amail-config-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	configPath := filepath.Join(tmpDir, "config.toml")

	// Create config
	cfg := DefaultConfig()
	cfg.Agents.Roles = []string{"pm", "dev", "qa"}
	cfg.Groups = map[string][]string{
		"engineers": {"dev", "qa"},
	}
	cfg.Watch.Interval = 5

	// Save
	err = cfg.Save(configPath)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Load
	loaded, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Verify
	if len(loaded.Agents.Roles) != 3 {
		t.Errorf("expected 3 roles, got %d", len(loaded.Agents.Roles))
	}
	if loaded.Watch.Interval != 5 {
		t.Errorf("expected interval 5, got %d", loaded.Watch.Interval)
	}
	if engineers, ok := loaded.Groups["engineers"]; !ok {
		t.Error("expected engineers group")
	} else if len(engineers) != 2 {
		t.Errorf("expected 2 members in engineers, got %d", len(engineers))
	}
}

func TestAllRoles(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Agents.Roles = []string{"pm", "dev", "qa"}

	allRoles := cfg.AllRoles()

	if len(allRoles) != 4 {
		t.Errorf("expected 4 roles (3 + user), got %d", len(allRoles))
	}

	// user should be included
	hasUser := false
	for _, r := range allRoles {
		if r == "user" {
			hasUser = true
			break
		}
	}
	if !hasUser {
		t.Error("expected 'user' in AllRoles")
	}
}

func TestIsValidRole(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Agents.Roles = []string{"pm", "dev", "qa"}

	tests := []struct {
		role  string
		valid bool
	}{
		{"pm", true},
		{"dev", true},
		{"qa", true},
		{"user", true},      // reserved
		{"unknown", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.role, func(t *testing.T) {
			if cfg.IsValidRole(tt.role) != tt.valid {
				t.Errorf("IsValidRole(%q) = %v, want %v", tt.role, !tt.valid, tt.valid)
			}
		})
	}
}

func TestResolveGroup(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Agents.Roles = []string{"pm", "dev", "qa"}
	cfg.Groups = map[string][]string{
		"engineers": {"dev", "qa"},
		"leads":     {"pm", "dev"},
	}

	tests := []struct {
		name     string
		identity string
		expected []string
	}{
		{"@all", "pm", []string{"pm", "dev", "qa", "user"}},
		{"@agents", "pm", []string{"pm", "dev", "qa"}},
		{"@others", "pm", []string{"dev", "qa", "user"}},
		{"@others", "dev", []string{"pm", "qa", "user"}},
		{"@engineers", "pm", []string{"dev", "qa"}},
		{"@leads", "qa", []string{"pm", "dev"}},
		{"@unknown", "pm", nil},
		{"notagroup", "pm", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cfg.ResolveGroup(tt.name, tt.identity)
			if tt.expected == nil {
				if result != nil {
					t.Errorf("expected nil, got %v", result)
				}
				return
			}
			if len(result) != len(tt.expected) {
				t.Errorf("expected %d members, got %d: %v", len(tt.expected), len(result), result)
			}
		})
	}
}

func TestGetNotifyCommands(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Notify = map[string]NotifyConfig{
		"default": {Commands: []string{"echo default"}},
		"urgent":  {Commands: []string{"echo urgent1", "echo urgent2"}},
	}

	// Existing priority
	commands := cfg.GetNotifyCommands("urgent")
	if len(commands) != 2 {
		t.Errorf("expected 2 urgent commands, got %d", len(commands))
	}

	// Fall back to default
	commands = cfg.GetNotifyCommands("normal")
	if len(commands) != 1 {
		t.Errorf("expected 1 default command, got %d", len(commands))
	}
	if commands[0] != "echo default" {
		t.Errorf("expected 'echo default', got '%s'", commands[0])
	}

	// Unknown priority with no default
	cfg.Notify = map[string]NotifyConfig{}
	commands = cfg.GetNotifyCommands("normal")
	if commands != nil {
		t.Errorf("expected nil, got %v", commands)
	}
}

func TestGenerateDefaultConfigContent(t *testing.T) {
	content := GenerateDefaultConfigContent([]string{"pm", "dev", "qa"})

	if content == "" {
		t.Fatal("expected non-empty content")
	}

	// Should contain roles
	if !contains(content, `"pm"`) {
		t.Error("expected content to contain pm role")
	}
	if !contains(content, `"dev"`) {
		t.Error("expected content to contain dev role")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
