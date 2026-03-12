# YOLO Self-Knowledge Memo

## Critical Facts to Remember

### 1. Source Code Location
- **Source code is in `.` (current directory), NOT in `yolo/`**
- Correct paths: `tools_inbox.go`, `main.go`, `prompt/format.go`
- Incorrect paths: `yolo/tools_inbox.go`, `yolo/main.go`
- Working directory is already `/Users/sgriepentrog/src/yolo`

### 2. Restarting YOLO
- **Use the `restart` tool** - do NOT use `os.Exit()` or kill commands
- The restart tool properly rebuilds and restarts YOLO
- Example: Call `restart()` function when code changes need to take effect

### 3. Email Handling Approach
- User prefers **direct LLM generation for ALL email responses**
- Do NOT use hardcoded pattern matching
- Use `generateAIResponse` function from tools_ai.go
- Make composeResponseToEmail call the AI model directly with full email context

### 4. Test Strategy
- Tests should simulate inbound emails and check what response would be generated
- Tests must prevent actual email sending while still verifying output
- Consider extracting response generation logic to make it testable

## File Locations
- Email handling: `tools_inbox.go`
- AI generation: `tools_ai.go` (has generateAIResponse function)
- Main entry: `main.go`
- Prompt templates: `prompt/` directory
