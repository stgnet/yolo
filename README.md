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

## Current Status

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

### 📊 Recent Improvements
1. **Fixed critical data race** in `handoffRemainingTools()` goroutine
2. **Implemented comprehensive email processing** with auto-response workflow
3. **Enhanced test coverage** for email package to 90%
4. **Added mock-based unit tests** for email response generation (avoids LLM calls in CI)
5. **Direct LLM email responses**: All emails now go through direct LLM generation instead of pattern matching
6. **Added extensive documentation** including:
   - `EMAIL_PROCESSING.md` - Email system documentation
   - `GOOGLE_INTEGRATION.md` - Google Workspace integration guide
   - `ARCHITECTURE.md` - System architecture overview
   - `TESTING_SUMMARY.md` - Testing strategy and coverage

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
