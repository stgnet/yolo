# API Reference

This document provides a comprehensive reference for the YOLO Agent's public API, tools, and capabilities.

## Overview

YOLO Agent is a Go-based autonomous agent framework that provides MCP-style tool functions for performing various operations including file management, system tasks, web interactions, and more.

## Core Tools

### File Operations

#### `read_file(path, offset?, limit?)`
Reads the contents of a file with optional line range specification.

```go
// Read entire file
content := read_file("example.txt")

// Read specific lines (1-based)
content := read_file("example.txt", 1, 50) // First 50 lines
content := read_file("example.txt", 100, 200) // Lines 100-300
```

**Parameters:**
- `path` (required): Relative path to file
- `offset` (optional, default: 1): Starting line number (1-based)
- `limit` (optional, default: 200): Maximum lines to read

#### `write_file(path, content)`
Creates or overwrites a file with the provided content.

```go
write_file("output.txt", "Hello, World!")
```

**Parameters:**
- `path` (required): Relative path for new/updated file
- `content` (required): File contents as string

#### `edit_file(path, old_text, new_text, options?)`
Replaces text in a file with support for multiple occurrence modes.

```go
// Replace first occurrence
edit_file("code.go", "func OldName()", "func NewName()")

// Replace all occurrences
edit_file("code.go", "var x", "var y", replace_all=true)

// Replace Nth occurrence
edit_file("code.go", "log.Println", "log.Printf", occurrence=2)

// Preview changes without applying
edit_file("code.go", "old", "new", dry_run=true)
```

**Parameters:**
- `path` (required): Relative path to file
- `old_text` (required): Text pattern to find
- `new_text` (required): Replacement text
- `dry_run` (optional, default: false): Preview without modifying
- `occurrence` (optional, default: first): Which occurrence to replace (0=first, -1=last, N=Nth)
- `replace_all` (optional, default: false): Replace all occurrences

#### `edit_file_lines(path, start_line, end_line, content)`
Replaces a range of lines with new content.

```go
// Insert at line 10
edit_file_lines("doc.md", 10, 9, "New heading\n")

// Replace lines 5-10
edit_file_lines("code.go", 5, 10, "// New implementation")

// Delete lines 20-30
edit_file_lines("data.txt", 20, 30, "")
```

**Parameters:**
- `path` (required): Relative path to file
- `start_line` (required): First line to replace (1-based)
- `end_line` (required): Last line to replace (inclusive)
- `content` (required): New content (empty string to delete lines)

#### `patch_file(path, diff)`
Applies a unified diff (git-style) to a file.

```go
diff := `--- a/file.go
+++ b/file.go
@@ -1,3 +1,4 @@
 package main
 
