# YOLO Agent - Critical Knowledge Base

This file contains essential information that YOLO must remember to function correctly. All other knowledge-related markdown files have been consolidated into this single source of truth.

---

## 📁 Source Code Location (CRITICAL)

**Working directory:** `/Users/sgriepentrog/src/yolo` (this is `.`)

**Source code is in the current directory (`.`), NOT in a `yolo/` subdirectory!**

### Correct Paths
- ✅ `tools_inbox.go`, `main.go`, `prompt/format.go`
- ✅ Relative to `/Users/sgriepentrog/src/yolo`

### Incorrect Paths (DO NOT USE)
- ❌ `yolo/tools_inbox.go`, `yolo/main.go`
- ❌ Absolute paths like `/Users/sgriepentrog/src/yolo/tools_inbox.go`

### Key File Locations
| File | Purpose |
|------|---------|
| `main.go` | Entry point and main loop |
| `tools_*.go` | Tool implementations (21 built-in tools) |
| `tools_*_test.go` | Unit tests for tools |
| `prompt/` directory | Prompt templates and formatting |
| `yolo/concurrency/` | Concurrency primitives and utilities |
| `yolo/email/` | Email package utilities |

---

## 🔄 Restarting YOLO (CRITICAL)

### DO USE
**Use the `restart` tool** to rebuild and restart YOLO after code changes.

```go
restart()  // Rebuilds and restarts properly
```

The `restart` tool:
- Rebuilds the binary with `go build`
- Runs tests if needed
- Restarts the agent cleanly

### DO NOT USE
**NEVER use these approaches:**
- ❌ `os.Exit()` - kills the agent instead of restarting
- ❌ `process.Kill()` - breaks things
- ❌ Manual restart commands

The proper workflow after code changes:
1. Edit source code
2. Run `go build` to verify compilation
3. Run `go test -v ./...` to ensure tests pass
4. Commit changes to git
5. Call `restart()` tool

---

## 📧 Email Handling System

### Approach (User Preference)
**Direct LLM generation for ALL email responses.**

- ✅ Every email goes directly to the LLM via `generateLLMText()` or `generateAIResponse()`
- ❌ NO pattern matching or template fallbacks
- ❌ No hardcoded response templates
- ✅ Let the LLM generate natural, conversational responses based on: sender, subject, body content

### Email Workflow
1. **check_inbox** - Read emails from Maildir at `/var/mail/b-haven.org/yolo/new/`
2. **process_inbox_with_response** - Full automation: read → respond with LLM → delete original
3. **send_email** - Send individual emails via yolo@b-haven.org (Postfix handles DKIM signing)
4. **send_report** - Send progress reports to scott@stg.net

### Email Testing Guidelines
To test email responses without actually sending:
1. Simulate inbound email data in tests
2. Verify what response would be generated
3. Prevent actual email from being sent (use flags or mocks)

Test pattern: `simulate_email + mock_LLM → verify_composeResponseToEmail`

### Key Functions
- `composeResponseToEmail()` in `tools_inbox.go` - Generates AI-powered responses
- `generateLLMText()` - Calls Ollama for LLM generation
- `generateAIResponse()` in `tools_ai.go` - Alternative LLM interface

---

## 🧪 Testing & Development Guidelines

### Test Timeout Handling
Tests that call real LLM APIs (Ollama) can timeout if the model is slow:
- Default timeout limit: 30 seconds
- Use `t.Skip()` to gracefully skip tests when dependencies aren't available
- Keep test timeouts short (< 30s) to avoid hanging

### Mocking for Unit Tests
For reliable unit tests:
1. Extract response generation logic into testable functions
2. Use dependency injection or function overrides
3. Mock LLM responses instead of calling real API
4. Test with `YOLO_TEST_EMAIL=1` flag for integration tests that send real emails

### Code Structure
- **Tool implementations:** `tools_*.go` files (21 tools total)
- **Tool tests:** `tools_*_test.go` files
- **All in root directory:** `.` not subdirectories
- **Concurrency utilities:** `yolo/concurrency/` package
- **Email utilities:** `yolo/email/` package

---

## 🛠️ Available Tools (21 Total)

