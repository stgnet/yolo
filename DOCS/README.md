# YOLO Documentation Hub

Welcome to the complete documentation for **YOLO (Your Own Living Operator)** - a self-evolving AI agent for autonomous software development.

## 📚 Documentation Index

### Getting Started
- **[README.md](../README.md)** - Project overview, quick start guide, and current status
- **[Quick Start](../README.md#quick-start)** - Step-by-step installation and first run

### Core Concepts
- **[Autonomous Operations](AUTONOMOUS_OPERATIONS.md)** - Complete guide to YOLO's autonomous mode and self-improvement cycle
- **[Architecture](../ARCHITECTURE.md)** - System architecture overview and key components
- **[Security Fixes](../SECURITY_FIXES_SUMMARY.md)** - Summary of security improvements and vulnerability fixes

### Tools & Integration
- **[All Tools Reference](tools.md)** - Complete catalog of YOLO's tools with examples
- **[Reddit Tool](reddit-tool.md)** - Reddit API integration details
- **[Google Workspace (gog)](gog-tool.md)** - Gmail, Calendar, Drive, and more
- **[Tool Verification](tool-verification.md)** - Testing and validation procedures

### Email System
- **[Email Processing](../EMAIL_PROCESSING.md)** - Email architecture, workflow, and configuration
- **[Inbox Management](tools.md#email-tools)** - Reading, responding, and sending emails

### Development & Maintenance
- **[Contributing](../CONTRIBUTING.md)** - How to contribute to YOLO development
- **[Testing Strategy](../ARCHITECTURE.md#testing)** - Testing guidelines and coverage goals
- **[Analysis](../ANALYSIS.md)** - Code analysis and improvement tracking

### Special Topics
- **[Web Search Improvements](../WEB_SEARCH_IMPROVEMENTS.md)** - DuckDuckGo integration enhancements
- **[Todo Items 2026-03-14](../TODO_ITEMS_2026-03-14.md)** - Historical task tracking

## 🔑 Quick Links to Key Information

### File Paths (DO NOT CHANGE WITHOUT RESTART)
- **Working directory**: `/Users/sgriepentrog/src/yolo`
- **Source code**: `/Users/sgriepentrog/src/yolo/yolo`
- **Current model**: `qwen3.5:27b-q4_K_M`

### Email Configuration
- **Address**: `yolo@b-haven.org`
- **Inbox**: `/var/mail/b-haven.org/yolo/new/`
- **Default recipient**: `scott@stg.net` (for reports)
- **Delivery**: Postfix with automatic DKIM signing

### Core Tools by Category

| Category | Tools |
|----------|-------|
| **File Operations** | `read_file`, `write_file`, `edit_file`, `list_files`, `search_files`, `make_dir`, `remove_dir`, `copy_file`, `move_file` |
| **System** | `run_command`, `restart`, `think` |
| **AI/Models** | `list_models`, `switch_model`, `learn`, `implement` |
| **Sub-agents** | `spawn_subagent`, `list_subagents`, `read_subagent_result`, `summarize_subagents` |
| **Web/Social** | `web_search`, `read_webpage`, `reddit` |
| **Google Workspace** | `gog` (Gmail, Calendar, Drive, Docs, Sheets, Slides, Contacts) |
| **Email** | `check_inbox`, `process_inbox_with_response`, `send_email`, `send_report` |
| **Tasks** | `add_todo`, `complete_todo`, `list_todos`, `delete_todo` |

## 🚀 Usage Modes

### Interactive Mode
```bash
./yolo
```
Type commands like:
- `read_file README.md`
- `web_search "go testing best practices"`
- `add_todo "Fix test coverage"`

### Autonomous Mode
```bash
./yolo --autonomous
```
YOLO works independently to:
- Process and respond to emails
- Analyze code quality and fix issues
- Run tests and increase coverage
- Update documentation
- Send progress reports

## 📊 Current Status

See [README.md](../README.md#current-status) for latest:
- Test coverage statistics
- Code quality metrics
- Recent improvements
- Operational health

## 🤝 Contributing

See [CONTRIBUTING.md](../CONTRIBUTING.md) for guidelines on:
- Development workflow
- Code style requirements
- Testing standards
- Submission process

---

**Note**: All documentation files are living documents and may be updated by YOLO itself as part of its self-improvement cycle. Check git history for changes.
