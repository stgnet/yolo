# YOLO - Your Own Living Operator

**Version**: 1.0 | **Status**: ✅ Production Ready | **Last Updated**: 2026-03-12T13:59:00-04:00

## Overview

YOLO is a self-evolving AI agent for autonomous software development. It operates independently to improve code quality, fix bugs, add tests, and implement new features.

## Core Capabilities

### 🤖 Autonomous Operation
- Self-improving through continuous analysis of own codebase
- Automatic bug detection and fixes
- Test coverage improvement
- Code quality optimization
- **See [Autonomous Operations Guide](DOCS/AUTONOMOUS_OPERATIONS.md) for detailed workflow documentation**

### 📧 Email Integration
**Address**: `yolo@b-haven.org`

Automatic email processing system:
- ✅ Read inbound emails from Maildir (`/var/mail/b-haven.org/yolo/`)
- ✅ Compose intelligent auto-responses using direct LLM generation
- ✅ Delete processed messages after responding (as requested)
- ✅ Smart heuristics to avoid responding to system logs
- ✅ Prioritize messages from Scott (@stg.net)

See `EMAIL_PROCESSING.md` for detailed documentation.

### 🌐 Web & Social Media
- **Web Search**: DuckDuckGo with Wikipedia fallback
- **Reddit Integration**: Search, list subreddits, get threads
- **Google Workspace**: Gmail, Calendar, Drive, Docs, Sheets, Slides, Contacts, Tasks, Chat, Classroom

### 🔧 Development Tools
- File operations (read, write, edit, copy, move, delete)
- Command execution with timeout protection
- Git integration (status, commit, push/pull)
- Build and test automation
- Model management (Ollama integration)
- Sub-agent spawning for parallel tasks

### Current Status (Last Updated: 2026-03-12T21:59:00-04:00)

### ✅ Operational Health
- **Tests**: All passing (63.3% overall coverage)
  - Concurrency: 95.3%
  - Email: 90.0%
  - Main package: 60.4%
- **Code Quality**: 
  - No data races (verified with `-race` flag)
  - All files formatted (`gofmt`)
  - No vet warnings
  - No security vulnerabilities detected
- **Git Status**: Clean working directory, up-to-date with remote

### 📚 Documentation
Comprehensive documentation available:
- [Autonomous Operations Guide](DOCS/AUTONOMOUS_OPERATIONS.md) - Complete guide to YOLO's autonomous mode
- [Email Processing](EMAIL_PROCESSING.md) - Email system architecture and workflow
- [Google Integration](GOOGLE_INTEGRATION.md) - Google Workspace integration guide
- [Architecture](ARCHITECTURE.md) - System architecture overview
- [Testing Summary](TESTING_SUMMARY.md) - Testing strategy and coverage analysis

### 📊 Recent Improvements
1. **Fixed critical data race** in `handoffRemainingTools()` goroutine
2. **Implemented comprehensive email processing** with auto-response workflow
3. **Enhanced test coverage** for email package to 90%
4. **Added mock-based unit tests** for email response generation (avoids LLM calls in CI)
5. **Direct LLM email responses**: All emails now go through direct LLM generation instead of pattern matching
6. **Added comprehensive autonomous operations documentation** including self-improvement cycle and best practices

## Quick Start

### Step-by-Step Setup Guide

Follow these steps to get YOLO up and running:

#### 1. Prerequisites

Ensure you have the following installed:

```bash
# Check Go version (1.21+ required)
go version

# Install Ollama if not already installed
# macOS: brew install ollama
# Linux: curl -fsSL https://ollama.ai/install.sh | sh
# Windows: Download from https://ollama.ai

# Pull a model (qwen3.5:27b recommended)
ollama pull qwen3.5:27b

# Verify Ollama is running
ollama list

# Ensure Git is configured
git config --global user.name "Your Name"
git config --global user.email "your@email.com"
```

#### 2. Installation

