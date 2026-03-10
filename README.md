# YOLO - Your Own Living Operator

A self-evolving AI agent for software development that continuously runs,
thinks, and improves — even when you're not typing.

## Features

- **Autonomous operation** — runs in the background, thinks on its own after
  30 seconds of idle, and acts without asking for permission.
- **18 built-in tools** — file I/O, shell commands, regex search, sub-agent
  spawning, model switching, and self-restart.
- **Conversation history** — persisted to `.yolo/history.json`; sessions
  resume automatically.
- **Sub-agents** — background goroutines that run focused LLM tasks in
  parallel.
- **Split-screen terminal UI** — scrollable output on top, fixed input line
  on the bottom, with word wrapping, horizontal scrolling, and resize
  handling.
- **Self-improving** — can read and modify its own source code, rebuild, and
  replace the running binary.
- **UTF-8 input** — full support for multi-byte characters (accented
  letters, CJK, emoji).

## Quick start

```bash
# Requires Go 1.24+ and a running Ollama instance
git clone https://github.com/yourusername/yolo.git
cd yolo
go build -o yolo .
./yolo
```

On first launch YOLO connects to Ollama, lists available models, and asks
you to pick one. After that, just type what you want done.

## Configuration

| Environment variable | Default | Description |
|---|---|---|
| `OLLAMA_URL` | `http://localhost:11434` | Ollama API endpoint |
| `YOLO_NUM_CTX` | *(auto-detected)* | Override the model's context window size |

Compile-time constants are in [`config.go`](config.go):

| Constant | Default | Purpose |
|---|---|---|
| `YoloDir` | `.yolo` | State directory |
| `IdleThinkDelay` | 30 s | Idle time before autonomous thinking |
| `ThinkLoopDelay` | 120 s | Interval between think cycles |
| `MaxContextMessages` | 40 | Max messages sent to the LLM |
| `CommandTimeout` | 30 s | Shell command timeout |
| `DefaultNumCtx` | 8192 | Fallback context window size |

## Usage

### Interactive commands

| Command | Action |
|---|---|
| `/help` | Show available commands |
| `/model` | Show current model |
| `/models` | List available Ollama models |
| `/switch <name>` | Switch to a different model |
| `/history` | Show message and evolution counts |
| `/clear` | Clear conversation history |
| `/status` | Show agent status |
| `/restart` | Rebuild and restart YOLO |
| `/exit`, `/quit` | Exit |

### Keyboard shortcuts

| Key | Action |
|---|---|
| `Enter` | Submit input |
| `Ctrl-C` | Cancel current operation (or exit if idle) |
| `Ctrl-D` | Exit (when input is empty) |
| `Ctrl-U` | Clear entire input line |
| `Ctrl-W` | Delete last word |

## Architecture

See [ARCHITECTURE.md](ARCHITECTURE.md) for a full design document. The
short version:

```
User input ──► YoloAgent ──► OllamaClient ──► Ollama /api/chat
                  │                                   │
                  │◄────── tool calls ◄───────────────┘
                  │
                  ├──► ToolExecutor (file ops, shell, sub-agents)
                  ├──► HistoryManager (.yolo/history.json)
                  └──► TerminalUI (split-screen output)
```

### State directory

YOLO stores all runtime state in `.yolo/`:

```
.yolo/
├── history.json          # Conversation history and config
└── subagents/
    ├── agent_1.json      # Sub-agent results
    └── agent_2.json
```

## Safety

- **Path sandboxing** — all file operations are validated by `safePath()` to
  stay within the working directory.
- **Command timeout** — shell commands are killed after 30 seconds.
- **stdin isolation** — child processes get `/dev/null` as stdin so they
  can't steal terminal input.
- **Atomic history writes** — write-to-temp then rename prevents corruption.
- **Graceful shutdown** — Ctrl-C cancels the in-flight LLM request, saves
  history, and restores the terminal.

## Development

```bash
# Run all tests
go test ./...

# Verbose with race detection
go test -race -v ./...

# Cross-compile
GOOS=linux  GOARCH=amd64 go build -o yolo-linux  .
GOOS=darwin GOARCH=arm64 go build -o yolo-darwin  .
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for development workflow, code style,
and how to add new tools.

## License

MIT License — see LICENSE file for details.
