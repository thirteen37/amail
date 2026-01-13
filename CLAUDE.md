# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build Commands

```bash
make build          # Build binary to ./amail
make test           # Run all tests
make coverage       # Generate coverage report
make install        # Install to GOPATH/bin
make install-skill  # Install Claude Code skill to ~/.claude/skills/amail/
```

Run a single test:
```bash
go test -v ./internal/db -run TestSendMessage
```

## Architecture

**Module:** `github.com/thirteen37/amail`

```
cmd/amail/main.go → internal/cli/root.go (Cobra command tree)
                        ↓
    ┌───────────────────┼───────────────────┐
    ↓                   ↓                   ↓
identity.Resolve()  config.LoadProject()  db.OpenProject()
(env/tmux/none)     (.amail/config.toml)  (.amail/mail.db)
```

### Package Responsibilities

| Package | Purpose |
|---------|---------|
| `internal/cli` | Cobra command handlers, one file per command |
| `internal/db` | SQLite persistence, `Message` and `Recipient` types |
| `internal/config` | TOML config loading, role/group validation |
| `internal/identity` | Identity resolution chain (env var → tmux mapping → undefined) |
| `internal/notify` | Shell command execution with template substitution |
| `internal/tui` | Bubbletea terminal UI |

### Database Schema

- `messages` - Core message storage with threading support (thread_id, reply_to_id)
- `recipients` - Per-recipient read status (message_id + to_id compound PK)

### Identity Resolution Order

1. `$AMAIL_IDENTITY` environment variable
2. Tmux session mapping from `.amail/config.toml`
3. Undefined (commands that require identity will error)

## Key Patterns

- **Project discovery:** Commands search parent directories for `.amail/` to find project root
- **Notification safety:** Template variables passed via environment to prevent shell injection
- **Lazy identity:** Identity only resolved when a command actually needs it
