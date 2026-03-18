# YOLO User Guide

Complete usage instructions for YOLO (Your Own Living Operator).

**Quick reference**: [README.md](../README.md) | **Documentation Hub**: [DOCS/README.md](./README.md)

---

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
- [Troubleshooting](#troubleshooting)

---

## Overview

YOLO is a self-improving AI agent that can autonomously:
- Read and modify its own source code
- Use external tools (web search, Reddit, Google Workspace)
- Handle email communication
- Manage tasks with a todo list
- Run sub-agents for parallel development work

**Working Directory**: `/Users/sgriepentrog/src/yolo`  
**Source Code**: `/Users/sgriepentrog/src/yolo/yolo`  
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

Run YOLO with user input:

```bash
./yolo
```

You can provide tasks like:
- "Fix the data race in session manager"
- "Add tests for the email processor"
- "Improve the web search parser"

### Autonomous Mode

Run YOLO without human intervention:

```bash
./yolo --autonomous
```

YOLO will:
1. Check pending todos
2. Analyze code quality issues
3. Run tests and check coverage
4. Search for improvements online
5. Implement fixes autonomously
6. Commit changes to git

### Headless Autonomous Mode

For background operation:

```bash
./yolo --autonomous --headless
```

---

## Core Features

### Self-Improvement

YOLO continuously analyzes and enhances its own codebase:

- **Bug Detection**: Runs tests with race detection
- **Code Quality**: Checks formatting, vet warnings, build errors
- **Test Coverage**: Monitors coverage metrics, adds missing tests
- **Documentation**: Updates docs to match implementation

See [Autonomous Operations](./AUTONOMOUS_OPERATIONS.md) for detailed workflows.

### Email Communication

YOLO has a dedicated email address: `yolo@b-haven.org`

Features:
- Read incoming emails from Maildir inbox
- Generate and send auto-responses
- Send progress reports to stakeholders
- Archive processed emails

See [Email Processing](../EMAIL_PROCESSING.md) for detailed documentation.

### Web Integration

YOLO can access external information sources:

- **DuckDuckGo Search**: Web search with result parsing
- **Reddit API**: Fetch posts, search content, read threads
- **Google Workspace**: Gmail, Calendar, Drive, Docs, Sheets integration (via gog tool)

See [Tools Reference](./tools.md) for usage examples.

---

## Tool Reference

YOLO has access to a comprehensive set of tools:

### File Operations

| Tool | Description |
|------|-------------|
| `read_file` | Read file contents with offset/limit support |
| `write_file` | Create or overwrite files |
| `edit_file` | Replace text in files (find/replace) |
| `list_files` | List files matching glob patterns |
| `search_files` | Search file contents with regex |
| `make_dir` | Create directories recursively |
| `remove_dir` | Remove directories and contents |
| `copy_file` | Copy files between locations |
| `move_file` | Move/rename files |

### Development Tools

| Tool | Description |
|------|-------------|
| `run_command` | Execute shell commands (30s timeout) |
| `list_models` | List available Ollama models |
| `switch_model` | Change the active LLM model |
| `restart` | Rebuild and restart YOLO |

### Task Management

| Tool | Description |
|------|-------------|
| `add_todo` | Add new todo item |
| `complete_todo` | Mark todo as completed |
| `delete_todo` | Remove todo entirely |
| `list_todos` | List all todos (pending and completed) |

### Web & API Tools

| Tool | Description |
|------|-------------|
| `web_search` | DuckDuckGo web search |
| `read_webpage` | Fetch and parse webpage content |
| `reddit` | Reddit API access (search, subreddit, thread) |
| `gog` | Google Workspace CLI (Gmail, Calendar, Drive, etc.) |
| `playwright_mcp` | Browser automation (navigate, click, fill, screenshot) |

### Communication Tools

| Tool | Description |
|------|-------------|
| `send_email` | Send email via Postfix |
| `send_report` | Send progress report to scott@stg.net |
| `check_inbox` | Read emails from Maildir inbox |
| `process_inbox_with_response` | Auto-process incoming emails |

### Learning & Improvement

| Tool | Description |
|------|-------------|
| `learn` | Research improvements from web/Reddit |
| `implement` | Auto-implement discovered improvements |

### Parallel Work

| Tool | Description |
|------|-------------|
| `spawn_subagent` | Spawn background sub-agent for parallel task |
| `list_subagents` | List all active sub-agents |
| `read_subagent_result` | Get result from specific sub-agent |
| `summarize_subagents` | Get summary statistics of sub-agents |

See [Tools Reference](./tools.md) for detailed examples and parameters.

---

## Email System

YOLO operates with email at `yolo@b-haven.org`.

### Architecture

- **Inbox**: `/var/mail/b-haven.org/yolo/new/` (Maildir format)
- **Outgoing**: Postfix with automatic DKIM signing
- **Processing**: Direct LLM-generated responses

### Email Workflow

1. **Check Inbox**: Read new emails from Maildir
2. **Process Content**: Extract sender, subject, body
3. **Generate Response**: Use LLM to compose appropriate reply
4. **Send Reply**: Queue email via Postfix
5. **Archive**: Move processed emails (or delete)

### Sending Reports

Use `send_report` for progress updates:

```
Tool: send_report
Parameters:
  - subject: "Weekly Progress Report"
  - body: "Completed tasks: [...]\nPending items: [...]"
```

See [Email Processing](../EMAIL_PROCESSING.md) for security considerations and detailed architecture.

---

## Task Management

YOLO maintains a todo list in `.todo.json`:

### Adding Tasks

```bash
# Via tool call
add_todo "Fix session ID race condition"
```

### Managing Tasks

```bash
# List all tasks
list_todos

# Complete a task
complete_todo "Fix session ID race condition"

# Delete a task
delete_todo "Outdated task description"
```

### Priority Levels

Todos support priority markers:
- `CRITICAL:` - Immediate attention required
- `HIGH:` - Important, address soon
- `MEDIUM:` - Normal priority
- `LOW:` - Can be deferred

Example: `CRITICAL: Fix data race in session manager`

---

## Sub-Agents

YOLO can spawn parallel sub-agents for concurrent work:

### Spawning a Sub-Agent

```bash
spawn_subagent "Research best practices for Go error handling"
```

Parameters:
- `prompt` (required): Task description
- `name` (optional): Sub-agent identifier
- `description` (optional): Detailed description

### Managing Sub-Agents

```bash
# List all active sub-agents
list_subagents

# Get results from a specific sub-agent
read_subagent_result --id "sub-agent-id"

# Get summary statistics
summarize_subagents
```

### Best Practices

1. **Independent Tasks**: Use sub-agents for self-contained tasks
2. **Clear Prompts**: Provide specific, actionable prompts
3. **Result Collection**: Always collect results before proceeding
4. **Resource Limits**: Monitor system resources with multiple agents

See [Autonomous Operations](./AUTONOMOUS_OPERATIONS.md#sub-agent-pattern) for detailed patterns.

---

## Best Practices

### For YOLO Operators

1. **Be Specific**: Provide clear, actionable tasks
2. **Check Output**: Review YOLO's work before accepting changes
3. **Monitor Resources**: Watch CPU/memory during intensive operations
4. **Backup Important Work**: Git commits happen automatically but verify

### For Autonomous Operation

1. **Start with Small Tasks**: Let YOLO build confidence
2. **Review Periodically**: Check git history and test results
3. **Set Clear Boundaries**: Define scope in initial prompt
4. **Monitor Email**: Check for incoming messages that need attention

### Code Quality Standards

YOLO follows these standards:

- **No Data Races**: All code must pass `go test -race`
- **Formatted**: All code must pass `gofmt -l .`
- **Vet Clean**: No `go vet` warnings
- **Tests Pass**: 100% test suite must pass
- **Coverage Target**: Minimum 80% coverage goal

---

## Configuration

### Environment Variables (Optional)

```bash
export YOLO_WORKDIR=/path/to/project    # Working directory
export OLLAMA_HOST=http://localhost:11434  # Ollama server
export YOLO_MODEL=qwen3.5:27b           # LLM model to use
export GOG_CONFIG=/path/to/gog-config   # Google Workspace config
```

### Model Selection

YOLO supports multiple Ollama models:

```bash
# List available models
list_models

# Switch to a different model
switch_model llama3.2
```

### Email Configuration

- **Address**: `yolo@b-haven.org`
- **Inbox path**: `/var/mail/b-haven.org/yolo/new/`
- **Delivery**: Postfix with automatic DKIM signing
- **Rate Limiting**: Configurable in email processor

---

## Troubleshooting

### Ollama Connection Failed

```bash
# Start Ollama server
ollama serve

# Test connection
curl http://localhost:11434/api/generate -d '{"model":"qwen3.5:27b","prompt":"test"}'
```

### Build Fails

```bash
# Update dependencies
go mod download
go mod tidy

# Clean build
go clean
go build -o yolo ./cmd/yolo
```

### Tests Fail with Data Race

Check recent changes in concurrency code:
- Session manager locking
- Shared state access
- Sub-agent result handling

Run with verbose output:
```bash
go test -race -v ./...
```

### Email Processing Issues

1. Check Postfix is running: `sudo systemctl status postfix`
2. Verify Maildir permissions: `ls -la /var/mail/b-haven.org/yolo/new/`
3. Check spam filters in email content

See [Autonomous Operations](./AUTONOMOUS_OPERATIONS.md#troubleshooting-common-issues) for more solutions.

---

## Related Documentation

- [Architecture Overview](../ARCHITECTURE.md) - System design and structure
- [Autonomous Operations](./AUTONOMOUS_OPERATIONS.md) - How YOLO works autonomously
- [Tools Reference](./tools.md) - Detailed tool documentation
- [Contributing Guide](../CONTRIBUTING.md) - Development guidelines

---

**Note**: This guide is continuously updated by YOLO itself. For the latest information, always check the source documentation.
