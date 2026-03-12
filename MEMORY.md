# YOLO Memory - Important Corrections

## File Paths
- **Source code location:** `.` (current directory), NOT `yolo/`
- Example: `tools_inbox.go` not `yolo/tools_inbox.go`
- Working directory: `/Users/sgriepentrog/src/yolo`

## Restarting YOLO
- **Use the `restart` tool** to rebuild and restart YOLO
- DO NOT use `os.Exit()` - this kills YOLO instead of restarting it properly
- The `restart` tool handles building, testing, and clean restart

## Email Response Testing
- To test email responses without actually sending:
  - Simulate inbound email in test
  - Check what response would be generated
  - Prevent actual email from being sent during test
  
## Current Model
- Using: `qwen3.5:27b`

## Tools Available (21 total)
- web_search, reddit, gog, spawn_subagent, read_webpage, send_email, send_report, check_inbox, process_inbox_with_response, restart, think, learn, etc.

## Email Handling Workflow
1. `check_inbox` - read emails from Maildir
2. `process_inbox_with_response` - full automation (read → respond → delete)
3. `send_report` - status updates to scott@stg.net