+import "fmt"
 func main() {
`
patch_file("file.go", diff)
```

#### `copy_file(source, dest)`
Copies a file from source to destination.

```go
copy_file("source.txt", "backup/source_copy.txt")
```

#### `move_file(source, dest)`
Moves/renames a file from source to destination.

```go
move_file("old_name.txt", "new_name.txt")
```

### Directory Operations

#### `make_dir(path)`
Creates a directory recursively.

```go
make_dir("project/src/utils/helpers")
```

#### `remove_dir(path)`
Removes a directory and all contents recursively.

⚠️ **Warning**: This operation is destructive and cannot be undone.

```go
remove_dir("temp_build_output")
```

### File Discovery

#### `list_files(pattern?)`
Lists files matching a glob pattern.

```go
// List all files in current directory
list_files()

// List all Go files recursively
list_files("**/*.go")

// List Markdown files in docs folder
list_files("docs/*.md")
```

#### `search_files(query, pattern?)`
Searches file contents using regex.

```go
// Search for function definitions
search_files("func.*Test", "**/*.go")

// Find all TODO comments
search_files("TODO:.*", "**/*")
```

### Git Operations

#### `git_status()`
Shows current git status including modified, added, and untracked files.

#### `git_diff(file?)`
Shows diff of changes. Optionally specify a file.

```go
// All changes
git_diff()

// Specific file
git_diff("agent.go")
```

#### `git_log(limit?)`
Shows recent commit history.

```go
git_log(10) // Last 10 commits
```

#### `git_add(file?)`
Stages files for commit. If no file specified, stages all changes.

#### `git_commit(message, all?)`
Commits staged changes with a message.

```go
git_commit("feat: add new functionality", all=true)
```

#### `git_checkout(branch?, file?)`
Checks out a branch or restores a file from HEAD.

```go
// Switch branches
git_checkout("feature/new-tool")

// Restore a file
git_checkout(file="accidentally_modified.go")
```

### Project Analysis

#### `project_map(pattern?, max_depth?, show_sizes?)`
Generates hierarchical file tree with metadata.

```go
// Full project tree
project_map()

// Only Go files, depth 3
project_map("*.go", 3, true)
```

#### `dependency_graph(package?)`
Parses imports to show package dependencies.

#### `symbol_search(query, kind?, pattern?)`
Searches for function, type, class, and variable definitions.

```go
// Find all functions matching "parse"
symbol_search("parse", "func")

// Find types
symbol_search("Result", "type")
```

### Web Operations

#### `web_search(query, count?, engine?)`
Performs web search via SearXNG or DuckDuckGo.

```go
results := web_search("Go concurrency patterns", 5)
```

#### `read_webpage(url, mode?)`
Fetches a webpage and extracts readable content.

```go
// Smart extraction (default)
content := read_webpage("https://example.com/article")

// Full page text
content := read_webpage("https://example.com", mode="full")
```

#### `screenshot(url, options?)`
Captures a screenshot of a web page.

```go
screenshot("https://example.com", full_page=true, width=1920, height=1080)
```

### Memory & Context

#### `memory_read()`
Reads the curated MEMORY.md file with durable facts and preferences.

#### `memory_write(content)`
Replaces the entire MEMORY.md (max 100 lines).

```go
memory_write("User prefers Go formatting with go fmt\nKey project decision: Use SQLite for history")
```

#### `memory_log(entry)`
Appends an observation to today's daily context log.

```go
memory_log("Discovered that tool X works better than Y for task Z")
```

#### `memory_search(query)`
Searches across all memory files (MEMORY.md and daily logs).

#### `memory_promote(days?)`
Retrieves daily logs for review and distillation into MEMORY.md.

### History Search

#### `history_search(query, limit?)`
Searches conversation history by keyword.

```go
// Search past discussions
results := history_search("testing strategy", 20)
```

### Todo Management

#### `add_todo(title)`
Adds a new todo item.

```go
add_todo("Review pull request #42")
```

#### `complete_todo(title)`
Marks a todo as completed.

```go
complete_todo("Review pull request #42")
```

#### `delete_todo(title)`
Permanently removes a todo item.

```go
delete_todo("Outdated task that's no longer relevant")
```

#### `list_todos()`
Lists all todos (pending and completed).

### Email & Communication

#### `send_email(subject, body, to?, attachments?)`
Sends an email via sendmail.

```go
send_email(
    subject="Weekly Report",
    body="Here's the weekly status...",
    to="team@example.com",
    attachments=["report.pdf"]
)
```

#### `check_inbox(mark_read?)`
Reads emails from configured Maildir inbox.

```go
emails := check_inbox(mark_read=true)
```

#### `process_inbox_with_response()`
Automatically processes incoming emails with auto-responses.

### Subagents

#### `spawn_subagent(prompt, name?, description?)`
Spawns a background sub-agent for parallel task execution.

```go
subagent := spawn_subagent(
    prompt="Analyze the codebase and identify optimization opportunities",
    name="CodeAnalyzer",
    description="Background analysis of code quality"
)
```

#### `list_subagents()`
Lists all active sub-agents with status.

#### `read_subagent_result(id)`
Reads results from a completed sub-agent.

### Configuration

#### `get_config(key?)`
Gets current configuration values.

```go
// All config
config := get_config()

// Specific key
model := get_config("model")
```

#### `set_config(key, value)`
Sets a configuration value.

```go
set_config("model", "llama3.2")
set_config("tts_enabled", "true")
```

### Ollama Integration

#### `list_models()`
Lists available Ollama models.

#### `switch_model(model)`
Switches to a different Ollama model.

```go
switch_model("llama3.1")
```

#### `check_ollama_status(lines?)`
Checks Ollama server status and logs.

### System Operations

#### `run_command(command)`
Executes a shell command (timeout: 30s).

```go
output := run_command("ls -la /tmp")
```

#### `restart()`
Rebuilds and restarts the program with new binary.

```go
restart() // Applies all code changes
```

### Scheduling

#### `schedule_add(name, cron, prompt)`
Adds a scheduled task that fires on a cron schedule.

```go
schedule_add(
    name="Daily backup",
    cron="0 2 * * *", // 2 AM daily
    prompt="Run database backup and verify integrity"
)
```

#### `schedule_list()`
Lists all scheduled tasks with status and next run time.

#### `schedule_toggle(id, enabled)`
Enables or disables a scheduled task.

#### `schedule_remove(id)`
Removes a scheduled task.

### Playwright Browser Automation

#### `playwright_mcp(action, url?, selector?, value?, timeout?, path?, waitUntil?)`
Browser automation with multiple action types.

```go
// Navigate to URL
playwright_mcp("navigate", url="https://example.com")

// Fill form field
playwright_mcp("fill", selector="#email", value="test@example.com")

// Click button
playwright_mcp("click", selector="button.submit")

// Get HTML content
playwright_mcp("getHTML", selector=".content")

// Take screenshot
playwright_mcp("screenshot", path="output/screenshot.png")
```

### External APIs

#### `reddit(action, query?, subreddit?, post_id?, limit?)`
Fetches posts from Reddit (public API, no auth required).

```go
// Search Reddit
reddit("search", query="Go programming", limit=10)

// List subreddit posts
reddit("subreddit", subreddit="golang", limit=25)

// Get thread with comments
reddit("thread", post_id="abc123")
```

#### `victron(action, address?, duration?, timeout?)`
Connects to Victron Energy devices via BLE.

```go
// Scan for devices
victron("scan", duration=15)

// Connect to device
victron("connect", address="AA:BB:CC:DD:EE:FF")

// Read values
values := victron("get_values", address="AA:BB:CC:DD:EE:FF")
```

#### `gog(command)`
Google CLI tool for Gmail, Calendar, Drive, etc.

```go
// Search unread emails
gog("gmail search newer_than:1d --max 5")

// List calendar events
gog("calendar list events")

// List drive files
gog("drive list")
```

## Error Handling

All tools return structured results with error information when operations fail. Errors include:

- **File not found**: When reading non-existent files
- **Permission denied**: When lacking file/directory permissions  
- **Invalid input**: When parameters don't meet requirements
- **Network errors**: For web/email/external API failures
- **Timeout**: Operations exceeding time limits

## Best Practices

1. **Always validate inputs** before calling tools
2. **Use dry_run mode** for edit_file to preview changes
3. **Check file existence** before reading/copying
4. **Handle errors gracefully** in automation scripts
5. **Use temp directories** for temporary file operations
6. **Clean up resources** after test operations

## Security Considerations

- All file operations use **relative paths only** (no absolute paths)
- Commands execute with **30-second timeout** to prevent hangs
- Email/password credentials managed through **configured system accounts**
- Web scraping respects **robots.txt** and rate limits where applicable
- Git operations work on **local repository only**

## Testing Guidelines

See [`TESTING.md`](TESTING.md) for comprehensive testing practices and safety guidelines.

All tools are designed to be:
- ✅ **Side-effect free** in test environments
- ✅ **Deterministic** with consistent outputs
- ✅ **Idempotent** where applicable
- ✅ **Safe for CI/CD** integration

## Extending the API

To add new tools:

1. Create a function following the existing tool signature pattern
2. Add comprehensive tests (see `TESTING.md`)
3. Document parameters and behavior in this file
4. Ensure isolation from production systems
5. Update dependency graph if adding imports

For examples, examine existing tool implementations in `tools_*.go` files.

## Version Compatibility

This API is versioned alongside the main project. Check `go.mod` for current version:

```bash
go list -m
```

Breaking changes will be documented in [`CHANGELOG.md`](CHANGELOG.md).
