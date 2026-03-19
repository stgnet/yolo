# Architecture Deep Dive

This document provides detailed technical specifications for YOLO's internal design and components. For general usage information, see [README.md](README.md).

## High-level Architecture

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

## Component Specifications

### YoloAgent (`agent.go`)

The central orchestrator. `Run()` is the main entry point called from `main()`. It:

1. Loads or creates session history
2. Initializes terminal UI and input manager
3. Registers signal handlers (SIGINT for cancel, SIGWINCH for resize)
4. Enters event loop processing user input and autonomous thinking

**Key Methods:**

- `chatWithAgent(msg string, autonomous bool)`: Send message to LLM, loops tool calls until text-only reply
- `parseTextToolCalls(response string)`: Fallback parser for 5 text-based tool-call formats
- `handleCommand(cmd string)`: Process `/help`, `/model`, `/clear`, etc.
- `spawnSubagent(task string, model string)`: Launch background goroutine, returns monotonic ID
- `switchModel(modelName string)`: Validate and switch active model

### OllamaClient (`ollama.go`)

HTTP client for Ollama REST API with streaming support.

**Endpoints:**

| Method | Endpoint | Purpose |
|------|-|---|--|
| `ListModels()` | `GET /api/tags` | Enumerate available models |
| `GetModelContextLength(model string) int` | `POST /api/show` | Detect context window size |
| `Chat(ctx, model, messages, tools)` | `POST /api/chat` | Streaming chat with tools |

**Streaming Behavior:** Reads response line-by-line, prints display text to terminal as it arrives, accumulates tool calls for agent execution. Handles thinking models with `content.text` vs `content.thinking`.

### ToolExecutor (`tools.go`)

Dispatches tool calls from LLM to concrete implementations. All file operations sandboxed under working directory via `safePath()` which rejects absolute paths and directory traversal.

**Safety Mechanisms:**
- `safePath(path string)`: Rejects paths starting with `/` or containing `..`
- Shell commands: stdin=/dev/null, 30s timeout
- File operations relative to configured working directory only

### HistoryManager (`history.go`)

Thread-safe persistence layer for conversation messages. Data stored as JSON in `.yolo/history.json`.

**Features:**
- **Atomic writes**: Save to `.tmp` file then rename (prevents corruption on crash)
- **Corruption recovery**: Malformed JSON resets to empty rather than crashing
- **Context conversion**: Maps internal roles (tool, system) to prefixed user messages for LLM understanding
- **Mutex protection**: All reads/writes synchronized via `sync.RWMutex`

### InputManager (`input.go`)

Handles terminal input in raw mode, running in its own goroutine:

- Reads stdin byte-by-byte
- Assembles UTF-8 multi-byte characters from individual bytes
- Handles control chars: backspace, Ctrl-C (cancel), Ctrl-D (EOF), Ctrl-U (clear line), Ctrl-W (delete word)
- Consumes ANSI escape sequences (arrow keys, function keys) without leaking bytes
- Sends completed lines to agent via `buffered channel` with input delay (default 10s for multiline paste handling)

### TerminalUI (`terminal.go`)

Manages split-screen layout with dynamic bottom area:

**Output region (top):** Scrollable, word-wrapped, ANSI-aware cursor tracking. Region shrinks/grows as bottom area changes.

**Divider:** Horizontal line labelled `──you──` separating output from input. Moves up when input grows.

**Input area (bottom):** Multiline editing buffer. Expands upward instead of horizontal scroll. Capped at 50% terminal height. After user presses Enter and pauses for configured delay (`DefaultInputDelay`, default 10s), entire buffer sent as one block to prevent multiline paste splitting.

**Helpers:** `cprint`/`cprintNoNL` (colour-aware output), `stripAnsiCodes` (removes escapes for width calculations)

### Configuration (`config.go`)

Compile-time constants and environment-variable overrides with thread-safe access:

| Variable | Default | Env Var | Purpose | Access Method |
|----------|----------|-|-|---|
| `YoloDir` | `.yolo` | — | State directory | `GetYoloDir()` |
| `IdleThinkDelay` | 30s | — | Idle time before autonomous thinking | `GetIdleThinkDelay()` |
| `ThinkLoopDelay` | 120s | — | Interval between think cycles | `GetThinkLoopDelay()` |
| `MaxContextMessages` | 40 | — | Max messages in LLM context | `GetMaxContextMessages()` |
| `CommandTimeout` | 30s | — | Shell command timeout | `GetCommandTimeout()` |
| `DefaultNumCtx` | 8192 | — | Default context window size | `GetDefaultNumCtx()` |
| `OllamaURL` | `localhost:11434` | `OLLAMA_URL` | Ollama API endpoint | `GetOllamaURL()` |
| `NumCtxOverride` | (auto) | `YOLO_NUM_CTX` | Force context window size | `GetNumCtxOverride()` |

**Thread Safety:** All configuration uses atomic operations or mutex protection. Initialized via single init() function to prevent race conditions.

## Data Flow Diagrams

### User Chat Flow

