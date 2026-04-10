# YOLO Agent Architecture

## Overview

YOLO (Your Own Language Operator) is an autonomous AI agent that can perform tasks across multiple domains including file management, email processing, git operations, web automation, and IoT device monitoring. The architecture is built around a modular tool system that enables flexible, extensible functionality.

```
┌─────────────────────────────────────────────────────────────────┐
│                        YOLO Agent                               │
├─────────────────────────────────────────────────────────────────┤
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐      │
│  │   Input      │    │   Context    │    │   Output     │      │
│  │   Handler    │───▶│   Manager    │───▶│   Renderer   │      │
│  │  (terminal,  │    │  (memory,    │    │  (text, TTS) │      │
│  │   email, web)│    │   history)   │    │              │      │
│  └──────────────┘    └──────────────┘    └──────────────┘      │
├─────────────────────────────────────────────────────────────────┤
│                     Tool Execution Engine                       │
├─────────────────────────────────────────────────────────────────┤
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐          │
│  │   File   │ │   Git    │ │   Email  │ │   Web    │          │
│  │  Tools   │ │  Tools   │ │  Tools   │ │  Tools   │          │
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘          │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐          │
│  │ Victron  │ │ Project  │ │ Schedule │ │   MCP    │          │
│ │  (BLE)   │ │  Tools   │ │  Tools   │ │ Connect  │          │
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘          │
├─────────────────────────────────────────────────────────────────┤
│                    External Integrations                        │
│   • Ollama LLM    • Victron Energy • Gmail/Calendar            │
│   • Git           • Web APIs     • Reddit API                 │
└─────────────────────────────────────────────────────────────────┘
```

## Core Components

### 1. Agent Core (`agent.go`)

The central orchestrator that:
- Processes user input and tool output
- Manages conversation context and memory
- Handles autonomous task execution
- Coordinates with external LLM (Ollama)

**Key Responsibilities:**
- Context window management
- Tool selection and execution
- Memory persistence (MEMORY.md + daily logs)
- Autonomous mode operation

### 2. Input Handling (`input.go`, `terminal.go`)

Manages multiple input channels:
- Terminal-based interaction (TUI with line editing)
- Email inbox processing
- Scheduled tasks
- Web callbacks (future)

**Features:**
- Line buffering and history
- Command-line completion
- Real-time email checking
- Cron-based scheduling

### 3. Memory System (`tools_memory.go`)

Multi-layer memory architecture:
- **MEMORY.md**: Curated persistent facts (100 line limit)
- **Daily logs**: Raw observations with timestamps
- **History DB**: Full conversation history for search

**Memory Operations:**
```
user input → memory_log() → daily/YYYY-MM-DD.md
             ↓
        periodic review → memory_promote() → MEMORY.md
             ↓
         curate & distill (AI-assisted)
```

### 4. Tool System (`tools/`)

Modular tool architecture with each tool as a Go package:

| Package | Function | External Dependencies |
|---------|----------|----------------------|
| `tools_file.go` | File operations (CRUD, search, edit) | None |
| `tools_git.go` | Git repository management | git CLI |
| `tools_email.go` | Email sending via sendmail | sendmail/Maildir |
| `tools_inbox.go` | Inbox reading with auto-response | Maildir |
| `tools_gog.go` | Google services (Gmail, Calendar, Drive) | gog CLI |
| `tools_victron.go` | Victron Energy BLE device monitoring | Bluetooth LE |
| `tools_webpage.go` | Web scraping with readability | HTTP client |
| `tools_playwright.go` | Browser automation | Playwright node package |
| `tools_reddit.go` | Reddit API interaction | Reddit public API |
| `tools_todo.go` | Task management (local) | None |
| `tools_project.go` | Project analysis & mapping | AST parsing |
| `tools_memory.go` | Memory CRUD operations | None |
| `tools_search.go` | Web search (SearXNG/DDG) | HTTP client |
| `tools_subagent.go` | Parallel task spawning | Go concurrency |
| `tools_model.go` | Model switching/listing | Ollama API |
| `tools_command.go` | Shell command execution | Shell |

### 5. Victron BLE Integration (`victron/`)

