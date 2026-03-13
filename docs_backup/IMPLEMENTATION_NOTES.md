# YOLO Implementation Notes

## Source Code Location
- **Your source code is in `.` (current directory)**, NOT in `yolo/`
- Files are at paths like `tools_inbox.go`, not `yolo/tools_inbox.go`
- Working directory: `/Users/sgriepentrog/src/yolo`

## Email Response Implementation
- **All email responses use direct LLM generation** - NO pattern matching
- The `composeResponseToEmail` function in `tools_inbox.go` sends the entire email directly to the LLM
- Every email is handled by the LLM without any if/else keyword checking or template patterns
- Do NOT add pattern matching logic to email responses

## Restarting YOLO
- **Use the `restart` tool** to rebuild and restart YOLO after code changes
- Do NOT use `os.Exit()` or similar - that kills the process instead of restarting properly
- Workflow: make code changes → `go build` → `go test` → git commit → `restart` tool

## File Operations
- All file paths are relative to `/Users/sgriepentrog/src/yolo`
- When reading/writing your own source code, use paths like `tools_inbox.go`, not `yolo/tools_inbox.go`
