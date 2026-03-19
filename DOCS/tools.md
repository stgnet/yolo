# YOLO Tools Reference

Complete catalog of all available tools in YOLO with usage examples and parameters.

**Quick Start**: See [README.md](../README.md)  
**Documentation Hub**: [DOCS/README.md](README.md)  
**Autonomous Operations**: [AUTONOMOUS_OPERATIONS.md](AUTONOMOUS_OPERATIONS.md)

---

## 📁 File Operations

| Tool | Description | Parameters |
|------|---|--------|
| `read_file` | Read file contents | `path`, `offset` (line, 1-based), `limit` (max lines) |
| `write_file` | Create or overwrite file | `path`, `content` |
| `edit_file` | Replace text in file | `path`, `old_text`, `new_text` |
| `list_files` | List files matching glob | `pattern` (default: `*`) |
| `search_files` | Search file contents with regex | `query` (required), `pattern` |
| `make_dir` | Create directory recursively | `path` |
| `remove_dir` | Remove directory and contents | `path` |
| `copy_file` | Copy file (creates dirs if needed) | `source`, `dest` |
| `move_file` | Move file (creates dirs if needed) | `source`, `dest` |

### Examples

```json
// Read first 50 lines of a file
{
  "name": "read_file",
  "arguments": {
    "path": "src/main.go",
    "offset": 1,
    "limit": 50
  }
}

// Write new file
{
  "name": "write_file",
  "arguments": {
    "path": "new_feature.go",
    "content": "package main\n\nfunc NewFeature() {}"
  }
}

// Edit file (replace first occurrence)
{
  "name": "edit_file",
  "arguments": {
    "path": "config.go",
    "old_text": "const Timeout = 30",
    "new_text": "const Timeout = 60"
  }
}

// Search for files
{
  "name": "list_files",
  "arguments": {
    "pattern": "**/*.go"
  }
}

// Grep-like search
{
  "name": "search_files",
  "arguments": {
    "query": "func.*\\(.*context\\.Context",
    "pattern": "**/*.go"
  }
}
```

---

## ⚙️ System & Execution

| Tool | Description | Parameters |
|------|---|--------|
| `run_command` | Execute shell command (30s timeout) | `command` |
| `restart` | Rebuild and restart YOLO | (none) |
| `think` | Record reasoning without action | `thought` |

### Examples

```json
// Run build command
{
  "name": "run_command",
  "arguments": {
    "command": "go build -o /tmp/yolo . && echo 'Build successful'"
  }
}

// Internal reasoning (for planning)
{
  "name": "think",
  "arguments": {
    "thought": "Need to check if race condition exists before fixing"
  }
}

// Restart after code changes
{"name": "restart", "arguments": {}}
```

---

## 🤖 AI & Model Management

| Tool | Description | Parameters |
|------|---|--------|
| `list_models` | List available Ollama models | (none) |
| `switch_model` | Change active model | `model` |
| `learn` | Research improvements online | (optional params) |
| `implement` | Auto-implement learned improvements | `count` (default: 2) |

### Examples

```json
// List models
{"name": "list_models", "arguments": {}}

// Switch model
{
  "name": "switch_model", 
  "arguments": {"model": "llama3.2"}
}

// Learn about new features
{"name": "learn", "arguments": {}}

// Implement top improvements
{
  "name": "implement",
  "arguments": {"count": 3}
}
```

---

## 👥 Sub-Agents (Parallel Tasks)

| Tool | Description | Parameters |
|------|---|--------|
| `spawn_subagent` | Start background agent | `prompt` (required), `name`, `description` |
| `list_subagents` | List all active agents | (none) |
| `read_subagent_result` | Get result by ID | `id` |
| `summarize_subagents` | Get completion stats | (none) |

### Examples

```json
// Spawn sub-agent
{
  "name": "spawn_subagent",
  "arguments": {
    "prompt": "Add test coverage for the email processing functions",
    "name": "email-tests",
    "description": "Write unit tests for email package"
  }
}

// Check progress
{"name": "list_subagents", "arguments": {}}

// Get result
{
  "name": "read_subagent_result",
  "arguments": {"id": "email-tests-123"}
}

// Summary stats
{"name": "summarize_subagents", "arguments": {}}
```

---

## 🌐 Web Search Tool

Search DuckDuckGo with Wikipedia fallback for comprehensive results.

### Parameters

| Field | Required | Description |
|-------|----------|------|
| `query` | Yes | Search query string |
| `count` | No | Results to return (default: 5, max: 10) |

### Example

```json
{
  "name": "web_search",
  "arguments": {
    "query": "Go concurrency patterns goroutine channels",
    "count": 7
  }
}
```

### How It Works

1. Queries DuckDuckGo Instant Answer API for direct answers
2. Falls back to Wikipedia if DuckDuckGo returns no results
3. Combines both sources when available

---

## 📰 Reddit Tool

Access Reddit's public API (no authentication required).

### Actions

| Action | Description | Additional Params |
|--------|---|--|
| `search` | Search all of Reddit | `query` (required) |
| `subreddit` | List posts from r/{name} | `subreddit` (required) |
| `thread` | Get post + comments | `post_id` (required) |

### Examples

```json
// Search Reddit
{
  "name": "reddit",
  "arguments": {
    "action": "search",
    "query": "golang best practices",
    "limit": 15
  }
}

// List subreddit posts
{
  "name": "reddit",
  "arguments": {
    "action": "subreddit",
    "subreddit": "golang",
    "limit": 20
  }
}

// Get thread with comments
{
  "name": "reddit",
  "arguments": {
    "action": "thread",
    "post_id": "abc123"
  }
}
```

