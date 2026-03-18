# YOLO - Your Own Living Operator

**Version**: 1.0 | **Status**: ✅ Production Ready | **Last Updated**: 2026-03-18T13:37:52-04:00

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

# Pull a model
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

For detailed setup instructions, see the [Documentation Hub](DOCS/README.md).

## Core Capabilities

| Category | Features | Docs |
|----------|----------|------|
| **Self-Improvement** | Auto bug fixes, test coverage, code optimization | [Autonomous Operations](DOCS/AUTONOMOUS_OPERATIONS.md) |
| **Email System** | Read inbox, auto-respond, send reports | [Email Processing](EMAIL_PROCESSING.md) |
| **Web Integration** | DuckDuckGo search, Reddit, Google Workspace | [Tools Reference](DOCS/tools.md) |
| **File Operations** | Read/write/edit files, directory management | [Tools Reference](DOCS/tools.md) |
| **Task Management** | Todo list with add/complete/delete | [Tools Reference](DOCS/tools.md#task-management) |
| **Parallel Work** | Spawn sub-agents for concurrent tasks | [Autonomous Operations](DOCS/AUTONOMOUS_OPERATIONS.md#sub-agent-pattern) |

## Current Status

### ✅ Operational Health

- **Tests**: All passing (coverage improving toward 80% target)
- **Code Quality**: No data races, formatted, no vet warnings
- **Git**: Clean working directory, up-to-date with remote

### 📊 Coverage Highlights

| Package | Coverage | Status |
|---------|----------|--------|
| Concurrency | 95.3% | ✅ Excellent |
| Email | 90.0% | ✅ Good |
| Main package | 60.4% | ⚠️ Improving |

### 🎯 Recent Improvements

1. **Fixed data race** in `handoffRemainingTools()` goroutine
2. **Mock-based tests** for websearch to avoid external dependencies
3. **Enhanced email processing** with direct LLM generation
4. **Consolidated documentation** into unified hub

## Documentation

📚 **Complete Documentation Hub**: [DOCS/README.md](DOCS/README.md)

Key guides:
- [Autonomous Operations Guide](DOCS/AUTONOMOUS_OPERATIONS.md) - How YOLO works autonomously
- [Tools Reference](DOCS/tools.md) - All available tools with examples
- [Architecture Overview](ARCHITECTURE.md) - System design and structure
- [Email Processing](EMAIL_PROCESSING.md) - Email system deep dive

## Configuration

### Environment Variables (Optional)

```bash
export YOLO_WORKDIR=/path/to/project    # Working directory
export OLLAMA_HOST=http://localhost:11434  # Ollama server
export YOLO_MODEL=qwen3.5:27b           # LLM model to use
```

### Email Configuration

- **Address**: `yolo@b-haven.org`
- **Inbox path**: `/var/mail/b-haven.org/yolo/new/`
- **Delivery**: Postfix with automatic DKIM signing

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

## Troubleshooting

**Ollama connection failed:**
```bash
ollama serve                # Start server
curl http://localhost:11434/api/generate -d '{"model":"qwen3.5:27b","prompt":"test"}'
```

**Build fails:**
```bash
go mod download
go mod tidy
```

See [Autonomous Operations Guide](DOCS/AUTONOMOUS_OPERATIONS.md#troubleshooting-common-issues) for more solutions.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines on:
- Development workflow
- Code style requirements
- Testing standards  
- Submission process

## License

MIT License - see LICENSE file for details

---

**Note**: YOLO continuously improves its own code and documentation. Changes are automatically committed to git as part of the self-improvement cycle.
