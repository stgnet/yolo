# YOLO - Your Own Living Operator

**Version**: 1.0 | **Status**: ✅ Production Ready | **Last Updated**: 2026-03-18

[![Go](https://img.shields.io/badge/Go-1.21+-blue.svg)](https://golang.org)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## Overview

YOLO is a **self-evolving AI agent** for autonomous software development. It operates independently to improve code quality, fix bugs, add tests, and implement new features by reading and modifying its own source code.

### What Makes YOLO Special?

- 🤖 **Self-improving**: Continuously analyzes and enhances its own codebase
- 📧 **Email-enabled**: Full email processing with auto-responses at `yolo@b-haven.org`
- 🌐 **Web-connected**: DuckDuckGo search, Reddit API, Google Workspace integration
- ⚡ **Autonomous mode**: Works independently without human intervention
- 🔧 **Developer tools**: File operations, command execution, sub-agent spawning

## Quick Start

### Prerequisites

```bash
# Go 1.21+ required
go version

# Install Ollama
brew install ollama  # macOS
curl -fsSL https://ollama.ai/install.sh | sh  # Linux

# Pull a model (qwen3.5:27b recommended)
ollama pull qwen3.5:27b
```

### Installation

```bash
git clone https://github.com/your-username/yolo.git
cd yolo
go mod download
go build -o yolo
./yolo --version
```

### Running YOLO

**Interactive Mode:**
```bash
./yolo
```

**Autonomous Mode:**
```bash
./yolo --autonomous
```

## Architecture

YOLO consists of several key components that work together:

```
┌───────────────────────────────────────────┐
│              Terminal (raw mode)          │
│  ┌───────────────────────┬───────────────┤
│  │   Output (scrolling)  │   Input area  │
│  │   TerminalUI          │   InputManager│
│  └──────────┬────────────┴───────┬───────┘
└─────────────┼────────────────────┼────────┘
              │                    │
              ▼                    ▼
      ┌─────────────────────────────────────┐
      │           YoloAgent                 │
      │  ┌────────────┬────────────────────┤
      │  │ History    │ ToolExecutor       │
      │  │ Manager    │  (21 tools)        │
      │  └────────────┴────────────────────┘
      └───────────┬──────────────┬──────────┘
                  │              │
                  ▼              ▼
      ┌──────────────────┐  ┌────────────────┐
      │ .yolo/          │  │ OllamaClient   │
      │ history.json    │  │ /api/chat      │
      │ subagents/      │  │ /api/tags      │
      └──────────────────┘  └────────────────┘
```

### Components

- **YoloAgent** (`agent.go`): Central orchestrator handling chat loops and commands
- **OllamaClient** (`ollama.go`): HTTP client for Ollama REST API with streaming support
- **ToolExecutor** (`tools.go`): Dispatches tool calls to 21 concrete implementations
- **HistoryManager** (`history.go`): Thread-safe persistence in `.yolo/history.json`
- **InputManager** (`input.go`): Raw terminal input handling in separate goroutine
- **TerminalUI** (`terminal.go`): Split-screen layout with scrollable output

## Tools Reference

YOLO has 21 built-in tools that the LLM can call:

### File Operations
| Tool | Description |
|------|-------------|
| `read_file` | Read file contents with optional offset/limit |
| `write_file` | Create or overwrite a file |
| `edit_file` | First-occurrence string replacement |
| `list_files` | Glob matching, including recursive `**/` patterns |
| `search_files` | Regex search across file contents |
| `make_dir` | Create directories recursively |
| `remove_dir` | Remove a directory tree |
| `copy_file` | Copy a file, creating destination directories |
| `move_file` | Move/rename, with cross-filesystem fallback |

### Agent Management
| Tool | Description |
|------|-------------|
| `spawn_subagent` | Create a background sub-agent for parallel work |
| `list_subagents` | List sub-agent statuses and progress |
| `read_subagent_result` | Retrieve a sub-agent's output |
| `summarize_subagents` | Aggregate sub-agent statistics |
| `think` | Record reasoning without side effects |
| `restart` | Rebuild from source and exec the new binary |

### External Services
| Tool | Description |
|------|-------------|
| `web_search` | DuckDuckGo search with Wikipedia fallback (5-min cache) |
| `reddit` | Reddit API: search, subreddit posts, thread details |
| `gog` | Google Workspace: Gmail, Calendar, Drive, Docs, Sheets |
| `send_email` | Send email via postfix from yolo@b-haven.org |
| `check_inbox` | Read Maildir inbox at /var/mail/b-haven.org/yolo/new/ |
| `process_inbox_with_response` | Auto-respond to emails then delete |
| `list_models` | List available Ollama models |
| `switch_model` | Change the active model |

### Task Management
| Tool | Description |
|------|-------------|
| `add_todo` | Add item to todo list |
| `complete_todo` | Mark todo as completed |
| `delete_todo` | Remove todo entirely |
| `list_todos` | List all todos (pending and completed) |

### System Commands
| Tool | Description |
|------|-------------|
| `run_command` | Execute shell command (30s timeout, stdin=/dev/null) |

## Email Processing

YOLO provides intelligent email processing for `yolo@b-haven.org`:

- **Read inbox**: Check new emails from Maildir
- **Auto-responses**: LLM-generated natural responses to questions/requests
- **Auto-deletion**: Delete emails after successful response
- **Progress reports**: Scheduled status updates to scott@stg.net

See [EMAIL_PROCESSING.md](EMAIL_PROCESSING.md) for detailed email system documentation.

## Configuration

### Environment Variables (Optional)

```bash
export OLLAMA_HOST=http://localhost:11434  # Ollama server URL
export YOLO_MODEL=qwen3.5:27b              # Default model to use
export YOLO_NUM_CTX=8192                   # Override context window size
```

### Runtime Configuration

YOLO stores state in `.yolo/` directory (gitignored):
- `history.json`: Conversation history
- `subagents/`: Background agent results
- `todos.json`: Task list

## Development

### Running Tests

```bash
# All tests
go test -v ./...

# With coverage
go test -cover ./...

# Race detection
go test -race ./...
```

### Code Quality Checks

```bash
gofmt -l .    # Check formatting
go vet ./...  # Static analysis
go build      # Verify build
```

### Adding New Tools

1. Create tool function in `tools.go` or `tools_xxx.go`
2. Add to `ToolDefinitions` slice in `agent.go`
3. Write unit tests for all code paths
4. Update this README with tool documentation

Example:
```go
func newToolName(args string) string {
    // Parse arguments
    var input struct {
        Query string `json:"query"`
    }
    if err := json.Unmarshal([]byte(args), &input); err != nil {
        return "Error parsing args: " + err.Error()
    }
    // Execute logic
    result := doWork(input.Query)
    return fmt.Sprintf("Result: %s", result)
}
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for detailed development guidelines.

## Troubleshooting

**Ollama connection failed:**
```bash
ollama serve
curl http://localhost:11434/api/generate -d '{"model":"qwen3.5:27b","prompt":"test"}'
```

**Build fails:**
```bash
go mod download
go mod tidy
```

**Tests fail with race conditions:**
```bash
go test -race ./...  # Run with race detector
# Check for unprotected global variable access
```

## Architecture Deep Dive

For detailed architecture documentation, see [ARCHITECTURE.md](ARCHITECTURE.md) covering:

- Data flow diagrams (user chat, autonomous thinking, sub-agents)
- Component specifications
- Concurrency patterns and thread safety
- Design principles and safety mechanisms

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines on:

- Development workflow
- Code style requirements  
- Testing standards (>90% coverage goal)
- Submission process

## License

MIT License - see LICENSE file for details.

---

**Note**: YOLO continuously improves its own code and documentation. Changes are automatically committed to git as part of the self-improvement cycle. For the system prompt template, see [SYSTEM_PROMPT.md](SYSTEM_PROMPT.md).
