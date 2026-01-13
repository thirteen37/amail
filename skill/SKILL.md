# amail - Agent Communication Skill

Use `amail` for agent-to-agent and agent-to-user messaging within the current project.

## Overview

`amail` is a CLI-based mailbox system for multi-agent coordination. Each project has its own message database, and agents identify by role (pm, dev, qa, etc.).

## When to Use This Skill

**Sending messages:**
- User says "send a message to [agent]" or "notify [agent]"
- User says "tell [agent] about..." or "ask [agent]..."
- You need to communicate with another agent session
- You need to notify the user asynchronously

**Receiving messages:**
- User says "check inbox" or "check messages"
- User says "read my messages" or "any new messages?"
- At the start of a task, to see if other agents have sent requests

## Setup

### Establish Your Identity

At the start of a session, establish your identity:

1. Check if identity is already set:
   ```bash
   amail whoami
   ```

2. If identity is not set, see what roles are available:
   ```bash
   amail list
   ```

3. Pick the role that matches your current task:
   - `dev` - coding, implementation, debugging
   - `qa` - testing, validation, quality checks
   - `pm` - planning, coordination, requirements
   - `research` - investigation, exploration, documentation

   If unclear from your task context, ask the user which role to use.

4. Set your identity:
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
amail send @others "<subject>" "<body>"   # Everyone except you

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

# Delete from your inbox
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

## Examples

### Example 1: Notifying Another Agent

When you've completed a task that another agent needs to know about:

```bash
amail send qa "Feature ready for testing" "Implemented user authentication at src/auth/. Please run integration tests."
```

### Example 2: Requesting Work

When you need another agent to do something:

```bash
amail send dev -p high -t request "Need API endpoint" "Please implement GET /api/users returning {id, name, email}. Spec in docs/api.md"
```

### Example 3: Starting a Session

Good practice at the start of any task:

```bash
# Check/establish identity
amail whoami
# If not set: amail list, then source <(amail use dev)

# Check for messages
amail count
amail inbox
amail read --latest
```

### Example 4: Asking the User a Question

When you need clarification from the human:

```bash
amail send user "Question: Auth approach" "Should we use JWT or session-based auth? JWT is simpler but sessions offer better revocation."
```

### Example 5: Broadcast Announcement

Notify all agents:

```bash
amail send @agents "Build complete" "Main branch CI passed. Ready for deployment review."
```

### Example 6: Replying to a Request

After receiving a request and completing it:

```bash
amail reply abc123 "Done. Implemented GET /api/users at routes/users.ts:45. Returns paginated results with ?page=1&limit=20"
```

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
4. **Use appropriate priority** - Don't cry wolf with urgent
5. **Reply when work is done** - Close the communication loop
6. **Send to user for decisions** - When you need human input

## Identity Boundaries

**Your inbox** = Messages sent TO YOUR role by other agents

**You can read:**
- ✅ Messages in your own inbox
- ✅ Threads you're part of

**You should NOT read:**
- ❌ Other agents' private inboxes
- ❌ Messages not addressed to your role

## Troubleshooting

### "not in an amail project"

Run `amail init` to initialize the project, or `cd` to the project root.

### "identity not set"

Run `source <(amail use <role>)` or set `export AMAIL_IDENTITY=<role>`

### Message not found

Use the first 8 characters of the message ID, e.g., `amail read abc12345`
