# YOLO Usage Examples

This document provides practical examples of using YOLO for various tasks.

## Basic Usage

### Starting a Conversation

```bash
yolo
```

Enter your message and press Enter. YOLO will process it and respond with tool usage if needed.

### Commands Reference

| Command | Description | Example |
|---------|-------------|---------|
| `/help` | Show available commands | `/help` |
| `/auto [on\|off]` | Toggle autonomous mode | `/auto on` |
| `/model` | Show current model | `/model` |
| `/models` | List all models | `/models` |
| `/switch <name>` | Change model | `/switch llama3.1` |
| `/history` | Show message count | `/history` |
| `/clear` | Clear conversation | `/clear` |
| `/status` | Agent status | `/status` |
| `/debug [on\|off]` | Toggle debug mode | `/debug on` |
| `/think [on\|off]` | Show thinking blocks | `/think off` |
| `/todo` | Show todos | `/todo` |
| `/todo <text>` | Add todo | `/todo fix the bug` |

## File Operations

### Reading Files

```
Please read the contents of main.go and summarize it.
```

YOLO will use the `read_file` tool to fetch and display the file.

For large files, you can specify line ranges:

```
Read lines 50-100 from agent.go
```

### Writing Files

```
Create a new file config.json with {"debug": true}
```

YOLO will create or overwrite the file with the specified content.

### Editing Files

**Replace specific text:**
```
In main.go, replace "TODO: fix this" with "// Fixed on April 10"
```

**Replace all occurrences:**
```
Replace all instances of "log.Printf" with "logger.Info" in *.go files
```

## Git Operations

### Check Repository Status

```
What's the current git status?
```

YOLO will check for modified, staged, and untracked files.

### View Changes

```
Show me what changed in agent.go since last commit
```

### Commit Workflow

```
1. Stage all changes
2. Commit with message "Fix critical bug in parsing"
```

YOLO will run `git add .` and `git commit -m "Fix critical bug in parsing"`.

## Web Research

### Search the Web

```
Search for "Go concurrency patterns best practices 2024"
```

YOLO will search using SearXNG or DuckDuckGo and return top results.

### Read Articles

```
Read this article: https://example.com/go-tips
```

YOLO will fetch and extract readable content from the page.

## Email Management

### Check Inbox

```
Check my inbox for unread messages
```

YOLO will read emails from the configured Maildir.

### Send Email

```
Send an email to john@example.com with subject "Project Update" and body attached as a file
```

## Task Automation

### Adding Tasks

```
Add a reminder to review pull requests tomorrow at 10am
```

YOLO creates a todo item for tracking.

### Managing Subagents

YOLO can spawn parallel subagents for complex tasks:

```
Spawn subagent to refactor utils.go and another to write tests for agent.go
```

This runs two parallel operations that YOLO can coordinate.

## Configuration

### Config Location

Configuration is stored in `config.json` (location depends on your platform):
- Linux/macOS: `$HOME/.config/yolo/config.json`
- Windows: `%APPDATA%\yolo\config.json`

### Key Settings

```json
{
  "model": "llama3.1",           // Default AI model
  "email_from": "you@example.com",
  "email_to": "recipient@example.com", 
  "inbox_path": "/var/mail/inbox",
  "tts_enabled": false,
  "debug_mode": false,
  "terminal_mode": true         // Split-screen UI mode
}
```

### Changing Model

```
Switch to the mistral model for faster responses
/switch mistral
```

## Advanced Features

### Memory System

YOLO has a tiered memory system:
- **MEMORY.md**: Durable facts, preferences, conventions (max 100 lines)
- **Daily logs**: Raw observations and session notes
- **History DB**: Full conversation history with compaction

Query memory:
```
Show me what I saved about project structure in memory
/search-memory project structure
```

### Voice Input/Output

Enable voice mode (requires STT/TTS setup):
```
/listen        # Record one-shot voice input
/tts on        # Enable text-to-speech output
/voice female  # Select a TTS voice
```

### Autonomous Mode

Let YOLO work without user input:
```
/auto on
Please refactor the codebase to improve performance and add tests.
```

YOLO will continue working until tasks are complete, then report back.

## Troubleshooting

### Common Issues

**Issue**: "NEEDS RESTART" message
**Solution**: Use the `/restart` command or press Ctrl+C and restart

**Issue**: Model not responding
**Solution**: Check Ollama is running: `ollama serve`

**Issue**: Tool execution fails
**Solution**: Enable debug mode with `/debug on` to see full error details

**Issue**: Slow performance
**Solution**: Use a smaller model with `/switch llama3.2` or reduce context with `/clear`

## Performance Tips

1. **Use appropriate models**: Smaller models are faster for simple tasks
2. **Clear history when context grows too large**: `/clear` to reset
3. **Enable terminal mode for better UX**: `/terminal on`
4. **Disable TTS if not needed**: Save resources with `/tts off`
5. **Use debug mode only when needed**: Debug output adds overhead

## Best Practices

### For Code Tasks

1. Always read files before modifying them
2. Use `edit_file` for small changes, `patch_file` for larger ones
3. Test changes with `go test` after modifications
4. Commit changes with descriptive messages

### For Research Tasks

1. Start broad with web search
2. Narrow down with specific article reads
3. Synthesize findings before taking action
4. Save key insights to memory if reusable

### For Email Tasks

1. Process emails systematically with `/inbox-process`
2. Review auto-responses before sending in production
3. Keep inbox configured with correct paths

## Integration Examples

### CI/CD Pipeline

```yaml
# GitHub Actions example
- name: Run YOLO autonomous task
  run: |
    echo "Please ensure all tests pass" > prompt.txt
    yolo --auto < prompt.txt
    cat response.log
```

### Scripting with YOLO

```bash
#!/bin/bash
# Ask YOLO to generate documentation
echo "Generate README.md for the current repository" | yolo
```

## Additional Resources

- [ARCHITECTURE.md](./ARCHITECTURE.md) - Technical architecture details
- [README.md](./README.md) - Installation and setup guide
- [TODO.md](./TODO.md) - Feature tracking and completion status
