# Architecture

This document describes the internal design of YOLO, how the major components
interact, and where to find key logic in the source tree.

## High-level overview

```
┌──────────────────────────────────────────────────────────┐
│                      Terminal (raw mode)                  │
│  ┌──────────────────────┐  ┌───────────────────────────┐ │
│  │   Output (scrolling) │  │   Input (fixed bottom)    │ │
│  │   TerminalUI         │  │   InputManager            │ │
│  └──────────┬───────────┘  └─────────┬─────────────────┘ │
└─────────────┼────────────────────────┼───────────────────┘
              │                        │
              ▼                        ▼
        ┌───────────────────────────────────┐
        │            YoloAgent              │
        │  ┌─────────┐  ┌───────────────┐  │
        │  │ History  │  │ ToolExecutor  │  │
        │  │ Manager  │  │  (18 tools)   │  │
        │  └────┬─────┘  └───────┬───────┘  │
        │       │                │           │
        └───────┼────────────────┼───────────┘
                │                │
                ▼                ▼
        ┌──────────────┐  ┌──────────────┐
        │ .yolo/       │  │ OllamaClient │
        │ history.json │  │ /api/chat    │
        │ subagents/   │  │ /api/tags    │
        └──────────────┘  └──────────────┘
```

## Components

### YoloAgent (`agent.go`)

The central orchestrator.  `Run()` is the main entry point called from
`main()`.  It:

1. Loads or creates session history.
2. Initialises the terminal UI and input manager.
3. Registers signal handlers (SIGINT for cancel, SIGWINCH for resize).
4. Enters an event loop that processes user input and autonomous thinking.

Key methods:

| Method | Purpose |
|---|---|
| `chatWithAgent` | Send a user message (or autonomous prompt) to the LLM. Loops: call LLM, parse tool calls, execute tools, feed results back, repeat until the model produces a text-only reply. |
| `parseTextToolCalls` | Fallback parser for five text-based tool-call formats when the model doesn't use native `tool_calls`. |
| `handleCommand` | Process interactive slash commands (`/help`, `/model`, `/clear`, etc.). |
| `spawnSubagent` | Launch a background goroutine that runs a one-shot LLM call and writes results to `.yolo/subagents/`. |
| `switchModel` | Validate and switch to a different Ollama model. |

### OllamaClient (`ollama.go`)

HTTP client for the Ollama REST API.

| Method | Endpoint | Purpose |
|---|---|---|
| `ListModels` | `GET /api/tags` | Enumerate available models |
| `GetModelContextLength` | `POST /api/show` | Detect a model's context window size |
| `Chat` | `POST /api/chat` | Streaming chat completion with tool definitions |

`Chat` reads the response line-by-line, printing display text to the
terminal as it arrives, and accumulates tool calls for the agent to execute.

### ToolExecutor (`tools.go`)

Dispatches tool calls from the LLM to concrete implementations.  All file
operations are sandboxed under the working directory via `safePath()`, which
rejects absolute paths and directory-traversal attempts.

The 18 built-in tools:

| Tool | Description |
|---|---|
| `read_file` | Read file contents with optional offset/limit |
| `write_file` | Create or overwrite a file |
| `edit_file` | First-occurrence string replacement |
| `list_files` | Glob matching, including recursive `**/` patterns |
| `search_files` | Regex search across file contents |
| `run_command` | Execute a shell command (stdin is /dev/null, timeout enforced) |
| `make_dir` | Create directories recursively |
| `remove_dir` | Remove a directory tree |
| `copy_file` | Copy a file, creating destination directories |
| `move_file` | Move/rename, with cross-filesystem fallback |
| `spawn_subagent` | Create a background sub-agent |
| `list_subagents` | List sub-agent statuses |
| `read_subagent_result` | Retrieve a sub-agent's output |
| `summarize_subagents` | Aggregate sub-agent statistics |
| `list_models` | List available Ollama models |
| `switch_model` | Change the active model |
| `think` | Record reasoning without side effects |
| `restart` | Rebuild from source and `exec` the new binary |

### HistoryManager (`history.go`)

Thread-safe persistence layer for conversation messages and evolution events.
Data is stored as JSON in `.yolo/history.json`.

- **Atomic writes**: saves to a `.tmp` file then renames.
- **Corruption recovery**: if the JSON is malformed on load, resets to empty.
- **Context conversion**: `GetContextMessages` maps internal roles (tool,
  system) to `user`-role messages with prefixes so the LLM can understand them.

### InputManager (`input.go`)

Handles terminal input in raw mode, running in its own goroutine:

- Reads stdin byte-by-byte.
- Assembles UTF-8 multi-byte characters from individual bytes.
- Handles control characters (backspace, Ctrl-C, Ctrl-D, Ctrl-U, Ctrl-W).
- Consumes ANSI escape sequences (arrow keys, function keys) without leaking
  bytes into the input buffer.
- Sends completed lines to the agent via a buffered channel.

