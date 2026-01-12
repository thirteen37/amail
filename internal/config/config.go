package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Config represents the project configuration
type Config struct {
	Agents   AgentsConfig            `toml:"agents"`
	Groups   map[string][]string     `toml:"groups"`
	Identity IdentityConfig          `toml:"identity"`
	Watch    WatchConfig             `toml:"watch"`
	Notify   map[string]NotifyConfig `toml:"notify"`
}

// AgentsConfig defines the agent roles for the project
type AgentsConfig struct {
	Roles []string `toml:"roles"`
}

// IdentityConfig handles identity mapping
type IdentityConfig struct {
	Tmux map[string]string `toml:"tmux"`
}

// WatchConfig defines watch/polling settings
type WatchConfig struct {
	Interval int `toml:"interval"`
}

// NotifyConfig defines notification commands for a priority level
type NotifyConfig struct {
	Commands []string `toml:"commands"`
}

// DefaultConfig returns a new config with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		Agents: AgentsConfig{
			Roles: []string{},
		},
		Groups:   make(map[string][]string),
		Identity: IdentityConfig{
			Tmux: make(map[string]string),
		},
		Watch: WatchConfig{
			Interval: 2,
		},
		Notify: map[string]NotifyConfig{
			"default": {
				Commands: []string{"echo '📬 New message from {from}: {subject}'"},
			},
		},
	}
}

// Load reads the config from the given path
func Load(path string) (*Config, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	if _, err := toml.Decode(string(data), cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return cfg, nil
}

// Save writes the config to the given path
func (c *Config) Save(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer f.Close()

	encoder := toml.NewEncoder(f)
	if err := encoder.Encode(c); err != nil {
		return fmt.Errorf("failed to encode config: %w", err)
	}

	return nil
}

// ConfigPath returns the config path for a project root
func ConfigPath(projectRoot string) string {
	return filepath.Join(projectRoot, ".amail", "config.toml")
}

// LoadProject loads the config for the given project root
func LoadProject(projectRoot string) (*Config, error) {
	return Load(ConfigPath(projectRoot))
}

// AllRoles returns all defined roles plus the reserved "user" role
func (c *Config) AllRoles() []string {
	roles := make([]string, len(c.Agents.Roles)+1)
	copy(roles, c.Agents.Roles)
	roles[len(c.Agents.Roles)] = "user"
	return roles
}

// IsValidRole checks if a role is defined or is the reserved "user" role
func (c *Config) IsValidRole(role string) bool {
	if role == "user" {
		return true
	}
	for _, r := range c.Agents.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// ResolveGroup resolves a group name (with @ prefix) to its members
// Returns nil if not a group or group not found
func (c *Config) ResolveGroup(name string, currentIdentity string) []string {
	if len(name) == 0 || name[0] != '@' {
		return nil
	}

	groupName := name[1:]

	// Built-in groups
	switch groupName {
	case "all":
		return c.AllRoles()
	case "agents":
		// All roles except user
		return c.Agents.Roles
	case "others":
		// All roles except current identity
		var others []string
		for _, r := range c.AllRoles() {
			if r != currentIdentity {
				others = append(others, r)
			}
		}
		return others
	}

	// Custom groups
	if members, ok := c.Groups[groupName]; ok {
		return members
	}

	return nil
}

// GetNotifyCommands returns the notification commands for a priority level
func (c *Config) GetNotifyCommands(priority string) []string {
	if cfg, ok := c.Notify[priority]; ok {
		return cfg.Commands
	}
	if cfg, ok := c.Notify["default"]; ok {
		return cfg.Commands
	}
	return nil
}

// GenerateDefaultConfigContent generates a default config file content
func GenerateDefaultConfigContent(roles []string) string {
	content := `# amail project configuration

[agents]
roles = [`

	for i, role := range roles {
		if i > 0 {
			content += ", "
		}
		content += fmt.Sprintf("%q", role)
	}

	content += `]

[groups]
# Define custom groups
# engineers = ["dev", "qa"]
# leads = ["pm", "dev"]

[identity.tmux]
# Map tmux session names to roles
# "myproject-dev" = "dev"
# "myproject-pm" = "pm"

[watch]
interval = 2  # polling interval in seconds

[notify.default]
commands = [
  "echo '📬 New message from {from}: {subject}'"
]

[notify.high]
commands = [
  "echo '📬 {from}: {subject}'"
]

[notify.urgent]
commands = [
  "echo '🚨 URGENT from {from}: {subject}'"
]
`
	return content
}
