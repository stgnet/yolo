# YOLO - Your Own Living Operator

**Version**: 1.0 | **Status**: ✅ Production Ready | **Last Updated**: 2026-03-18

[![Go](https://img.shields.io/badge/Go-1.26+-blue.svg)](https://golang.org)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## Overview

YOLO is a **self-evolving AI agent** for autonomous software development. It operates independently to improve code quality, fix bugs, add tests, and implement new features by reading and modifying its own source code.

### What Makes YOLO Special?

- 🤖 **Self-improving**: Continuously analyzes and enhances its own codebase
- 📧 **Email-enabled**: Full email processing with auto-responses at `yolo@example.com`
- 🌐 **Web-connected**: DuckDuckGo search, Reddit API, Google Workspace integration
- ⚡ **Autonomous mode**: Works independently without human intervention
- 🔧 **Developer tools**: 63 built-in tools for file operations, version control, command execution, browser automation, and hardware integration

## Quick Start

### Prerequisites

```bash
# Go 1.26+ required
go version

# Install Ollama
brew install ollama  # macOS
curl -fsSL https://ollama.ai/install.sh | sh  # Linux

# Pull a model (qwen3.5:27b-q4_K_M recommended)
ollama pull qwen3.5:27b-q4_K_M
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
        │  │ Manager    │  (40 tools)        │
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
- **ToolExecutor** (`tools.go`): Dispatches tool calls to 40 concrete implementations
- **HistoryManager** (`history.go`): Thread-safe persistence in `.yolo/history.json`
- **InputManager** (`input.go`): Raw terminal input handling in separate goroutine
- **TerminalUI** (`terminal.go`): Split-screen layout with scrollable output

## Tools Reference

YOLO has 63 built-in tools that the LLM can call:

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
| `check_ollama_status` | Check Ollama server status and read debug logs |

### External Services
| Tool | Description |
|------|-------------|
| `web_search` | DuckDuckGo search with Wikipedia fallback (5-min cache) |
| `read_webpage` | Fetch webpage content, convert HTML to text |
| `reddit` | Reddit API: search, subreddit posts, thread details |
| `gog` | Google Workspace: Gmail, Calendar, Drive, Docs, Sheets |
| `send_email` | Send email via postfix from yolo@example.com |
| `send_report` | Send progress report to scott@stg.net |
| `check_inbox` | Read Maildir inbox at /var/mail/example.com/yolo/new/ |
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

### Version Control (Git)
| Tool | Description |
|------|-------------|
| `git_status` | Show current repository status |
| `git_diff` | Show changes (staged/unstaged), optional file filter |
| `git_log` | Recent commit history with oneline format |
| `git_branch` | List all branches with current marked |
| `git_checkout` | Checkout branch or restore file from HEAD |
| `git_commit` | Commit staged changes with message |
| `git_add` | Stage files for commit (all or specific) |
| `git_show` | Show details of a specific commit |
| `git_remote` | Show configured git remotes with URLs |

  ### Browser Automation
  | Tool | Description |
  |------|-------------|
  | `playwright_mcp` | Navigate URLs, interact with DOM, fill forms, screenshots |

### Hardware Integration
| Tool | Description |
|------|-------------|
| `victron` | Victron Energy BLE devices: scan, connect, monitor (SmartSolar, SmartShunt) |

**Victron Device Monitoring:**

YOLO can connect to Victron Energy devices via Bluetooth Low Energy (BLE) to read real-time sensor values from charge controllers and battery monitors.

```bash
# Scan for nearby Victron devices (10 seconds by default)
./yolo "victron scan"

# Connect to a device and get current values  
./yolo "victron connect --address AA:BB:CC:DD:EE:FF"
./yolo "victron get_values --address AA:BB:CC:DD:EE:FF"

# Monitor real-time updates for 60 seconds
./yolo "victron subscribe --address AA:BB:CC:DD:EE:FF --duration 60"

# Get device information
./yolo "victron device_info --address AA:BB:CC:DD:EE:FF"

# Disconnect from device
./yolo "victron disconnect --address AA:BB:CC:DD:EE:FF"
```

**Supported Devices:**
- SmartSolar MPPT charge controllers
- SmartShunt battery monitors  
- VE.Direct adapters with BLE support

**Supported Value Types:**
YOLO automatically decodes all standard Victron sensor values including:
- **Voltage**: Battery voltage (V106), panel voltage (V107), etc.
- **Current**: Battery current (A103), discharge/charge current
- **Power**: Panel power (W105), battery power (W108)
- **Temperature**: Device temperature sensors
- **State of Charge (SOC)**: Battery percentage (P255)
- **Energy**: Ah consumed (Ah265), Wh totals
- **Status codes**: Charging states, alarm conditions

**Automatic Device Type Detection:**
YOLO infers the device type from GATT service discovery and presents appropriate labels.

**Example Output:**
```
Connected to SmartSolar MPPT 100/30
Device Address: AA:BB:CC:DD:EE:FF
Current Values:
  • Battery Voltage: 13.42 V (V106)
  • Panel Voltage: 18.56 V (V107)
  • Battery Current: 4.2 A (A103)
  • Panel Power: 78 W (W105)
  • Status: Charging - Bulk
```

**Device Value Metadata:**
Each sensor value includes type, unit, and VE.Direct code for reference:
- `Type`: Voltage, Current, Power, Temperature, Percentage, Energy, Enum, Raw
- `Unit`: V, A, W, °C, %, mAh, Wh, h
- `Code`: VE.Direct protocol identifier (e.g., "V106", "A103")

Total: 63 tools across file operations, agent management, version control, external services, task management, system commands, browser automation, memory/context management, scheduling, and hardware integration.







## Email Processing

YOLO provides intelligent email processing for `yolo@example.com`:

- **Read inbox**: Check new emails from Maildir
- **Auto-responses**: LLM-generated natural responses to questions/requests
- **Auto-deletion**: Delete emails after successful response
- **Progress reports**: Scheduled status updates to scott@stg.net

See [DOCS/EMAIL_PROCESSING.md](DOCS/EMAIL_PROCESSING.md) for detailed email system documentation.

## Configuration

### Environment Variables (Optional)

```bash
export OLLAMA_HOST=http://localhost:11434  # Ollama server URL
export YOLO_MODEL=qwen3.5:27b-q4_K_M              # Default model to use
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

See [DOCS/tools.md](DOCS/tools.md) for detailed tool examples and the [README.md](README.md#development) section for development guidelines.

## Troubleshooting

**Ollama connection failed:**
```bash
ollama serve
curl http://localhost:11434/api/generate -d '{"model":"qwen3.5:27b-q4_K_M","prompt":"test"}'
```

**Verbose [OLLAMA DEBUG] messages appearing in terminal:**

The `OLLAMA_DEBUG` environment variable causes Ollama to output debug logs directly to your terminal. To capture these logs so YOLO can read them for self-diagnosis:

```bash
# Option 1: Disable debug mode (recommended for normal use)
unset OLLAMA_DEBUG
# or
export OLLAMA_DEBUG=0

# Option 2: Enable YOLO-managed logging (best for debugging)
# Set either of these before starting YOLO, and it will automatically:
#   - Stop any existing ollama process
#   - Restart ollama with output redirected to log files
#   - Make logs readable via check_ollama_status tool
export OLLAMA_DEBUG=1    # Enables ollama's verbose debug output
export YOLO_OLLAMA_LOG=1 # Tells YOLO to redirect output to files

# Then start/restart YOLO
./yolo

# Logs will be written to:
#   - logs/ollama.log      (stdout)
#   - logs/ollama.err.log  (stderr, includes [OLLAMA DEBUG] messages)

# Monitor logs in real-time:
tail -f logs/ollama.err.log

# YOLO can automatically check Ollama health by calling:
# Tool: check_ollama_status(lines=50)  # Reads last 50 lines from log files
```

**How it works:** When you set `OLLAMA_DEBUG=1` or `YOLO_OLLAMA_LOG=1`, YOLO detects this at startup and automatically restarts the Ollama server with output redirected to log files. This allows YOLO to read the logs using the `check_ollama_status` tool for self-diagnosis of connection issues, timeouts, and other problems.

**Why this approach:** Logging to files instead of your terminal keeps your screen clean while still capturing all debug information. YOLO can then autonomously diagnose Ollama-related issues by reading these logs.

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

For detailed architecture information, see the code comments in `agent.go`, `tools.go`, and other source files covering:

- Data flow diagrams (user chat, autonomous thinking, sub-agents)
- Component specifications
- Concurrency patterns and thread safety
- Design principles and safety mechanisms

## Contributing

See [DOCS/README.md](DOCS/README.md) for guidelines on:

- Development workflow
- Code style requirements  
- Testing standards (>90% coverage goal)
- Submission process

## License

MIT License - see LICENSE file for details.

---

**Note**: YOLO continuously improves its own code and documentation. Changes are automatically committed to git as part of the self-improvement cycle. YOLO operates without a fixed system prompt, allowing for maximum flexibility and adaptability in conversations.
