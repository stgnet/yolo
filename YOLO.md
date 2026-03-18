# YOLO User Manual

This document provides detailed usage instructions for YOLO (Your Own Living Operator). For an overview, see [README.md](README.md). For the complete documentation index, see [DOCS/README.md](DOCS/README.md).

## Table of Contents

- [Overview](#overview)
- [Installation & Setup](#installation--setup)
- [Usage Modes](#usage-modes)
- [Core Features](#core-features)
- [Tool Reference](#tool-reference)
- [Email System](#email-system)
- [Task Management](#task-management)
- [Sub-Agents](#sub-agents)
- [Best Practices](#best-practices)
- [Configuration](#configuration)

---

## Overview

YOLO is a self-improving AI agent that can autonomously:
- Read and modify its own source code
- Use external tools (web search, Reddit, Google Workspace)
- Handle email communication
- Manage tasks with a todo list
- Run sub-agents for parallel development work

**Working Directory**: `/Users/sgriepentrog/src/yolo`  
**Your Source Code**: `/Users/sgriepentrog/src/yolo/yolo`  
**Current Model**: `qwen3.5:27b-q4_K_M`

---

## Installation & Setup

### Prerequisites

1. **Go 1.21+**: Check with `go version`
2. **Ollama**: Install and run the server
3. **Git**: Configure user.name and user.email

### Installing Ollama

```bash
# macOS
brew install ollama

# Linux
curl -fsSL https://ollama.ai/install.sh | sh

# Windows
# Download from https://ollama.ai
```

### Pulling Models

```bash
# Recommended model (balanced performance)
ollama pull qwen3.5:27b

# Alternative models
ollama pull llama3.2
ollama pull mistral
```

### Building YOLO

```bash
cd yolo
go mod download
go build -o yolo ./cmd/yolo
./yolo --version
```

---

## Usage Modes

### Interactive Mode

Run with `./yolo` to enter interactive mode:

```
user@yolo$ read_file README.md
user@yolo$ web_search "best practices go testing"
user@yolo$ add_todo "Fix test coverage"
```

### Autonomous Mode

Run with `./yolo --autonomous` for unattended operation. YOLO will:
1. Check and respond to emails
2. Analyze code quality
3. Run tests and increase coverage
4. Update documentation
5. Send progress reports

See [Autonomous Operations Guide](DOCS/AUTONOMOUS_OPERATIONS.md) for details.

---

## Core Features

### Self-Improvement

YOLO can read and modify its own source code to:
- Fix bugs automatically
- Improve test coverage
- Optimize code quality
- Add new features

### Tool Integration

See [Tools Reference](DOCS/tools.md) for the complete catalog. Key categories:

- **File Operations**: read, write, edit, copy, move files; create/remove directories
- **Web Tools**: DuckDuckGo search, Reddit API, webpage reading
- **Google Workspace**: Gmail, Calendar, Drive, Docs, Sheets, Slides (via `gog`)
- **Email System**: Check inbox, auto-respond, send reports
- **Task Management**: Todo list operations
- **System Tools**: Command execution, model switching, restart

---

## Email System

YOLO has a dedicated email address: **`yolo@b-haven.org`**

### Email Workflow

1. **Check inbox**: `check_inbox()` reads from `/var/mail/b-haven.org/yolo/new/`
2. **Auto-respond**: `process_inbox_with_response()` handles read → respond → delete
3. **Send reports**: `send_report()` sends progress updates to scott@stg.net

### Email Tools

```json
// Send custom email
{
  "name": "send_email",
  "arguments": {
    "to": "recipient@example.com",
    "subject": "Test",
    "body": "Message content"
  }
}

// Send progress report (defaults to scott@stg.net)
{
  "name": "send_report",
  "arguments": {
    "subject": "Weekly Update",
    "body": "Completed tasks: 1, 2, 3"
  }
}

// Check inbox
{
  "name": "check_inbox",
  "arguments": {
    "mark_read": true
  }
}
```

See [Email Processing](EMAIL_PROCESSING.md) for detailed documentation.

---

## Task Management

YOLO maintains a todo list in `.todo.json`:

| Tool | Description |
|------|------------|
| `add_todo(title)` | Add new task |
| `complete_todo(title)` | Mark as completed |
| `delete_todo(title)` | Remove task entirely |
| `list_todos()` | View all tasks |

Example:
```json
{
  "name": "add_todo",
  "arguments": {
    "title": "Fix race condition in session manager"
  }
}
```

---

## Sub-Agents

For parallel work, spawn sub-agents:

### Workflow

1. **Spawn**: `spawn_subagent(prompt)` starts background task
2. **Monitor**: `list_subagents()` shows progress
3. **Retrieve**: `read_subagent_result(id)` gets output
4. **Summary**: `summarize_subagents()` shows completion stats

### Example

```json
{
  "name": "spawn_subagent",
  "arguments": {
    "prompt": "Fix the race condition in tools_email.go by adding mutex protection",
    "name": "fix-email-race",
    "description": "Add synchronization to email processing"
  }
}
```

---

## Best Practices

### Code Changes Workflow

When modifying code:
1. Spawn sub-agent for the task
2. Test changes: `go build`, `go test -v ./...`
3. Check formatting: `gofmt -l .`
4. Commit to git with descriptive message
5. Use `restart` tool to reload changes

### Email Handling Best Practices

- Always check inbox at startup in autonomous mode
- Use `process_inbox_with_response()` for full automation
- Send daily progress reports via `send_report()`
- Prioritize messages from known senders (scott@stg.net)

### Testing Guidelines

```bash
# Run all tests
go test -v ./...

# Check coverage
go test -cover ./...

# Race detection
go test -race ./...

# Static analysis
go vet ./...
```

### Error Recovery

- If web search fails, it automatically falls back to Wikipedia
- If an action fails, use `think` tool to plan alternative approaches
- Sub-agents can be retried with modified prompts on failure

---

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `YOLO_WORKDIR` | Current directory | Working directory for file operations |
| `OLLAMA_HOST` | `http://localhost:11434` | Ollama server URL |
| `YOLO_MODEL` | `qwen3.5:27b` | Default model to use |

### File Paths (Internal - Do Not Modify Without Restart)

```
Working directory: /Users/sgriepentrog/src/yolo
Source code:       /Users/sgriepentrog/src/yolo/yolo
Email inbox:       /var/mail/b-haven.org/yolo/new/
Config file:       ~/.yolo_config.json (created on first run)
```

### Model Management

```json
// List available models
{"name": "list_models", "arguments": {}}

// Switch to different model
{
  "name": "switch_model", 
  "arguments": {"model": "llama3.2"}
}
```

---

## Known Limitations

1. **Test Coverage**: Target is 80%, currently at ~63% average (UI/runtime functions are hard to test)
2. **External Dependencies**: Some tests require network access or external services
3. **Email Security**: Prompt injection vulnerability being addressed
4. **Race Conditions**: Some concurrent access patterns need additional synchronization

See [TODO Items](TODO_ITEMS_2026-03-14.md) for tracking ongoing improvements.

---

## Additional Resources

- [Complete Documentation Index](DOCS/README.md)
- [Autonomous Operations Guide](DOCS/AUTONOMOUS_OPERATIONS.md)
- [Tools Reference](DOCS/tools.md)
- [Architecture Overview](ARCHITECTURE.md)
- [Security Fixes](SECURITY_FIXES_SUMMARY.md)

---

**Note**: This document is maintained by YOLO itself as part of its self-improvement cycle. Check git history for updates.
