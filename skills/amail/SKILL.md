---
name: amail
description: This skill should be used when the user asks to "send a message to an agent", "check inbox", "notify another agent", "tell dev about...", "ask qa...", "read my messages", "any new messages?", or mentions inter-agent communication. At session start, use this skill to establish identity and check for pending messages.
version: 1.0.0
---

# amail - Agent Communication Skill

A CLI-based mailbox system for multi-agent coordination. Each project has its own message database, and agents identify by role (pm, dev, qa, etc.).

## When to Use

**Sending messages:**
- User says "send a message to [agent]" or "notify [agent]"
- User says "tell [agent] about..." or "ask [agent]..."
- Need to communicate with another agent session
- Need to notify the user asynchronously

**Receiving messages:**
- User says "check inbox" or "check messages"
- User says "read my messages" or "any new messages?"
- At session start, check if other agents have sent requests

## Setup

### Establish Identity

At session start, establish identity:

1. Check if identity is already set:
   ```bash
   amail whoami
   ```

2. If identity is not set, list available roles:
   ```bash
   amail list
   ```

3. Pick the role that matches the current task:
   - `dev` - coding, implementation, debugging
   - `qa` - testing, validation, quality checks
   - `pm` - planning, coordination, requirements
   - `research` - investigation, exploration, documentation

   If unclear from task context, prompt user for role selection.

4. Set identity:
   ```bash
   source <(amail use <role>)
   ```

## Commands Reference

### Sending Messages

```bash
# Basic send
amail send <to> "<subject>" "<body>"

# Send with priority
amail send <to> -p urgent "<subject>" "<body>"
amail send <to> -p high "<subject>" "<body>"

# Send to multiple recipients
amail send dev,qa "<subject>" "<body>"

# Send to groups
amail send @all "<subject>" "<body>"      # All roles + user
amail send @agents "<subject>" "<body>"   # All agent roles
amail send @others "<subject>" "<body>"   # Everyone except sender

# Send to user (human operator)
amail send user "<subject>" "<body>"
```

### Checking Messages

```bash
# List unread messages
amail inbox

# List all messages (including read)
amail inbox -a

# Filter by sender
amail inbox --from pm

# Get unread count (useful for status checks)
amail count

# Read a specific message
amail read <message-id>

# Read the most recent unread
amail read --latest
```

### Replying

```bash
# Reply to sender only
amail reply <message-id> "<body>"

# Reply to sender + all original recipients
amail reply <message-id> --all "<body>"
```

### Message Management

```bash
# Mark as read
amail mark-read <message-id>
amail mark-read --all

# Archive a message
amail archive <message-id>

# Delete from inbox
amail delete <message-id>
```

### Viewing Threads

```bash
# View all messages in a thread
amail thread <message-id>
```

### Other Commands

```bash
# List all roles and groups
amail list

# Show message statistics
amail stats

# Launch interactive TUI
amail tui
```

## JSON Output

Commands automatically output JSON when piped (useful for parsing):

```bash
# Get message ID from inbox
amail inbox | jq '.data.messages[0].id'

# Force JSON in terminal
amail inbox --json

# Force text when piped
amail inbox --text | cat
```

JSON envelope format:
```json
{
  "success": true,
  "data": { ... }
}
```

Commands with JSON support: `inbox`, `read`, `thread`, `check`, `count`, `list`, `stats`, `whoami`, `version`, `send`, `reply`

## Message Types and Priorities

### Priorities

- `low` - FYI, no action needed
- `normal` - Standard communication (default)
- `high` - Important, needs attention soon
- `urgent` - Critical, immediate attention needed

### Types (use with `-t`)

- `message` - General communication (default)
- `request` - Asking for work/action
- `response` - Replying to a request
- `notification` - Status update, no response expected

## Best Practices

1. **Check inbox at session start** - Other agents may have sent requests
2. **Use meaningful subjects** - Makes inbox scanning easier
3. **Include context in body** - File paths, line numbers, specifics
4. **Use appropriate priority** - Reserve urgent for truly critical items
5. **Reply when work is done** - Close the communication loop
6. **Send to user for decisions** - When human input is needed

## Identity Boundaries

**Inbox scope** = Messages sent TO current role by other agents

**Accessible:**
- Messages in own inbox
- Threads participated in

**Not accessible:**
- Other agents' private inboxes
- Messages not addressed to current role

## Troubleshooting

### "amail: command not found"

Download from GitHub releases to a local directory:

```bash
VERSION=$(curl -s https://api.github.com/repos/thirteen37/amail/releases/latest | grep -o '"tag_name": "[^"]*' | cut -d'"' -f4)
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m); [[ "$ARCH" == "x86_64" ]] && ARCH="amd64"; [[ "$ARCH" == "aarch64" || "$ARCH" == "arm64" ]] && ARCH="arm64"
curl -sLO "https://github.com/thirteen37/amail/releases/download/${VERSION}/amail_${VERSION#v}_${OS}_${ARCH}.tar.gz"
tar xzf amail_*.tar.gz && rm amail_*.tar.gz
```

Then use `./amail` or add to PATH: `export PATH="$PWD:$PATH"`

### "not in an amail project"

Run `amail init` to initialize the project, or navigate to the project root.

### "identity not set"

Run `source <(amail use <role>)` or set `export AMAIL_IDENTITY=<role>`

### Message not found

Use the first 8 characters of the message ID, e.g., `amail read abc12345`
