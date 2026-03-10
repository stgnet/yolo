# Contributing to YOLO

## Prerequisites

- Go 1.24 or later
- A running [Ollama](https://ollama.com) instance (for integration testing)

## Getting started

```bash
git clone https://github.com/yourusername/yolo.git
cd yolo
go build -o yolo .
./yolo
```

## Project structure

See [ARCHITECTURE.md](ARCHITECTURE.md) for a detailed breakdown. In short:

| File | Responsibility |
|---|---|
| `main.go` | Entry point |
| `agent.go` | Orchestration, chat loop, slash commands |
| `ollama.go` | Ollama API client |
| `tools.go` | Tool definitions and implementations |
| `history.go` | Conversation persistence |
| `input.go` | Raw terminal input handling |
| `terminal.go` | Split-screen terminal UI |
| `config.go` | Constants and environment variables |

## Running tests

```bash
# All tests
go test ./...

# Verbose output
go test -v ./...

# A specific package
go test -v ./internal/mcp/...

# With race detection
go test -race ./...
```

Tests do not require a running Ollama instance — they use file-system
operations, in-memory structs, and the MCP protocol layer directly.

## Code style

- Run `gofmt` before committing. The CI will reject unformatted code.
- Follow standard Go conventions:
  - Exported types and functions must have godoc comments.
  - Error messages start with a lowercase letter and do not end with
    punctuation.
  - Use `t.Helper()` in test helpers.
- Tool error strings must start with `"Error:"` so that callers (and tests)
  can detect failures via the `isError()` helper.

## Adding a new tool

1. **Define the tool** — add a `toolDef(...)` entry to the `ollamaTools`
   slice in `tools.go`.
2. **Register the name** — add the tool name to the `validTools` slice
   (same file). This is needed for text-based tool-call parsing.
3. **Implement the handler** — write a method on `ToolExecutor`.  Use
   `safePath()` for any file path argument.
4. **Wire it up** — add a `case` in `ToolExecutor.Execute()`.
5. **Write tests** — add table-driven tests covering success, missing
   arguments, and error paths.

## Making changes

1. Create a feature branch off `main`.
2. Make small, focused commits with clear messages.
3. Ensure `go test ./...` passes before pushing.
4. Open a pull request with a description of what changed and why.

## Conventions

### Error handling

- Tool implementations return error strings inline (prefixed with `"Error:"`).
  They do not return Go `error` values.
- Infrastructure code (HistoryManager, OllamaClient) uses standard Go errors.

### Thread safety

- `YoloAgent.mu` protects `busy`, `cancelChat`, and `subagentCounter`.
- `HistoryManager.mu` protects `Data` reads and writes.
- `TerminalUI.mu` protects all terminal output.
- `InputManager.mu` protects the input buffer.

### File safety

All file tool operations must go through `safePath()` to validate that paths
stay within the working directory. Never use absolute paths directly.
