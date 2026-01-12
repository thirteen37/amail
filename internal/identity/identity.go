package identity

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/thirteen37/amail/internal/config"
)

const (
	// EnvIdentity is the environment variable for explicit identity
	EnvIdentity = "AMAIL_IDENTITY"
)

// Resolution represents the result of identity resolution
type Resolution struct {
	Identity string
	Source   string
}

// Resolve determines the current identity using the priority chain:
// 1. AMAIL_IDENTITY env var
// 2. tmux session mapping from config
// 3. Returns empty if not found
func Resolve(cfg *config.Config) (*Resolution, error) {
	// 1. Check environment variable
	if id := os.Getenv(EnvIdentity); id != "" {
		return &Resolution{
			Identity: id,
			Source:   "environment variable ($AMAIL_IDENTITY)",
		}, nil
	}

	// 2. Check tmux session mapping
	if tmuxSession := getTmuxSession(); tmuxSession != "" {
		if cfg != nil && cfg.Identity.Tmux != nil {
			if id, ok := cfg.Identity.Tmux[tmuxSession]; ok {
				return &Resolution{
					Identity: id,
					Source:   fmt.Sprintf("tmux session mapping (%s)", tmuxSession),
				}, nil
			}
		}
	}

	// 3. Not resolved
	return nil, nil
}

// getTmuxSession returns the current tmux session name, or empty if not in tmux
func getTmuxSession() string {
	// Check if we're in tmux
	if os.Getenv("TMUX") == "" {
		return ""
	}

	// Get session name
	cmd := exec.Command("tmux", "display-message", "-p", "#S")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(out))
}

// MustResolve resolves identity and returns an error if not found
func MustResolve(cfg *config.Config) (*Resolution, error) {
	res, err := Resolve(cfg)
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, fmt.Errorf("identity not set. Use 'source <(amail use <role>)' or set $AMAIL_IDENTITY")
	}
	return res, nil
}

// ExportCommand generates the shell command to export identity
// The identity is escaped to prevent shell injection
func ExportCommand(identity string) string {
	// Escape shell-unsafe characters by wrapping in single quotes
	// and escaping any embedded single quotes
	safeIdentity := "'" + strings.ReplaceAll(identity, "'", "'\\''") + "'"
	return fmt.Sprintf("export %s=%s", EnvIdentity, safeIdentity)
}

// GetTmuxSession returns the current tmux session name (exported for use elsewhere)
func GetTmuxSession() string {
	return getTmuxSession()
}

// IsInTmux returns true if running inside tmux
func IsInTmux() bool {
	return os.Getenv("TMUX") != ""
}