### Communication Tools
- `web_search` - DuckDuckGo search with Wikipedia fallback
- `reddit` - Search, subreddit listing, thread details
- `gog` - Google Workspace integration (Gmail, Calendar, Drive, Docs, Sheets, Slides, Contacts, Tasks)
- `send_email` - Send email via yolo@b-haven.org
- `send_report` - Send progress report to scott@stg.net
- `check_inbox` - Read emails from Maildir inbox
- `process_inbox_with_response` - Full email automation (read → respond → delete)

### File Operations
- `read_file` - Read file contents with offset/limit support
- `write_file` - Create or overwrite files
- `edit_file` - Replace text in files
- `copy_file` - Copy files (creates dest directory if needed)
- `move_file` - Move files (creates dest directory if needed)

### Development Tools
- `spawn_subagent` - Spawn background sub-agent for parallel tasks
- `list_subagents` - List all active sub-agents with status
- `read_subagent_result` - Get results from completed sub-agents
- `summarize_subagents` - Get summary statistics
- `learn` - Autonomous research for self-improvement opportunities

### Agent Control
- `restart` - Rebuild and restart YOLO after code changes
- `think` - Record internal reasoning without taking action
- `switch_model` - Switch to different Ollama model
- `list_models` - List available Ollama models

### Todo Management
- `add_todo` - Add new todo item
- `complete_todo` - Mark todo as completed
- `list_todos` - List all todos (pending and completed)

---

## 🤖 Current Configuration

**Current Model:** `qwen3.5:27b` (Ollama)

### Performance Metrics
- **Web Search Average Time:** 5.14s (with caching and Wikipedia fallback)
- **Test Suite Duration:** ~2 seconds
- **Memory Usage:** <50MB idle
- **Concurrent Tests:** All pass with `-race` flag enabled

---

## 📊 Test Coverage Status

### High Coverage (>90%)
- `yolo/concurrency` - 95.3%
- `yolo/email` - 90.0%
- Most individual tool implementations

### Moderate Coverage (48-80%)
- Agent core logic and context-aware parsing
- Web search with caching
- History management
- Buffer UI rendering (~90%)

### Low Coverage Areas (<50%)
- Interactive terminal commands (`restart`, `switchModel`)
- User input handling (integration-tested via manual testing)
- First-run setup flows
- Command handlers for admin tasks

> **Note:** Low coverage in I/O and interactive code is expected and acceptable. These are tested through integration/manual testing.

---

## 🔄 Email Response Testing Strategy

The proper test pattern for email responses:

```go
func TestEmailResponse(t *testing.T) {
    // 1. Simulate inbound email data
    email := Email{From: "user@example.com", Subject: "...", Body: "..."}
    
    // 2. Mock LLM response (don't call real Ollama API)
    mockLLMResponse := "Hello user, ..."
    
    // 3. Verify composeResponseToEmail generates correct prompt
    
    // 4. Prevent actual email from being sent during test
    
    // 5. Assert expected behavior
}
```

Key principle: **Tests should verify the logic, not require external services.**

---

## 📝 Documentation Files Reference

### Core Documentation (Essential)
- **README.md** - Main project documentation with overview, quick start, installation
- **SYSTEM_PROMPT.md** - System prompt defining YOLO's behavior and autonomy rules
- **ARCHITECTURE.md** - High-level system architecture overview
- **CHANGELOG.md** - Version history and release notes
- **KNOWLEDGE_BASE.md** - This file (consolidated critical information)

### Feature Documentation
- **LEARNING.md** - Autonomous learning system documentation
- **EMAIL_PROCESSING.md** - Email handling workflow details
- **GOOGLE_INTEGRATION.md** - Google Workspace integration guide
- **CONTRIBUTING.md** - Contribution guidelines

### Status & Reports (Generated/Temporary)
- **STATUS_REPORT.md** - Current system status (regenerated periodically)
- **IMPROVEMENTS_SUMMARY.md** - Summary of recent improvements
- **TESTING_SUMMARY.md** - Test coverage and results summary
- **TODO.md** - Todo list tracking

### Legacy Files (Candidates for Removal)
These files have been consolidated into KNOWLEDGE_BASE.md:
- ~~MEMO.md~~ → Merged into KNOWLEDGE_BASE.md
- ~~MEMORY.md~~ → Merged into KNOWLEDGE_BASE.md
- ~~NOTES.md~~ → Merged into KNOWLEDGE_BASE.md
- ~~REMEMBER.md~~ → Merged into KNOWLEDGE_BASE.md

---

*This knowledge base is maintained by YOLO for autonomous operation.*
*Last updated: 2026-03-12*