Specialized IoT device monitoring:

```
┌─────────────────┐    ┌──────────────┐    ┌──────────────┐
│   Victron       │    │   BLE        │    │  VE.Direct   │
│ Devices         │───▶│ Backend      │───▶│ Parser       │
│ (SmartSolar,    │    │ (macOS/Bluez)│    │              │
│  SmartShunt)    │    └──────────────┘    └──────────────┘
└─────────────────┘           │                 │
                              ▼                 ▼
                       ┌──────────────┐    ┌──────────────┐
                       │   Mock       │    │  Historical  │
                       │  Backend     │◀───│  Data Store  │
                       │  (testing)   │    │              │
                       └──────────────┘    └──────────────┘
```

**Supported Devices:**
- SmartSolar MPPT charge controllers
- SmartShunt battery monitors
- VE.Direct adapters
- MultiPlus/Quattro inverters (experimental)

### 6. Configuration System (`config.go`, `yoloconfig.go`)

JSON-based configuration:
- API endpoints and timeouts
- Email settings (from/to addresses, inbox path)
- Model selection (Ollama model name)
- Feature toggles (debug mode, auto mode, etc.)

**Location:** `~/.yolo/config.json`

### 7. MCP Connection (`mcp.go`)

Model Context Protocol support:
- Bidirectional JSON-RPC over stdio
- Tool discovery and execution
- Error handling with retry logic

## Data Flow

### Autonomous Task Execution

```
1. Check scheduled tasks (cron.go)
   ↓
2. Process new input (input.go / check_inbox)
   ↓
3. Update context with memory/history
   ↓
4. Call Ollama API for tool selection
   ↓
5. Execute selected tools (tools/)
   ↓
6. Log results to daily context
   ↓
7. Loop or complete task
```

### Email Processing Pipeline

```
check_inbox() → read Maildir new/ directory
     ↓
parse MIME headers and body
     ↓
generate auto-response (AI)
     ↓
send_email() via sendmail
     ↓
mark_read() / delete from inbox
```

### Git Integration Flow

```
git_* tools → construct git CLI command
           → execute with timeout
           → parse structured output
           → return to agent for processing
```

## Security Considerations

- **No network calls without explicit tool selection**
- **File operations restricted to project directory by default**
- **Email sending requires configured sendmail**
- **Git operations limited to repository root**
- **BLE scanning uses standard Bluetooth permissions**
- **API keys not stored in codebase (config.json only)**

## Testing Strategy

```
┌─────────────────────────────────────────┐
│         Unit Tests                      │
│  - Mock dependencies                    │
│  - Test coverage targets: >60%          │
├─────────────────────────────────────────┤
│        Integration Tests                │
│  - Real tool execution                  │
│  - External API mocking where needed    │
├─────────────────────────────────────────┤
│         End-to-End Tests                │
│  - Full agent workflow                  │
│  - Autonomous mode validation           │
└─────────────────────────────────────────┘
```

## Extension Points

### Adding New Tools

1. Create `tools_newtool.go` in `tools/` package
2. Implement tool function with proper error handling
3. Register in MCP connection (if applicable)
4. Add tests to `tools_newtool_test.go`
5. Update documentation in `DOCS/tools.md`

### Custom Memory Prompts

Modify `agent.go` system prompt section for:
- Different AI behaviors
- Custom tool descriptions
- Specialized task patterns

### Platform-Specific Features

Use build tags for platform-specific code:
- `//go:build darwin` for macOS
- `//go:build linux` for Linux
- `//go:build windows` for Windows

## Performance Considerations

- **Memory**: Context window managed to prevent token overflow
- **Concurrency**: Sub-agents for parallel task execution
- **Caching**: Project maps and file trees cached in JSON
- **Network**: Timeout limits on all external API calls
- **BLE**: Polling intervals configurable per device

## Future Enhancements

- [ ] Web UI dashboard for monitoring
- [ ] Vector database for semantic search
- [ ] Plugin system for third-party tools
- [ ] Multi-agent collaboration patterns
- [ ] Cloud backup integration
- [ ] Voice interface improvements

---

*Architecture version: 2.0*  
*Last updated: 2025*
