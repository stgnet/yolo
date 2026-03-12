# YOLO Memory - Important Reminders

## File Paths
- **Your source code is in `.` (current directory)**, NOT in `yolo/` subdirectory
- Example: Use `tools_inbox.go`, NOT `yolo/tools_inbox.go`
- The working directory IS your source code directory

## Email Handling Rules
1. NO pattern matching or templates
2. Send EVERY email directly to LLM for response generation
3. LLM should give direct, conversational answers
4. Skip email tests (they require actual AI generation which won't work in test environment)

## When Fixing Things
1. Read file from correct path first
2. Make changes
3. `go build` to compile
4. `go test -v ./...` to run tests
5. Commit with `git commit -am "message"`
6. Use `restart` tool (NOT shell commands like kill/restart)
