# YOLO - Your Own Living Operator

Comprehensive documentation for the YOLO self-evolving AI agent.

## Overview

YOLO is a self-improving AI agent built in Go that can autonomously:
- Read and modify its own source code
- Use external tools (web search, Reddit, Google Workspace)
- Handle email communication
- Manage tasks with a todo list
- Run sub-agents for parallel development work

## Quick Start

### Installation & Setup

1. Ensure Ollama is installed and running
2. Install a model: `ollama pull qwen3.5:27b` (or any other model)
3. Build YOLO: `go build -o yolo ./cmd/yolo`
4. Run first-time setup: `./yolo`

### First Run Setup

On first run, YOLO will prompt for:
1. Initial task description (how you want to use it)
2. Confirmation of working directory and model

After setup, create `.yolo_config.json` in your home directory or edit it manually.

### Usage

```bash
./yolo                    # Interactive mode with custom tasks
./yolo --autonomous       # Run autonomously without user input
./yolo --model MODEL_NAME # Specify a different Ollama model
```

## Core Features

### Autonomous Self-Improvement
YOLO can read and modify its own source code to improve functionality, fix bugs, and add new features.

### Tool Integration
Built-in tools include:
- **File Operations**: read, write, edit, copy, move files; create/remove directories
- **Web Search**: DuckDuckGo with Wikipedia fallback
- **Reddit**: Search posts, browse subreddits, read threads
- **Google Workspace (gog)**: Gmail, Calendar, Drive, Docs, Sheets, Slides, Contacts, Tasks
- **Email Handling**: Check inbox, auto-respond, send reports
- **Task Management**: Add, complete, and list todos
- **Sub-agents**: Spawn parallel development tasks

### Email System
YOLO has a dedicated email address: `yolo@b-haven.org` (via Postfix with DKIM signing)

Email handling workflow:
1. Check inbox using `check_inbox()` 
2. Process emails with auto-responses via `process_inbox_with_response()`
3. Send progress reports using `send_report()`

### Task Management
Built-in todo system tracks both pending and completed tasks:
- `add_todo(title)` - Add new task
- `complete_todo(title)` - Mark as done
- `list_todos()` - View all tasks

## Architecture

### Key Files

**Working Directory**: `/Users/sgriepentrog/src/yolo`

```
yolo/
├── cmd/
│   └── yolo/main.go          # Entry point
├── yolo/
│   ├── agent.go              # Core agent logic and prompt management
│   ├── llm.go                # LLM interaction (qwen3.5:27b)
│   ├── tools.go              # All built-in tool functions
│   └── tools_test.go         # Unit tests with mocked LLM
├── go.mod                    # Dependencies
├── README.md                 # User documentation
└── YOLO.md                  # This comprehensive guide
```

### Tool System

Tools are defined in `tools.go` and exposed to the agent via JSON schema:

- **File tools**: read_file, write_file, edit_file, copy_file, move_file, make_dir, remove_dir, list_files, search_files
- **Web tools**: web_search, read_webpage, reddit (search/subreddit/thread)
- **Google tools**: gog (Gmail, Calendar, Drive, Docs, Sheets, Slides, Contacts, Tasks)
- **Email tools**: check_inbox, process_inbox_with_response, send_email, send_report
- **Task tools**: add_todo, complete_todo, list_todos
- **System tools**: run_command, think, learn, restart, list_models, switch_model
- **Sub-agent tools**: spawn_subagent, list_subagents, read_subagent_result, summarize_subagents

### Sub-Agent Pattern

For software development tasks, use this workflow:
1. Spawn sub-agent with clear task description
2. Monitor progress via `list_subagents()`
3. Retrieve results with `read_subagent_result(id)`
4. Process output and continue work

## Best Practices

### Code Changes
When modifying code:
1. Use sub-agents for development tasks
2. Test changes: `go build`, `go test -v ./...`
3. Check formatting: `gofmt -l .`
4. Commit to git
5. Restart YOLO to load changes

### Email Handling
- Always check inbox at startup
- Use `process_inbox_with_response()` for full automation (read → respond → delete)
- Send daily progress reports via `send_report()`

### Testing
Run comprehensive tests:
```bash
go test -v ./...          # Run all tests with verbose output
go test -cover ./...      # Check coverage
gofmt -l .               # Check formatting
go vet ./...             # Static analysis
```

### Error Recovery
- If web search fails, fallback to Wikipedia (automatic in `web_search`)
- If action fails, try alternative approaches
- Use `think` tool to plan before taking action

## Email Configuration

YOLO uses Postfix for email delivery with DKIM signing:
- **From**: `yolo@b-haven.org`
- **Default recipient**: `scott@stg.net` (for reports)
- **Inbox location**: `/var/mail/b-haven.org/yolo/new/`

### Sending Emails
```go
send_email(subject, body, to)        # Send custom email
send_report(body, subject)           # Send progress report to scott@stg.net
check_inbox()                        # Read incoming emails
process_inbox_with_response()        # Auto-respond and delete
```

## Known Facts & Important Information

### File Paths (DO NOT CHANGE WITHOUT RESTART)
- **Working directory**: `/Users/sgriepentrog/src/yolo`
- **Your own source code**: `/Users/sgriepentrog/src/yolo/yolo`
- **Current model**: `qwen3.5:27b`

### Email Address Pattern
YOLO has email at `yolo@b-haven.org`. Postfix handles DKIM signing automatically.

### Restart Tool
Use `restart()` to rebuild and restart the program after code changes. This is the recommended way to apply modifications.

### Sub-Agent Results Format
Results are returned as JSON with these fields:
- `task`: Task description given to sub-agent
- `output`: Final output/response from sub-agent
- `success`: Boolean indicating if task completed successfully
- `error`: Error message if applicable (only present on failure)

## Testing Guidelines

### Unit Test Structure
All unit tests use mocked LLM calls:
1. Create test file in `_test.go` pattern
2. Mock the LLM with specific prompts/responses
3. Call tool function
4. Verify results and LLM interactions

Example:
```go
func TestAddTodo(t *testing.T) {
    mockLLM(func(prompt string) (string, error) {
        if prompt == "TODO: Add item - Test task" {
            return `{"success": true, "message": "Added todo: Test task"}`, nil
        }
        return "", fmt.Errorf("unexpected prompt")
    })
    
    result := addTodo("Test task")
    assert.Contains(t, result, "Added todo")
}
```

### Test Coverage Goals
- All tool functions should have unit tests
- Aim for >50% code coverage
- Mock external dependencies (LLM, HTTP requests)
- Use table-driven tests for edge cases

## Contributing

1. Read CONTRIBUTING.md for detailed guidelines
2. Use sub-agents for code changes
3. Write comprehensive tests
4. Ensure all tests pass before committing
5. Check formatting with `gofmt`
6. Restart YOLO after changes

## History & Improvements

See IMPROVEMENTS_SUMMARY.md for a chronological log of major changes and improvements made to the YOLO system over time.

## License

[License information]
