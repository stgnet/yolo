# YOLO - Your Own Living Operator

A self-evolving AI agent for software development that continuously runs, thinks, and improves—even when you're not typing.

## Features

- **Autonomous Operation**: Runs in the background and responds to terminal input
- **Context-Aware**: Maintains conversation history and project context
- **Self-Improving**: Can analyze and modify its own codebase
- **Tool Integration**: Built-in tools for file operations, command execution, and more
- **Subagent System**: Creates specialized sub-agents for focused tasks
- **Terminal UI**: Split-screen interface with scrolling output and input line

## Installation

```bash
# Clone the repository
git clone https://github.com/yourusername/yolo.git
cd yolo

# Build the binary
go build -o yolo main.go

# Run the agent
./yolo
```

## Configuration

YOLO reads configuration from environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `OLLAMA_URL` | `http://localhost:11434` | Ollama API endpoint |

### Runtime Constants

The following constants control agent behavior (modify in `main.go`):

```go
const (
    YoloDir           = ".yolo"          // Directory for storing state
    IdleThinkDelay    = 30              // Seconds of idle time before autonomous thinking
    ThinkLoopDelay    = 120             // Seconds between think cycles
    MaxContextMessages = 40            // Maximum messages in context window
    MaxToolOutput     = 0               // Tool output truncation (0 = disabled)
    ToolNudgeAfter    = 0               // Disabled by default
    CommandTimeout    = 30              // Shell command timeout in seconds
)
```

## Usage

### Basic Commands

Once YOLO is running, you can interact with it via the terminal input line:

1. **Start Task**: Describe what you want to accomplish
2. **Interrupt**: Press Enter without typing to interrupt current operation
3. **Clear Input**: Use `esc` to clear the input line
4. **Quit**: Type `quit` or press Ctrl+C

### Example Session

```
$ ./yolo
[YOLO] Starting agent...
[UI] Terminal UI initialized

> Create a test file for the main package
[THINKING] Analyzing task: create a test file for the main package
[TOOL] read_file called on /Users/sgriepentrog/src/yolo/main.go
[TOOL] write_file called to /Users/sgriepentrog/src/yolo/main_test.go
[DONE] Created test file successfully

> Run the tests
[THINKING] Analyzing task: run the tests
[TOOL] run_command called with "go test -v"
[DONE] All tests passing!

> Improve the documentation
[THINKING] Analyzing task: improve the documentation
...
```

## Architecture

### Core Components

#### TerminalUI
Manages split terminal output with:
- Scrollable output region (top)
- Fixed input line (bottom)
- Automatic resizing support
- ANSI code handling

#### OllamaClient
Handles communication with Ollama:
- Model listing (`ollama list`)
- Chat completions (`/api/chat`)
- Tool definitions and responses

#### ToolExecutor
Provides built-in tools:
- `read_file`: Read file contents
- `write_file`: Write files (with safety checks)
- `run_command`: Execute shell commands
- `create_subagent`: Spawn specialized agents
- And more...

#### YoloAgent
Main agent logic:
- Input processing
- Context management
- Tool invocation
- Autonomous thinking cycles

### State Management

YOLO stores state in `.yolo/` directory:
- `history.json`: Conversation history
- `state.json`: Current agent state
- `subagents/`: Subagent directories

## Safety Features

1. **Path Validation**: File operations restricted to project directory
2. **Command Timeout**: Prevents hung processes (30s default)
3. **Output Truncation**: Configurable limits on tool output size
4. **Graceful Shutdown**: Cleanup on Ctrl+C or exit

## Development

### Running Tests

```bash
go test -v ./...
```

### Building for Distribution

```bash
# Cross-compile for different platforms
GOOS=linux GOARCH=amd64 go build -o yolo-linux main.go
GOOS=darwin GOARCH=amd64 go build -o yolo-darwin main.go
GOOS=windows GOARCH=amd64 go build -o yolo.exe main.go
```

### Adding New Tools

1. Define tool schema in `ToolDef` struct
2. Implement handler in `ToolExecutor.Execute()`
3. Add validation and error handling
4. Update tests in `main_test.go`

## License

MIT License - See LICENSE file for details

## Contributing

Contributions welcome! Please:
1. Fork the repository
2. Create a feature branch
3. Write tests for new functionality
4. Submit a pull request