```
User types "fix the bug in main.go" ──► InputManager.Lines channel
  │
  ▼
YoloAgent.chatWithAgent("fix...", autonomous=false)
  │
  ├─► history.AddMessage("user", ...)
  │
  ├─► Build context: system prompt + last N history + round messages
  │
  └─► Loop:
      ├─► ollama.Chat(ctx, model, allMsgs, tools)
      │     └─► Streams response to terminal, returns ChatResult
      │
      ├─► If ChatResult.ToolCalls non-empty:
      │     ├─► For each call: tools.Execute(name, args)
      │     ├─► Append tool results to roundMsgs
      │     └─► Continue loop with new context
      │
      └─► If no tool calls: save final text to history, exit loop
```

### Autonomous Thinking Flow

```
No user input for IdleThinkDelay seconds
  │
  ▼
YoloAgent.chatWithAgent("", autonomous=true)
  │
  ├─► System message instructs model to continue making progress
  ├─► Same tool-calling loop as user chat
  └─► Updates history with autonomous thoughts and actions
```

### Sub-agent Spawning Flow

```
LLM calls spawn_subagent(prompt="analyze test coverage")
  │
  ▼
YoloAgent.spawnSubagent(task, model)
  │
  ├─► Assigns monotonic ID (e.g. #3 via atomic.AddInt64)
  ├─► Launches goroutine:
  │     ├─► Calls ollama.Chat (no tools, single turn)
  │     └─► Writes result to .yolo/subagents/agent_3.json
  └─► Returns immediately: "Sub-agent #3 spawned"
```

## Concurrency Package (`concurrency/`)

Advanced synchronization primitives for managing goroutines safely.

### ThreadPool

Worker pool pattern limiting concurrent execution to fixed number of goroutines.

```go
pool := concurrency.NewThreadPool(10)
pool.Submit(func() { /* work */ })
pool.Close() // Waits for all jobs to complete
```

Features: Fixed worker count, context-aware submission, graceful shutdown.

### Group (Structured Concurrency)

Inspired by Java structured concurrency and Go's sync.WaitGroup. Ensures all goroutines complete before parent returns.

```go
g := concurrency.NewGroup(ctx)
g.Go(func(c context.Context) error { return work() })
g.Run() // Waits and cancels context
errors := g.Errors()
```

Features: Automatic error collection, context propagation, fan-out/fan-in patterns.

### Limiter (Semaphore-based Rate Limiting)

Controls maximum concurrent operations using semaphore pattern.

```go
limiter := concurrency.NewLimiter(5).WithTimeout(10 * time.Second)
limiter.Execute(func() error { return work() })
// Or with context:
limiter.ExecuteWithContext(ctx, func(c context.Context) error { return work() })
```

Features: `TryAcquire()` non-blocking, `Acquire()` with timeout, automatic acquire/release wrappers.

### LimiterGroup

Combines Group structured concurrency with rate limiting. All goroutines respect same limit. Ideal for parallel API calls with rate limits.

## Design Principles

1. **Safety first**: `safePath` prevents directory traversal. Shell commands run with stdin=/dev/null and timeout.
2. **Graceful degradation**: Tool call parsing tries 5 formats before giving up. History corruption resets to empty rather than crashing.
3. **Concurrency safety**: InputManager and sub-agents in separate goroutines. Shared state protected by mutexes and atomic operations.
4. **Minimal dependencies**: Only `golang.org/x/term` beyond standard library.
5. **Self-modification**: Agent can read/edit own source, rebuild, and exec new binary via `restart` tool.

## File Layout

```
.
├── main.go                 # Entry point
├── agent.go                # YoloAgent: orchestration, chat loop, commands
├── ollama.go               # OllamaClient: LLM communication
├── tools*.go               # ToolExecutor: definitions and dispatch
├── history.go              # HistoryManager: conversation persistence
├── input.go                # InputManager: raw terminal input
├── terminal.go             # TerminalUI: split-screen rendering
├── config.go               # Constants, env vars, thread-safe config access
├── SYSTEM_PROMPT.md        # System prompt template (interpolated at runtime)
├── README.md               # Comprehensive documentation
├── ARCHITECTURE.md         # This file - technical deep dive
├── CONTRIBUTING.md         # Development guidelines
├── EMAIL_PROCESSING.md     # Email system details
└── .yolo/                  # Runtime state (gitignored)
    ├── history.json        # Conversation history
    ├── todos.json          # Task list
    └── subagents/          # Background agent results
        └── agent_*.json
```

## Thread Safety Guarantees

| Component | Synchronization | Protected State |
|-----------|---|---|
| Config | `atomic.Value` + `sync.RWMutex` | All configuration variables |
| HistoryManager | `sync.RWMutex` | Messages slice, file I/O |
| Agent (model switch) | `sync.RWMutex` | `currentModel` string |
| Agent (sub-agent ID) | `atomic.Int64` | `subagentIDCounter` |
| InputManager | Channel-based | Lines sent to agent |

See [CONTRIBUTING.md](CONTRIBUTING.md) for concurrency testing guidelines with `-race` flag.
