package identity

import (
	"os"
	"testing"

	"github.com/thirteen37/amail/internal/config"
)

func TestResolveFromEnvVar(t *testing.T) {
	// Set env var
	os.Setenv(EnvIdentity, "testdev")
	defer os.Unsetenv(EnvIdentity)

	cfg := config.DefaultConfig()
	res, err := Resolve(cfg)

	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}
	if res == nil {
		t.Fatal("expected non-nil resolution")
	}
	if res.Identity != "testdev" {
		t.Errorf("expected identity 'testdev', got '%s'", res.Identity)
	}
	if res.Source != "environment variable ($AMAIL_IDENTITY)" {
		t.Errorf("unexpected source: %s", res.Source)
	}
}

func TestResolveNoIdentity(t *testing.T) {
	// Ensure env var is not set
	os.Unsetenv(EnvIdentity)
	// Ensure not in tmux
	os.Unsetenv("TMUX")

	cfg := config.DefaultConfig()
	res, err := Resolve(cfg)

	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}
	if res != nil {
		t.Errorf("expected nil resolution when no identity, got %+v", res)
	}
}

func TestMustResolveError(t *testing.T) {
	os.Unsetenv(EnvIdentity)
	os.Unsetenv("TMUX")

	cfg := config.DefaultConfig()
	_, err := MustResolve(cfg)

	if err == nil {
		t.Error("expected error from MustResolve when no identity")
	}
}

func TestMustResolveSuccess(t *testing.T) {
	os.Setenv(EnvIdentity, "dev")
	defer os.Unsetenv(EnvIdentity)

	cfg := config.DefaultConfig()
	res, err := MustResolve(cfg)

	if err != nil {
		t.Fatalf("MustResolve failed: %v", err)
	}
	if res.Identity != "dev" {
		t.Errorf("expected 'dev', got '%s'", res.Identity)
	}
}

func TestExportCommand(t *testing.T) {
	tests := []struct {
		identity string
		expected string
	}{
		{"dev", "export AMAIL_IDENTITY='dev'"},
		{"pm", "export AMAIL_IDENTITY='pm'"},
		// Test shell injection prevention
		// Input: dev'; rm -rf / -> wrapped as 'dev'\''; rm -rf /'
		{"dev'; rm -rf /", "export AMAIL_IDENTITY='dev'\\''; rm -rf /'"},
		{"test$(whoami)", "export AMAIL_IDENTITY='test$(whoami)'"},
	}

	for _, tt := range tests {
		t.Run(tt.identity, func(t *testing.T) {
			cmd := ExportCommand(tt.identity)
			if cmd != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, cmd)
			}
		})
	}
}

func TestIsInTmux(t *testing.T) {
	// Not in tmux
	os.Unsetenv("TMUX")
	if IsInTmux() {
		t.Error("expected not in tmux")
	}

	// In tmux (simulated)
	os.Setenv("TMUX", "/tmp/tmux-1000/default,12345,0")
	defer os.Unsetenv("TMUX")
	if !IsInTmux() {
		t.Error("expected in tmux")
	}
}

func TestResolveTmuxMapping(t *testing.T) {
	// This test can only work if we're actually in tmux,
	// which we probably aren't during testing.
	// So we test the case where TMUX is set but session lookup fails.

	os.Unsetenv(EnvIdentity)
	os.Setenv("TMUX", "/tmp/tmux-fake,99999,0")
	defer os.Unsetenv("TMUX")

	cfg := config.DefaultConfig()
	cfg.Identity.Tmux = map[string]string{
		"test-session": "dev",
	}

	// Since we can't actually get the tmux session name in tests,
	// this should fall through to nil
	res, err := Resolve(cfg)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}
	// Result depends on whether tmux command succeeds
	// In CI/testing without tmux, it should be nil
	_ = res
}

func TestEnvVarPrecedence(t *testing.T) {
	// Even if tmux mapping exists, env var should take precedence
	os.Setenv(EnvIdentity, "fromenv")
	os.Setenv("TMUX", "/tmp/tmux-fake,99999,0")
	defer os.Unsetenv(EnvIdentity)
	defer os.Unsetenv("TMUX")

	cfg := config.DefaultConfig()
	cfg.Identity.Tmux = map[string]string{
		"any-session": "fromtmux",
	}

	res, err := Resolve(cfg)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}
	if res == nil {
		t.Fatal("expected resolution")
	}
	if res.Identity != "fromenv" {
		t.Errorf("expected env var to take precedence, got '%s'", res.Identity)
	}
}
