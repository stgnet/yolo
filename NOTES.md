# YOLO Configuration Notes
## Critical Reminders

### Source Code Location
- **Working Directory**: `/Users/sgriepentrog/src/yolo`
- **Source Code**: Located in `.` (current directory), NOT in `yolo/`
  - Example: Use `tools_inbox.go`, NOT `yolo/tools_inbox.go`

### Restart Procedure
- **DO**: Use the `restart` tool to rebuild and restart YOLO
- **DON'T**: Call `os.Exit()` or try to kill yourself
  - The restart tool handles: go build, go test, git commit, clean restart

### File Operations
All file paths should be relative to: `/Users/sgriepentrog/src/yolo`

## Tool Usage Patterns
- Email handling: `check_inbox` → `process_inbox_with_response` → `send_report`
- Code changes: Use subagent for dev tasks, then build/test/commit/restart
- Web searches: Prefer `web_search` tool (handles DuckDuckGo + Wikipedia fallback)
