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
| `edit_file_lines` | Replace line range in file | `path`, `start_line`, `end_line`, `content` |
| `patch_file` | Apply unified diff to file | `path`, `diff` |
| `undo_edit` | Revert last edit to a file | `path` |
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

// Undo last edit to a file (only one level of undo)
{
  "name": "undo_edit",
  "arguments": {
    "path": "config.go"
  }
}

// Replace lines 50-60 in a file (1-based, inclusive)
{
  "name": "edit_file_lines",
  "arguments": {
    "path": "main.go",
    "start_line": 50,
    "end_line": 60,
    "content": "// New implementation\nfunc Improved() {}\n"
  }
}

// Apply a unified diff patch (git-format)
{
  "name": "patch_file",
  "arguments": {
    "path": "agent.go",
    "diff": "@@ -10,7 +10,7 @@\n-const Timeout = 30\n+const Timeout = 60"
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
| `check_ollama_status` | Check Ollama server health and logs | `lines` (default: 50) |

### Examples

```json
// List models
{"name": "list_models", "arguments": {}}

// Switch model
{
  "name": "switch_model", 
  "arguments": {"model": "qwen3.5:27b-q4_K_M"}
}

// Check Ollama status (diagnose connection issues)
{
  "name": "check_ollama_status",
  "arguments": {"lines": 100}
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

## 🧠 Memory & Context Management

YOLO maintains durable facts in MEMORY.md and daily logs for session context.

| Tool | Description | Parameters |
|------|---|--------|
| `memory_read` | Read the curated MEMORY.md file | (none) |
| `memory_write` | Replace entire MEMORY.md contents | `content` (max 100 lines) |
| `memory_log` | Append observation to today's log | `entry` |
| `memory_promote` | Review daily logs for distillation | `days` (default: 7) |
| `memory_search` | Search across all memory files | `query` |

### Examples

```json
// Read persistent memory
{"name": "memory_read", "arguments": {}}

// Update memory with key facts
{
  "name": "memory_write",
  "arguments": {
    "content": "User prefers Go 1.26+, model qwen3.5:27b-q4_K_M"
  }
}

// Log an observation for today
{
  "name": "memory_log", 
  "arguments": {
    "entry": "Discovered race condition in session manager fixed on 2026-03-18"
  }
}

// Search memory history
{
  "name": "memory_search",
  "arguments": {"query": "race condition"}
}
```

---

## 📅 Scheduling & Cron

Schedule recurring tasks that fire on cron expressions.

| Tool | Description | Parameters |
|------|---|--------|
| `schedule_add` | Add a scheduled task | `name`, `cron`, `prompt` |
| `schedule_list` | List all scheduled tasks | (none) |
| `schedule_remove` | Remove a scheduled task | `id` |
| `schedule_toggle` | Enable/disable a schedule | `id`, `enabled` |

### Cron Format: `minute hour day month weekday`

```json
// Run daily at 9 AM
{
  "name": "schedule_add",
  "arguments": {
    "name": "daily-check-inbox",
    "cron": "0 9 * * *",
    "prompt": "Check and process any new emails, then send a report."
  }
}

// Run every 10 minutes
{
  "name": "schedule_add",
  "arguments": {
    "name": "periodic-health-check",
    "cron": "*/10 * * * *",
    "prompt": "Verify Ollama is running and check system health."
  }
}

// List all schedules
{"name": "schedule_list", "arguments": {}}

// Disable a schedule
{
  "name": "schedule_toggle",
  "arguments": {"id": "daily-check-inbox", "enabled": false}
}
```

---

## 🔍 Project Analysis Tools

Codebase introspection and dependency visualization.

| Tool | Description | Parameters |
|------|---|--------|
| `project_map` | Generate file tree with sizes | `pattern`, `max_depth`, `show_sizes` |
| `project_summary` | Get cached project stats | `refresh` (rescan files) |
| `dependency_graph` | Show package/module dependencies | `package` (filter directory) |
| `symbol_search` | Find function/type definitions | `query`, `kind`, `pattern` |

### Examples

```json
// Generate file tree
{
  "name": "project_map",
  "arguments": {
    "pattern": "*.go",
    "max_depth": 3,
    "show_sizes": true
  }
}

// Find all functions containing 'edit'
{
  "name": "symbol_search",
  "arguments": {
    "query": "edit",
    "kind": "func"
  }
}

// Get dependency graph for concurrency package
{
  "name": "dependency_graph",
  "arguments": {"package": "concurrency"}
}
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

Full email system for `yolo@example.com` with DKIM signing via Postfix.

### check_inbox - Read Incoming Emails

Reads Maildir at `/var/mail/example.com/yolo/new/`.

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

## 🔧 Git Version Control

Full git repository management.

| Tool | Description | Parameters |
|------|---|--------|
| `git_status` | Show current repository status | (none) |
| `git_diff` | Show changes (staged/unstaged) | `file` (optional) |
| `git_log` | Recent commit history | `limit` (default: 10) |
| `git_branch` | List all branches | (none) |
| `git_checkout` | Switch branch or restore file | `branch`, `file` |
| `git_add` | Stage files for commit | `file` (default: all) |
| `git_commit` | Commit staged changes | `message`, `all` |
| `git_show` | Show commit details | `commit` (default: HEAD) |
| `git_remote` | Show configured remotes | (none) |

### Examples

```json
// Check status
{"name": "git_status", "arguments": {}}

// Stage all changes
{"name": "git_add", "arguments": {}}

// Commit with message
{
  "name": "git_commit",
  "arguments": {"message": "Fix documentation for memory tools"}
}

// Auto-stage and commit
{
  "name": "git_commit",
  "arguments": {
    "all": true,
    "message": "Update README with current model"
  }
}

// View recent commits
{"name": "git_log", "arguments": {"limit": 20}}

// Switch branches
{
  "name": "git_checkout",
  "arguments": {"branch": "feature/new-tool"}
}

// Restore a file from HEAD
{
  "name": "git_checkout",
  "arguments": {"file": "tools.go"}
}
```

---

## 📜 History Search

Search conversation history beyond the current context window.

### history_search

Search all past conversations by keyword. Returns matching messages with timestamps and context.

```json
{
  "name": "history_search",
  "arguments": {
    "query": "race condition fix",
    "limit": 10
  }
}
```

**Parameters:**
- `query` (required): Search terms (words ANDed; use quotes for exact phrases)
- `limit` (optional): Max results (default: 20)

**Use Cases:**
- Recall past decisions and implementations
- Find previous bug fixes
- Locate technical discussions

---

## 📸 Browser Automation & Screenshots

Web page interaction and screenshot capture.

| Tool | Description | Parameters |
|------|---|--------|
| `playwright_mcp` | Navigate, interact with DOM, extract content | `action`, `selector`, `url`, etc. |
| `screenshot` | Capture web page screenshot | `url`, `path`, `full_page`, `width`, `height` |

### playwright_mcp Actions

- `navigate`: Load a URL
- `click`: Click an element
- `fill`: Fill input field
- `getHTML`: Extract HTML from selector
- `screenshot`: Take screenshot

```json
// Navigate to a page
{
  "name": "playwright_mcp",
  "arguments": {
    "action": "navigate",
    "url": "https://example.com",
    "waitUntil": "networkidle"
  }
}

// Fill a search form and submit
{
  "name": "playwright_mcp",
  "arguments": {
    "action": "fill",
    "selector": "#search-input",
    "value": "Go programming concurrency"
  }
}
```

### screenshot Tool

```json
// Capture full page screenshot
{
  "name": "screenshot",
  "arguments": {
    "url": "https://example.com/documentation",
    "full_page": true,
    "path": "./screenshots/doc-page.png"
  }
}
```

**Parameters:**
- `url` (required): Web page to capture
- `path` (optional): Output file path (default: temp directory)
- `full_page` (optional): Capture entire scrollable page
- `width`, `height` (optional): Viewport size (default: 1280x720)

---

## 🛠️ Configuration Management

Runtime configuration for YOLO behavior.

| Tool | Description | Parameters |
|------|---|--------|
| `get_config` | Get all or specific config value | `key` (optional) |
| `set_config` | Update a configuration value | `key`, `value` |

### Examples

```json
// Get all configuration
{"name": "get_config", "arguments": {}}

// Get specific setting
{
  "name": "get_config",
  "arguments": {"key": "model"}
}

// Enable autonomous mode
{
  "name": "set_config",
  "arguments": {
    "key": "auto_mode",
    "value": "true"
  }
}

// Set custom model
{
  "name": "set_config",
  "arguments": {
    "key": "model", 
    "value": "qwen3.5:27b-q4_K_M"
  }
}
```

**Available Keys:** `model`, `auto_mode`, `debug_mode`, `terminal_mode`, `think_mode`, `tts_enabled`, `email_from`, `email_to`, `inbox_path`, and more.

---

## 🔋 Victron Device Integration (Bluetooth LE)

Connect to and read values from Victron SmartSolar, SmartShunt, and other BLE-enabled Victron devices.

### victron

Interact with Victron energy monitoring and charging devices via Bluetooth Low Energy.

**Actions:**
- `scan`: Search for nearby Victron BLE devices
- `connect`: Establish connection to a device
- `disconnect`: Close connection to device(s)
- `get_values`: Read current sensor values from connected device
- `subscribe`: Start real-time value monitoring
- `device_info`: Get device details and capabilities

```json
// Scan for Victron devices (default: 10 seconds)
{
  "name": "victron",
  "arguments": {
    "action": "scan",
    "duration": "30"  // optional, max 60 seconds
  }
}

// Connect to a specific device
{
  "name": "victron",
  "arguments": {
    "action": "connect",
    "address": "XX:XX:XX:XX:XX:XX"  // MAC address from scan results
  }
}

// Get current values from connected device
{
  "name": "victron",
  "arguments": {
    "action": "get_values",
    "address": "XX:XX:XX:XX:XX:XX",
    "timeout": "10"  // optional, seconds to collect values
  }
}

// Start real-time monitoring
{
  "name": "victron",
  "arguments": {
    "action": "subscribe",
    "address": "XX:XX:XX:XX:XX:XX"
  }
}

// Get device information
{
  "name": "victron",
  "arguments": {
    "action": "device_info",
    "address": "XX:XX:XX:XX:XX:XX"
  }
}

// Disconnect from device (or all devices if no address)
{
  "name": "victron",
  "arguments": {
    "action": "disconnect",
    "address": "XX:XX:XX:XX:XX:XX"  // optional
  }
}
```

**Supported Devices:**
- SmartSolar MPPT charge controllers
- SmartShunt battery monitors  
- VE.Direct Bluetooth Smart adapters
- Any Victron device with BLE/GATT support

**Available Values (depends on device type):**
- Voltage (`V`): Battery voltage in volts
- Current (`A`): Battery current in amps
- Power (`W`): Battery power in watts
- State of Charge (`SoC`): Battery remaining capacity %
- Temperature (`T`): Device or battery temperature
- State (`state`): System operating state (e.g., "Bulk", "Absorption", "Float")

**Parameters:**
- `action` (required): One of `scan`, `connect`, `disconnect`, `get_values`, `subscribe`, `device_info`
- `address` (required for most actions): Device MAC address from scan results
- `duration` (optional, scan only): Scan duration in seconds (default: 10, max: 60)
- `timeout` (optional, get_values/subscribe): How long to collect values

**Use Cases:**
- Monitor solar charging system status
- Track battery state of charge and health
- Log energy consumption data
- Integrate Victron devices into home automation systems

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

## 🔋 Victron Energy BLE Devices

Interact with Victron Energy solar and battery monitoring equipment via Bluetooth Low Energy. Supports SmartSolar MPPT charge controllers, SmartShunt battery monitors, and VE.Direct adapters.

### Overview

| Parameter | Required | Description |
|-----------|----------|-------------|
| `action` | Yes | Operation to perform: `scan`, `connect`, `disconnect`, `get_values`, `subscribe`, `device_info` |
| `address` | Conditional | Device MAC address (required for connect/disconnect/get_values/subscribe/device_info) |
| `duration` | No | Scan duration in seconds (default: 10, max: 60) |
| `timeout` | No | Operation timeout in seconds (default varies by action) |

### Actions

| Action | Description | Parameters | Returns |
|--------|-------------|------------|---------|
| `scan` | Discover nearby Victron devices | `duration` | List of devices with addresses, names, RSSI |
| `connect` | Connect to a device | `address`, `timeout` | Device info and connection status |
| `disconnect` | Disconnect from device(s) | `address` (optional) | Disconnection confirmation |
| `get_values` | Read current sensor values | `address`, `timeout` | Array of sensor readings |
| `subscribe` | Monitor real-time updates | `address`, `timeout` | Stream of value changes |
| `device_info` | Get device details | `address` | Device type, name, capabilities |

### Common Sensor Keys

#### Voltage
- `V` - System voltage
- `B.V` - Battery voltage
- `P.V` - PV input voltage (panel 1)
- `P.V(2)` - PV input voltage (panel 2)

#### Current
- `A` - System current (charging)
- `B.A` - Battery current
- `P.A` - PV input current (panel 1)
- `P.A(2)` - PV input current (panel 2)

#### Power
- `W` - System power (charging)
- `P.W` - PV input power (panel 1)
- `P.W(2)` - PV input power (panel 2)

#### Battery State
- `SoC` - State of charge percentage
- `V.Ar` - Energy yield today (Ah)
- `V.Wh.ar` - Energy yield today (Wh)

#### Other
- `T` - Device temperature
- `Alg.S` - Charging algorithm state
- `S` - Stage
- `E` - Error code

### Examples

```json
// Scan for Victron devices nearby
{
  "name": "victron",
  "arguments": {
    "action": "scan",
    "duration": "15"
  }
}

// Connect to a specific device
{
  "name": "victron",
  "arguments": {
    "action": "connect",
    "address": "CC:50:C3:29:7D:B5",
    "timeout": "10"
  }
}

// Get current sensor values
{
  "name": "victron",
  "arguments": {
    "action": "get_values",
    "address": "CC:50:C3:29:7D:B5",
    "timeout": "5"
  }
}

// Monitor device for 30 seconds
{
  "name": "victron",
  "arguments": {
    "action": "subscribe",
    "address": "CC:50:C3:29:7D:B5",
    "timeout": "30"
  }
}

// Get device information
{
  "name": "victron",
  "arguments": {
    "action": "device_info",
    "address": "CC:50:C3:29:7D:B5"
  }
}

// Disconnect from device
{
  "name": "victron",
  "arguments": {
    "action": "disconnect",
    "address": "CC:50:C3:29:7D:B5"
  }
}

// Disconnect from all devices
{
  "name": "victron",
  "arguments": {
    "action": "disconnect"
  }
}
```

### Workflow Example: Solar Monitoring

```json
// Step 1: Scan for devices
{"name": "victron", "arguments": {"action": "scan", "duration": "10"}}

// Step 2: Connect to SmartSolar (from scan results)
{
  "name": "victron",
  "arguments": {
    "action": "connect",
    "address": "CC:50:C3:29:7D:B5"
  }
}

// Step 3: Get current solar power and battery status
{
  "name": "victron",
  "arguments": {
    "action": "get_values",
    "address": "CC:50:C3:29:7D:B5",
    "timeout": "5"
  }
}

// Step 4: Monitor real-time for 1 minute to see charging patterns
{
  "name": "victron",
  "arguments": {
    "action": "subscribe",
    "address": "CC:50:C3:29:7D:B5",
    "timeout": "60"
  }
}

// Step 5: Disconnect when done
{
  "name": "victron",
  "arguments": {
    "action": "disconnect",
    "address": "CC:50:C3:29:7D:B5"
  }
}
```

### Typical Scan Response

```json
{
  "status": "success",
  "message": "Found 2 device(s)",
  "devices": [
    {
      "address": "CC:50:C3:29:7D:B5",
      "name": "SmartSolar MPPT 100/30",
      "rssi": -45,
      "is_victron": true
    },
    {
      "address": "A4:C3:F0:12:34:56",
      "name": "SmartShunt",
      "rssi": -62,
      "is_victron": true
    }
  ]
}
```

### Typical Values Response

```json
{
  "status": "success",
  "message": "Received 8 value(s)",
  "values": [
    {
      "key": "V",
      "raw_value": "13.85",
      "float_value": 13.85,
      "timestamp": "2026-03-20T14:30:25Z"
    },
    {
      "key": "A",
      "raw_value": "5.2",
      "float_value": 5.2,
      "timestamp": "2026-03-20T14:30:25Z"
    },
    {
      "key": "W",
      "raw_value": "72.0",
      "float_value": 72.0,
      "timestamp": "2026-03-20T14:30:25Z"
    },
    {
      "key": "SoC",
      "raw_value": "85",
      "float_value": 85.0,
      "timestamp": "2026-03-20T14:30:26Z"
    }
  ]
}
```

### Platform Support

| Platform | Status | Notes |
|----------|--------|-------|
| Linux | ✅ Partial | Scanning works, GATT reading in progress (uses BlueZ via D-Bus) |
| macOS | ❌ Mock only | Requires CoreBluetooth cgo bindings |
| Windows | ❌ Mock only | Requires WinRT Bluetooth bindings |

### Hardware Requirements (Linux)

- Bluetooth adapter supporting BLE 4.0+
- BlueZ 5.43 or higher
- Proper D-Bus permissions (may need root or udev rules)
- Victron device with Bluetooth Smart/VE.Direct support

### Use Cases

✅ **Solar monitoring**: Track PV production, battery charging status  
✅ **Battery health**: Monitor SOC, voltage, current for lithium/flooded batteries  
✅ **System diagnostics**: Detect errors, temperature alerts, abnormal readings  
✅ **Energy tracking**: Log daily energy yield and consumption patterns  

See [DOCS/VICTRON_BLE_LIBRARIES.md](VICTRON_BLE_LIBRARIES.md) for implementation details.

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
