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
```

Tests live alongside source code in `*_test.go` files. The `testify` library is used for assertions.

## Environment Variables

| Variable | Default | Purpose |
|----------|---------|---------|
| `OLLAMA_URL` | `http://localhost:11434` | Ollama server URL |
| `YOLO_MODEL` | `qwen3.5:27b` | Default LLM model |
| `YOLO_NUM_CTX` | `8192` | Context window size |
| `OLLAMA_DEBUG` | - | Enable Ollama debug logging |
| `YOLO_OLLAMA_LOG` | - | Redirect Ollama output to log files |

## Architecture

YOLO (Your Own Living Operator) is a self-evolving AI agent for autonomous software development, powered by local LLMs via Ollama.

### Core Loop (all in root package)

1. **`main.go`** — Entry point. Sets up logging, validates TTY, creates `YoloAgent`.
2. **`agent.go`** — Central orchestrator. Runs the main event loop: user input → LLM chat → tool execution → feed results back → repeat until no more tool calls.
3. **`ollama.go`** — HTTP client for Ollama REST API (`/api/chat`, `/api/tags`, `/api/generate`). Handles streaming, context window detection, native tool call schemas.
4. **`tools.go`** — Tool dispatcher. Routes tool calls to 32+ implementations. Contains tool definitions (`ollamaTools` slice), `ToolExecutor` struct, `Execute()` switch, error types, and argument helpers.
5. **`tools_file.go`** — File operation tools: read, write, edit, list, search, copy, move, make/remove directory, glob.
6. **`tools_command.go`** — Shell command execution (`run_command`) and binary rebuild/restart (`restart`).
7. **`tools_subagent.go`** — Background sub-agent spawning, listing, result reading, and summarization.
8. **`tools_model.go`** — Model listing, switching, and Ollama status checking.
9. **`tools_search.go`** — Web search via DuckDuckGo and Wikipedia, with in-memory result caching.
10. **`tools_reddit.go`** — Reddit API integration: search, subreddit listing, thread reading.
11. **`tools_email.go`** — Email composition and sending via Postfix/sendmail.
12. **`tools_inbox.go`** — Email inbox processing and Maildir integration.
13. **`tools_todo.go`** — Todo list: `TodoList` struct, CRUD operations, persistence to `.todo.json`, formatting, and tool wrappers.
14. **`tools_webpage.go`** — Web page fetching and HTML-to-text conversion.
15. **`tools_playwright.go`** — Browser automation via Playwright (JavaScript-based).
16. **`tools_gog.go`** — Google CLI wrapper (Gmail, Calendar, Drive, etc.).
17. **`history.go`** — Persistent conversation log in `.yolo/history.json` with automatic pruning (max 200 messages).
18. **`terminal.go`** — Split-screen terminal UI with ANSI color support.
19. **`input.go`** — Async terminal input via goroutine with signal handling (SIGINT, SIGWINCH).
20. **`tts.go`** — Text-to-speech output with platform detection (macOS `say`, espeak-ng).
21. **`config.go`** — Constants (timeouts, limits, context window defaults) and ANSI color definitions.
22. **`yoloconfig.go`** — Unified configuration: persistent settings in `.yolo/config.json` (model, modes) plus runtime paths from environment variables (Ollama URL, context override, subagent dir).

### Supporting Packages

- **`concurrency/`** — Limiter, pool, and group utilities
- **`email/`** — Email parsing, security validation, header encoding
- **`errors/`** — Custom error types with structured fields (FileNotFoundError, ToolExecutionError, etc.)

### Configuration

All configuration is managed by `YoloConfig` in `yoloconfig.go`, stored in `.yolo/config.json`. This includes:
- Persistent settings: model, terminal mode, debug mode, auto mode, think mode
- Runtime paths: Ollama URL (from `OLLAMA_URL` env), context window override (from `YOLO_NUM_CTX` env), subagent directory

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
- `TodoList.mu sync.RWMutex` for thread-safe todo operations
- Background handoff mechanism allows user interruption mid-loop

### Runtime State

The `.yolo/` directory (gitignored) stores runtime state: `history.json`, `subagents/` results, `config.json`, and optional `knowledge.md`. The todo list is stored in `.todo.json` in the project root.
