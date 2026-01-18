# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

> **Never work directly on main.** Always use `/workbranch` (or `git worktree`) first.

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
| `internal/cli` | Cobra command handlers, one file per command, JSON/text output helpers |
| `internal/db` | SQLite persistence, `Message` and `Recipient` types |
| `internal/config` | TOML config loading, role/group validation |
| `internal/identity` | Identity resolution chain (env var → tmux mapping → undefined) |
| `internal/notify` | Shell command execution with template substitution |
| `internal/tui` | Bubbletea terminal UI |

### Database Schema

- `messages` - Core message storage with threading support (thread_id, reply_to_id)
- `recipients` - Per-recipient read status (message_id + to_id compound PK)

### Database Concurrency

The database uses SQLite WAL mode for safe multi-process access:

- **WAL mode**: Enables concurrent readers during writes
- **Busy timeout**: 5-second wait on lock contention (vs immediate failure)
- **Checkpoint on close**: Minimizes WAL file size when closing gracefully

Files created by WAL mode:
- `.amail/mail.db-wal` - Write-ahead log
- `.amail/mail.db-shm` - Shared memory (coordinates processes)

These files are normal and will be cleaned up on checkpoint.

### Identity Resolution Order

1. `$AMAIL_IDENTITY` environment variable
2. Tmux session mapping from `.amail/config.toml`
3. Undefined (commands that require identity will error)

## Key Patterns

- **Project discovery:** Commands search parent directories for `.amail/` to find project root
- **Notification safety:** Template variables passed via environment to prevent shell injection
- **Lazy identity:** Identity only resolved when a command actually needs it
- **Smart output format:** JSON when piped/redirected, human-readable for terminals

### Output Format Pattern

Commands support dual output formats via `internal/cli/output.go`:

```go
// Check output mode (respects --json/--text flags and TTY detection)
if IsJSONOutput() {
    return PrintJSON(MyOutput{...})
}
// Text output follows...
```

- `IsJSONOutput()` - Returns true if JSON output should be used
- `PrintJSON(data)` - Wraps data in `{"success": true, "data": ...}`
- `PrintJSONError(err, code)` - Wraps error in `{"success": false, "error": ...}`

Each command defines its own output struct (e.g., `InboxOutput`, `ReadOutput`) in its file.

## Git Workflow

**Agents must never work directly on main.** Always use the `/workbranch` skill (or `git worktree` if the skill is unavailable) to create an isolated branch before making changes.

This enables:
- Multiple agents working in parallel without conflicts
- Clean rollbacks if changes need to be reverted
- Safe experimentation without affecting the main branch
