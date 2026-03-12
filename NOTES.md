# YOLO Development Notes

## Important Reminders

### File Paths
- Source code is in the current directory (`.`), NOT `yolo/`
- Example: use `tools_inbox.go`, NOT `yolo/tools_inbox.go`
- Working directory: `/Users/sgriepentrog/src/yolo`

### Restarting YOLO
- **DO USE:** `restart` tool - this rebuilds and restarts properly
- **DO NOT USE:** `os.Exit()`, `process.Kill()` or similar approaches to "rebuild"
- The `restart` tool is the correct way to apply code changes

### Testing with LLM
- Tests that call real LLM APIs can timeout (30s limit)
- For unit tests, consider mocking the LLM response
- Use dependency injection or function overrides for testable code
- Pattern: simulate email data + mock LLM → verify composeResponseToEmail logic

### Key Code Locations
- `tools_inbox.go` - Email handling tools (check_inbox, process_inbox_with_response)
- `tools_ai.go` - LLM integration (generateAIResponse uses Ollama)
- `tools_report.go` - Report generation
- `main.go` - Entry point

## Architecture Notes
- composeResponseToEmail calls generateLLMText which calls Ollama for AI responses
- Email workflow: check_inbox → respond with LLM → delete original
- Maildir format at `/var/mail/b-haven.org/yolo/new/`

## Tools Available
- 21 built-in tools including web_search, reddit, gog (Google Workspace)
- Todo management: add_todo, complete_todo, list_todos
- File operations: read_file, write_file, edit_file, copy_file, move_file
- Subagent support for parallel tasks
- Email handling: check_inbox, process_inbox_with_response, send_email, send_report
