# Key Reminders for YOLO Agent

## Source Code Location
- **Your source code is in `.` (the current directory)**, NOT in `yolo/`
- File paths should be relative to `/Users/sgriepentrog/src/yolo`
- Example: Read `tools_inbox.go`, NOT `yolo/tools_inbox.go`

## Restarting YOLO
- **Use the `restart` tool** to rebuild and restart YOLO after code changes
- **DO NOT use `os.Exit()` or kill yourself** - this breaks things
- Workflow: edit code → `go build` → `go test` → commit → `restart` tool

## Email Response System
- The `composeResponseToEmail` function in `tools_inbox.go` sends ALL emails directly to the LLM
- **NO pattern matching** - every email goes to the LLM for a custom response
- The LLM generates responses based on: sender, subject, and body content
- No templates or placeholders should be used

## Testing Guidelines
- Email tests should NOT send real emails unless gated behind `YOLO_TEST_EMAIL=1`
- Tests that call LLM (like `composeResponseToEmail`) may timeout if Ollama is slow
- Use `t.Skip()` to gracefully skip tests when dependencies aren't available
- Keep test timeouts short (under 30s) to avoid hanging

## Code Structure
- Tool implementations: `tools_*.go` files
- Tool tests: `tools_*_test.go` files  
- All in the root directory (`.`)