### TerminalUI (`terminal.go`)

Manages a split-screen terminal layout:

- **Output region** (top): scrollable, with word wrapping and ANSI-aware
  cursor tracking.
- **Divider** (second-to-last row): a horizontal line.
- **Input line** (bottom row): fixed position, with horizontal scrolling for
  long input.

Also provides:
- `cprint` / `cprintNoNL`: colour-aware output helpers.
- `stripAnsiCodes`: removes ANSI escapes for width calculations.
- `Spinner`: animated thinking indicator.

### Configuration (`config.go`)

Compile-time constants and environment-variable overrides:

| Constant | Default | Env var | Purpose |
|---|---|---|---|
| `YoloDir` | `.yolo` | — | State directory |
| `IdleThinkDelay` | 30s | — | Idle time before autonomous thinking |
| `ThinkLoopDelay` | 120s | — | Interval between think cycles |
| `MaxContextMessages` | 40 | — | Max messages in LLM context |
| `CommandTimeout` | 30s | — | Shell command timeout |
| `DefaultNumCtx` | 8192 | — | Default context window size |
| `OllamaURL` | `localhost:11434` | `OLLAMA_URL` | Ollama API endpoint |
| `NumCtxOverride` | (auto) | `YOLO_NUM_CTX` | Force context window size |

## Data flow

### User chat

```
User types "fix the bug in main.go" ──► InputManager.Lines channel
  │
  ▼
YoloAgent.chatWithAgent("fix the bug in main.go", autonomous=false)
  │
  ├─► history.AddMessage("user", ...)
  │
  ├─► Build context: system prompt + last N history messages + round messages
  │
  └─► Loop:
      ├─► ollama.Chat(ctx, model, allMsgs, tools)
      │     └─► Streams response to terminal, returns ChatResult
      │
      ├─► If ChatResult.ToolCalls is non-empty:
      │     ├─► For each call: tools.Execute(name, args)
      │     ├─► Append tool results to roundMsgs
      │     └─► Continue loop
      │
      └─► If no tool calls: save final text to history, exit loop
```

### Autonomous thinking

```
No user input for IdleThinkDelay seconds
  │
  ▼
YoloAgent.chatWithAgent("", autonomous=true)
  │
  ├─► System message instructs the model to continue making progress
  └─► Same tool-calling loop as above
```

### Sub-agent spawning

```
LLM calls spawn_subagent(prompt="analyze test coverage")
  │
  ▼
YoloAgent.spawnSubagent(task, model)
  │
  ├─► Assigns monotonic ID (e.g. #3)
  ├─► Launches goroutine:
  │     ├─► Calls ollama.Chat (no tools, single turn)
  │     └─► Writes result to .yolo/subagents/agent_3.json
  └─► Returns immediately: "Sub-agent #3 spawned"
```

## Internal packages

### `internal/mcp/` — MCP Server

Implements the [Model Context Protocol](https://modelcontextprotocol.io/)
server-side.  Provides:

- JSON-RPC 2.0 request routing (`handleRequest`)
- Tool, resource, and prompt registration
- Log level management
- SSE transport (`ServeSSE`)

### `internal/mcpclient/` — MCP Client

Client for connecting to external MCP server processes:

- Starts a subprocess, communicates over stdin/stdout with JSON-RPC
- Auto-discovers capabilities (tools, prompts, resources)
- `ToolRegistry` for managing tools from multiple MCP servers

## File layout

```
.
├── main.go                 # Entry point
├── agent.go                # YoloAgent: orchestration, chat loop, commands
├── ollama.go               # OllamaClient: LLM communication
├── tools.go                # ToolExecutor: tool definitions and dispatch
├── history.go              # HistoryManager: conversation persistence
├── input.go                # InputManager: raw terminal input
├── terminal.go             # TerminalUI: split-screen rendering
├── config.go               # Constants, env vars, ANSI colours
├── SYSTEM_PROMPT.md        # System prompt template (interpolated at runtime)
├── internal/
│   ├── mcp/                # MCP protocol server
│   │   ├── server.go
│   │   ├── types.go
│   │   └── errors.go
│   └── mcpclient/          # MCP protocol client
│       ├── client.go
│       └── registry.go
└── .yolo/                  # Runtime state (gitignored)
    ├── history.json
    └── subagents/
        └── agent_*.json
```

## Design principles

1. **Safety first**: `safePath` prevents directory traversal. Shell commands
   run with stdin connected to /dev/null and a timeout.
2. **Graceful degradation**: tool call parsing tries five formats before
   giving up. History corruption resets to empty rather than crashing.
3. **Concurrency**: InputManager, sub-agents, and the Spinner run in their
   own goroutines. Shared state is protected by mutexes.
4. **Minimal dependencies**: only `golang.org/x/term` beyond the standard
   library.
5. **Self-modification**: the agent can read and edit its own source, rebuild
   itself, and `exec` the new binary via the `restart` tool.