See [reddit-tool.md](reddit-tool.md) for detailed documentation.

---

## 📧 Google Workspace Tool (gog)

Full Google Workspace integration via OAuth CLI tool.

### Supported Services

- 📧 **Gmail**: Search, send, drafts, labels
- 📅 **Calendar**: Events CRUD, colors, multiple calendars
- 📁 **Drive**: List files, search, metadata
- 👥 **Contacts**: List and search contacts
- 📊 **Sheets**: Read/write cells and ranges
- 📝 **Docs/Slides**: Export and view content

### Quick Commands

```json
// Search Gmail for unread emails from boss in last 2 days
{
  "name": "gog",
  "arguments": {
    "command": "gmail search 'from:boss newer_than:2d' --max 10"
  }
}

// List calendar events for the week
{
  "name": "gog",
  "arguments": {
    "command": "calendar events primary --from 2026-03-10T00:00Z --to 2026-03-17T23:59Z"
  }
}

// List Drive files
{
  "name": "gog",
  "arguments": {
    "command": "drive ls --max 20"
  }
}

// Search contacts
{
  "name": "gog",
  "arguments": {
    "command": "contacts list --max 30"
  }
}
```

See [gog-tool.md](gog-tool.md) for complete reference.

---

## 📨 Email Tools

Full email system for `yolo@b-haven.org` with DKIM signing via Postfix.

### check_inbox - Read Incoming Emails

Reads Maildir at `/var/mail/b-haven.org/yolo/new/`.

```json
{
  "name": "check_inbox",
  "arguments": {
    "mark_read": true
  }
}
```

**Parameters:**
- `mark_read` (optional): If true, move processed emails from `new/` to `cur/`

### process_inbox_with_response - Full Automation

Complete workflow: read → respond → delete.

```json
{"name": "process_inbox_with_response", "arguments": {}}
```

**Process:**
1. Reads all unread emails
2. Composes intelligent auto-responses using LLM
3. Sends responses via sendmail (DKIM signed)
4. Deletes processed messages

### send_email - Send Custom Email

```json
{
  "name": "send_email",
  "arguments": {
    "to": "recipient@example.com",
    "subject": "Test Email",
    "body": "Hello from YOLO!"
  }
}
```

**Parameters:**
- `to` (optional): Recipient (default: scott@stg.net)
- `subject` (required): Email subject
- `body` (required): Email content

### send_report - Send Progress Report

Convenience wrapper for reports to scott@stg.net.

```json
{
  "name": "send_report",
  "arguments": {
    "subject": "Weekly Update",
    "body": "Completed: A, B, C\n\nNext: D, E"
  }
}
```

**Parameters:**
- `subject` (optional): Report subject (default: "YOLO Progress Report")
- `body` (required): Report content

See [EMAIL_PROCESSING.md](./EMAIL_PROCESSING.md) for architecture details.

---

## 📋 Task Management

Built-in todo system stored in `.todo.json`.

| Tool | Description | Parameters |
|------|---|--------|
| `add_todo` | Add new task | `title` (required) |
| `complete_todo` | Mark as done | `title` (required) |
| `delete_todo` | Remove entirely | `title` (required) |
| `list_todos` | View all tasks | (none) |

### Examples

```json
// Add task
{
  "name": "add_todo",
  "arguments": {
    "title": "Fix race condition in session manager"
  }
}

// Complete task
{
  "name": "complete_todo",
  "arguments": {"title": "Add unit tests for email package"}
}

// List all tasks (pending and completed)
{"name": "list_todos", "arguments": {}}
```

---

## 🌍 Web Page Reading

Read webpage content (HTML converted to plain text).

### read_webpage

```json
{
  "name": "read_webpage",
  "arguments": {
    "url": "https://example.com/documentation"
  }
}
```

**Parameters:**
- `url` (required): URL to fetch (prefixed with https:// if no scheme)

**Use Cases:**
- Read documentation pages
- Extract article content
- Check API references

---

## Tool Selection Best Practices

### When to Use Sub-Agents

✅ **Good for:**
- Independent development tasks
- Parallel code improvements
- Self-contained features or fixes

❌ **Not ideal for:**
- Tasks requiring frequent human input
- Operations with external side effects
- Very short/simple operations (just do it directly)

### Email Processing Strategy

1. **Check inbox** at startup in autonomous mode
2. **Process with response** for full automation (`process_inbox_with_response`)
3. **Send reports** daily max to scott@stg.net
4. **Avoid responding** to system logs or bounce messages

### Web Search Strategy

1. Use `web_search` for quick information and documentation
2. Use `read_webpage` to get full content from specific URLs
3. Combine both: search → find relevant URL → read page

---

## Error Handling

Most tools return structured results or errors. Check tool output before proceeding with dependent actions.

### Common Patterns

```json
// Think before complex operations
{"name": "think", "arguments": {"thought": "Plan approach for X"}}

// List models before switching
{"name": "list_models", "arguments": {}}

// Check sub-agent status before retrieving result
{"name": "list_subagents", "arguments": {}}
```

---

**Note**: Tool implementations may evolve through YOLO's self-improvement cycle. Always verify current behavior via actual execution or checking source code in `yolo/tools.go`.