```bash
# Clone the repository
git clone https://github.com/your-username/yolo.git
cd yolo

# Download dependencies
go mod download

# Build the binary
go build -o yolo

# Verify build succeeded
./yolo --version
```

#### 3. Configuration (Optional)

YOLO uses sensible defaults, but you can customize:

```bash
# Set working directory (default: current directory)
export YOLO_WORKDIR=/path/to/your/project

# Set Ollama server (default: localhost:11434)
export OLLAMA_HOST=http://localhost:11434

# Choose a different model (default: qwen3.5:27b)
export YOLO_MODEL=qwen3.5:27b
```

#### 4. First Run

Run YOLO in one of these modes:

**Interactive Mode:**
```bash
./yolo
```
You'll see a prompt where you can type commands like:
```
user@yolo$ read_file README.md
user@yolo$ web_search "best practices go testing"
user@yolo$ add_todo "Fix test coverage"
```

**Autonomous Mode:**
```bash
./yolo --autonomous
```
YOLO will work independently to:
- Check for emails and respond to them
- Analyze code quality and fix issues
- Run tests and increase coverage
- Update documentation
- Send progress reports

#### 5. Verify Setup

Run these commands to verify everything works:

```bash
# Run all tests
go test -v ./...

# Check for race conditions
go test -race ./...

# Verify code formatting
gofmt -l .

# Check dependencies
go mod tidy
```

### Troubleshooting

**Issue**: Ollama connection failed
```bash
# Ensure Ollama is running
ollama serve

# Check the model is available
ollama list

# Test the connection directly
curl http://localhost:11434/api/generate -d '{"model":"qwen3.5:27b","prompt":"test"}'
```

**Issue**: Build fails with missing dependencies
```bash
go mod download
go mod tidy
```

**Issue**: Tests fail due to timeout
- Increase test timeout: `go test -timeout 5m ./...`
- Mock LLM calls in tests using function variable injection (see Testing section)

### Usage Examples

**File Operations:**
```
user@yolo$ read_file tools.go
user@yolo$ write_file hello.txt "Hello, World!"
user@yolo$ edit_file README.md "old text" "new text"
```

**Web & Research:**
```
user@yolo$ web_search "Go performance best practices"
user@yolo$ reddit search "Golang testing tips"
user@yolo$ learn  # Autonomous research and self-improvement
```

**Task Management:**
```
user@yolo$ add_todo "Implement feature X"
user@yolo$ list_todos
user@yolo$ complete_todo "Implement feature X"
```

**Email (if configured):**
```
user@yolo$ check_inbox
user@yolo$ process_inbox_with_response  # Full automation: read → respond → delete
```

## Quick Reference

### Commands
```bash
# Build
go build -o yolo

# Run tests
go test -v ./...
go test -race ./...

# Check formatting
gofmt -l .

# Check dependencies
go mod tidy

# Git operations
git status
git log --oneline -5
```

### Configuration
- Working directory: `/Users/sgriepentrog/src/yolo`
- Email address: `yolo@b-haven.org`
- Current model: `qwen3.5:27b`
- Ollama server: localhost:11434

### Testing with Mocks
The codebase uses function variable injection for testable LLM integration:
```go
// In tools_inbox.go - replace this to mock
var llmResponseGenerator = func(prompt string) string {
    return generateLLMText(prompt)
}

// In tests - override it
llmResponseGenerator = func(prompt string) string {
    return "MOCKED response"
}
```

## Security Checklist

✅ No SQL injection risks  
✅ No eval/exec misuse  
✅ No unsafe environment variable manipulation  
✅ Proper file path validation  
✅ Command execution timeout protection  
✅ Rate limiting for external API calls  

## Next Priorities (Autonomous)

YOLO will automatically focus on:
1. Increasing main package test coverage to 75%+
2. Performance optimization of hot paths
3. Adding caching for repeated operations
4. Enhancing error handling and recovery
5. Expanding documentation with examples

## License

MIT License - see LICENSE file for details.

---

**YOLO is actively monitoring and improving itself.**  
Last autonomous check: Just now ✅
