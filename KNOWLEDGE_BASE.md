# YOLO Agent - Critical Information

## Where I Am
- **Working directory:** `/Users/sgriepentrog/src/yolo` (this is `.`)
- **My source code:** In `.` directory, NOT in `yolo/` subdirectory
  - Example: `tools_inbox.go` NOT `yolo/tools_inbox.go`
  - Example: `main.go` NOT `yolo/main.go`

## How to Restart
- Use the **`restart()` tool** to rebuild and restart
- DO NOT call `os.Exit()` or similar - this kills the agent improperly

## Email Handling Rules
1. All email responses go DIRECTLY to LLM via `generateLLMText()`
2. NO pattern matching or template fallbacks
3. Hand every email body/subject/sender directly to the prompt
4. Let the LLM generate natural, conversational responses

## File Paths Reference
- Email inbox: `/var/mail/b-haven.org/yolo/new/`
- My files: `tools_inbox.go`, `tools_email.go`, `main.go`, etc. (all in `.`)
