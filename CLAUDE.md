# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Run

```bash
go build -o yolo .        # Build executable
./build.sh                # Alternative: build script
```

The agent detects source staleness automatically — if source files are newer than the binary it shows "NEEDS COMPILE", and if the binary was rebuilt it shows "NEEDS RESTART".

## Testing

```bash
go test ./...             # Run all tests
go test -v ./...          # Verbose output
go test -cover ./...      # With coverage
go test -race ./...       # With race detector
go test -run TestName ./  # Single test in root package
go test -run TestName ./tools/  # Single test in subpackage
```

Tests live alongside source code in `*_test.go` files. The `testify` library is used for assertions.

## Environment Variables

| Variable | Default | Purpose |
|----------|---------|---------|
| `OLLAMA_HOST` | `http://localhost:11434` | Ollama server URL |
| `YOLO_MODEL` | `qwen3.5:27b` | Default LLM model |
| `YOLO_NUM_CTX` | `8192` | Context window size |
| `OLLAMA_DEBUG` | - | Enable Ollama debug logging |
| `YOLO_OLLAMA_LOG` | - | Redirect Ollama output to log files |

## Architecture

YOLO (Your Own Living Operator) is a self-evolving AI agent for autonomous software development, powered by local LLMs via Ollama.

### Core Loop (all in root package)

1. **`main.go`** — Entry point. Sets up logging, validates TTY, creates `YoloAgent`.
2. **`agent.go`** (~1900 lines) — Central orchestrator. Runs the main event loop: user input → LLM chat → tool execution → feed results back → repeat until no more tool calls.
3. **`ollama.go`** — HTTP client for Ollama REST API (`/api/chat`, `/api/tags`, `/api/generate`). Handles streaming, context window detection, native tool call schemas.
4. **`tools.go`** (~69KB) — Tool dispatcher. Routes tool calls to 32+ implementations. Tool definitions use `toolDef()` helper into `ollamaTools` slice.
5. **`tools_*.go`** — Tool implementations split by domain: `tools_email.go`, `tools_inbox.go`, `tools_playwright.go`, `tools_todo.go`, `tools_webpage.go`.
6. **`history.go`** — Persistent conversation log in `.yolo/history.json`.
7. **`terminal.go`** (~31KB) — Split-screen terminal UI with ANSI color support.
8. **`input.go`** — Async terminal input via goroutine with signal handling (SIGINT, SIGWINCH).

### Supporting Packages

- **`config/`** — Thread-safe configuration with `atomic.Value` fields
- **`tools/`** — Shared tool utilities: `file_ops.go`, `web.go`, `communication.go`, `helpers.go`
- **`tools/playwright/`** — Browser automation via Playwright
- **`tools/websearch/`** — DuckDuckGo search integration
- **`tools/todo/`** — Todo list management
- **`terminalui/`** — Terminal utilities, output buffer, ANSI sanitization
- **`concurrency/`** — Limiter, pool, and group utilities
- **`inputmanager/`** — Input handling abstraction
- **`search/`** — Search result caching
- **`utils/`** — File operations and safety checks
- **`types/`** — Shared type definitions

### Tool Registration

Tools are defined in `tools.go` as entries in the `ollamaTools` slice using `toolDef()`. Each tool has a name, description, and parameter schema. Implementations are methods on `*ToolExecutor` following the pattern `func (e *ToolExecutor) toolName(args map[string]any) string`.

### Tool Call Parsing

The agent supports two parsing modes:
- **Native**: JSON tool calls from the LLM response
- **Text fallback**: Parses `[tool activity] toolname(args)` syntax for models that don't support native tool calling

Deduplication prevents duplicate execution from streaming artifacts. File mutation tools use fail-fast — if one fails, remaining file operations in the batch are skipped.

### Concurrency

- `YoloAgent.mu sync.Mutex` protects agent state (busy flag, cancel, counters, handoffs)
- `OllamaClient.cacheMu sync.RWMutex` for context cache
- `atomic.Value` for config fields
- Background handoff mechanism allows user interruption mid-loop

### Runtime State

The `.yolo/` directory (gitignored) stores runtime state: `history.json`, `subagents/` results, `todos.json`, and optional `knowledge.md`.
