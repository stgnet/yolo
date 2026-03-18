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

## Documentation

📚 **Complete User Guide**: [DOCS/user-guide.md](DOCS/user-guide.md)

📁 **Documentation Hub**: [DOCS/README.md](DOCS/README.md)

Key guides:
- [Architecture Overview](ARCHITECTURE.md) - System design and structure
- [Email Processing](EMAIL_PROCESSING.md) - Email system deep dive
- [Contributing](CONTRIBUTING.md) - Development guidelines

## Core Capabilities

| Category | Features |
|----------|----------|
| **Self-Improvement** | Auto bug fixes, test coverage, code optimization |
| **Email System** | Read inbox, auto-respond, send reports |
| **Web Integration** | DuckDuckGo search, Reddit, Google Workspace |
| **File Operations** | Read/write/edit files, directory management |
| **Task Management** | Todo list with add/complete/delete |
| **Parallel Work** | Spawn sub-agents for concurrent tasks |

## Current Status

### ✅ Operational Health

- **Tests**: All passing (coverage improving toward 80% target)
- **Code Quality**: No data races, formatted, no vet warnings
- **Git**: Clean working directory, up-to-date with remote

### 🎯 Active Improvements

See [Pending Todos](TODO.md) for current development priorities.

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
ollama serve
curl http://localhost:11434/api/generate -d '{"model":"qwen3.5:27b","prompt":"test"}'
```

**Build fails:**
```bash
go mod download
go mod tidy
```

See [User Guide Troubleshooting](DOCS/user-guide.md#troubleshooting) for more solutions.

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
