# amail

A CLI-first mailbox system for multi-agent coordination.

`amail` provides async communication between AI agents (and humans) working on the same project. Each project has its own mailbox database. Agents identify by role (pm, dev, qa, etc.), not by session.

## Features

- **Per-project mailboxes** - SQLite database in `.amail/` directory
- **Role-based identity** - Agents are roles (dev, qa, pm), not sessions
- **Multi-recipient messages** - Send to individuals, multiple recipients, or groups
- **Threading** - Reply chains with full conversation history
- **Pluggable notifications** - Configure shell commands per priority level
- **Interactive TUI** - Terminal UI for browsing and composing messages
- **Claude Code skill** - Teach AI agents how to communicate

## Installation

### Pre-built Binaries

Download the latest release for your platform from [GitHub Releases](https://github.com/thirteen37/amail/releases/latest).

### Build from Source

```bash
git clone https://github.com/thirteen37/amail.git
cd amail
make build

# Install to GOPATH/bin
make install

# Install Claude Code skill (optional)
make install-skill
```

## Quick Start

### Project Setup (one-time)

```bash
cd ~/myproject
amail init --agents pm,dev,qa
```

### Session Workflow

```bash
# Set your identity
source <(amail use dev)

# Send a message
amail send pm "API ready" "GET /users endpoint implemented at routes/users.ts:45"

# Check inbox
amail inbox

# Read a message
amail read abc123

# Reply
amail reply abc123 "Thanks, I'll review it"

# Launch TUI
amail tui
```

## Commands

| Command | Description |
|---------|-------------|
| `amail init [--agents roles]` | Initialize project |
| `amail whoami` | Show current identity |
| `amail use <role>` | Set identity (use with `source`) |
| `amail send <to> <subject> <body>` | Send message |
| `amail inbox [-a]` | List messages |
| `amail read <id>` | Read message |
| `amail count` | Unread count |
| `amail reply <id> [--all] <body>` | Reply to message |
| `amail thread <id>` | View conversation thread |
| `amail mark-read <id\|--all>` | Mark as read |
| `amail archive <id>` | Archive message |
| `amail delete <id>` | Delete from inbox |
| `amail list` | List roles and groups |
| `amail stats` | Message statistics |
| `amail watch` | Watch for new messages |
| `amail check [--notify]` | One-shot check |
| `amail tui` | Interactive terminal UI |

## Output Formats

By default, amail outputs human-readable text in terminals and automatically switches to JSON when piped or redirected. This enables seamless integration with scripts and tools like `jq`.

```bash
# Human-readable in terminal
amail inbox

# Auto-JSON when piped
amail inbox | jq '.data.messages[0].id'

# Force JSON in terminal
amail inbox --json

# Force text when piped
amail inbox --text | cat
```

### JSON Envelope

All JSON output uses a consistent envelope:

```json
{
  "success": true,
  "data": { ... }
}
```

Errors:

```json
{
  "success": false,
  "error": {
    "message": "error description",
    "code": "ERROR_CODE"
  }
}
```

### Commands with JSON Support

Most read commands support JSON output:
- `inbox`, `read`, `thread`, `check`, `count`
- `list`, `stats`, `whoami`, `version`
- `send`, `reply` (return message ID and recipients)

Commands **without** JSON support (interactive/special):
- `init`, `use`, `tui`, `watch`
- `mark-read`, `archive`, `delete` (confirmations only)

## Recipients

```bash
# Single recipient
amail send dev "subject" "body"

# Multiple recipients
amail send dev,qa,pm "subject" "body"

# Built-in groups
amail send @all "subject" "body"      # All roles + user
amail send @agents "subject" "body"   # All agent roles
amail send @others "subject" "body"   # Everyone except sender

# Custom groups (defined in config)
amail send @engineers "subject" "body"
```

## Configuration

Project config at `.amail/config.toml`:

```toml
[agents]
roles = ["pm", "dev", "qa", "research"]

[groups]
engineers = ["dev", "qa"]
leads = ["pm", "dev"]

[identity.tmux]
# Map tmux session names to roles
"myproject-dev" = "dev"
"myproject-pm" = "pm"

[watch]
interval = 2  # polling interval in seconds

[notify.default]
commands = [
  "tmux display-message '📬 {from}: {subject}'"
]

[notify.urgent]
commands = [
  "tmux display-message '🚨 {from}: {subject}'",
  "terminal-notifier -title '🚨 {from}' -message '{body}'"
]
```

### Notification Variables

- `{id}` - Message ID
- `{from}` - Sender
- `{to}` - Recipients
- `{subject}` - Subject line
- `{body}` - Message body (truncated)
- `{priority}` - Priority level
- `{type}` - Message type
- `{timestamp}` - Time sent

## TUI Keybindings

| Key | Action |
|-----|--------|
| `↑/↓` or `j/k` | Navigate |
| `Enter` | Read message |
| `c` | Compose |
| `r` | Reply |
| `R` | Reply all |
| `d` | Delete |
| `m` | Mark read |
| `g` | Refresh |
| `Tab` | Switch mailbox |
| `Ctrl+S` | Send (compose mode) |
| `Esc/q` | Back/quit |

## Identity Resolution

Identity is resolved in order:

1. `$AMAIL_IDENTITY` environment variable
2. tmux session mapping from config
3. Prompt to register

```bash
# Set via environment
export AMAIL_IDENTITY=dev

# Or use the helper
source <(amail use dev)
```

## Multi-Agent Workflow Example

```bash
# Terminal 1: PM agent
source <(amail use pm)
amail send dev "Feature request" "Please implement user auth"

# Terminal 2: Dev agent
source <(amail use dev)
amail inbox
amail read --latest
amail send qa "Ready for testing" "Auth implemented in src/auth/"
amail reply abc123 "Done, sent to QA"

# Terminal 3: QA agent
source <(amail use qa)
amail inbox
amail send dev,pm "Tests passed" "All auth tests passing"
```

## Claude Code Integration

Install the skill:

```bash
make install-skill
```

This installs `SKILL.md` to `~/.claude/skills/amail/`. Claude Code will automatically use it when messaging intents are detected.

Example prompts Claude will understand:
- "Send a message to dev that the API is ready"
- "Check my inbox"
- "Reply to that message saying I'll handle it"
- "Notify the team that deployment is complete"

## Development

```bash
make build      # Build binary
make test       # Run tests
make coverage   # Test coverage report
make demo       # Run demo
make clean      # Clean build artifacts
```

## License

MIT

## Acknowledgments

Inspired by [AI Maestro](https://github.com/23blocks-OS/ai-maestro) and the concept of agent-to-agent communication for multi-agent AI workflows.

Built with:
- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [Bubbletea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [Lipgloss](https://github.com/charmbracelet/lipgloss) - TUI styling
- [modernc.org/sqlite](https://modernc.org/sqlite) - Pure Go SQLite
