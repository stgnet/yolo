# YOLO Tool Documentation

## Overview

YOLO (Your Own Living Operator) has these powerful tools for autonomous software development and web interaction:

---

## 🗂️ File Operations

| Tool | Description |
|------|-------------|
| `read_file` | Read file contents with offset/limit for large files |
| `write_file` | Create or overwrite files |
| `edit_file` | Replace text in files (find/replace) |
| `list_files` | List files matching glob patterns |
| `search_files` | Search file contents using regex |
| `make_dir` | Create directories recursively |
| `remove_dir` | Remove directories and contents |
| `copy_file` | Copy files with auto-directory creation |
| `move_file` | Move files with auto-directory creation |

---

## ⚙️ System & Execution

| Tool | Description |
|------|-------------|
| `run_command` | Execute shell commands (30s timeout) |
| `spawn_subagent` | Run parallel background agents |
| `list_subagents` | List all active sub-agents |
| `read_subagent_result` | Get result from specific sub-agent |
| `summarize_subagents` | Get completion statistics |

---

## 🤖 AI & Model Management

| Tool | Description |
|------|-------------|
| `list_models` | List available Ollama models |
| `switch_model` | Change the active LLM model |
| `think` | Record internal reasoning (no action) |
| `restart` | Rebuild and restart YOLO |

---

## 🌐 Web & External APIs

| Tool | Description | Docs |
|------|-------------|---|
| `web_search` | Search DuckDuckGo Instant Answer API with Wikipedia fallback | [below](#web_search-tool) |
| `reddit` | Search Reddit, list subreddit posts, get threads | [reddit-tool.md](./reddit-tool.md) |
| `gog` | Google Workspace: Gmail, Calendar, Drive, Contacts, Sheets, Docs | [gog-tool.md](./gog-tool.md) |
| `send_email` | Send emails via sendmail from yolo@b-haven.org (DKIM signed by Postfix) | [below](#email-tools) |
| `send_report` | Send progress reports to scott@stg.net | [below](#email-tools) |
| `check_inbox` | Read incoming emails from Maildir inbox | [below](#email-tools) |

---

## 🔧 Key Tools Deep Dive

### web_search Tool
Search the internet using DuckDuckGo's Instant Answer API with Wikipedia fallback for comprehensive results.

```json
{
  "name": "web_search",
  "arguments": {
    "query": "go programming language concurrency",
    "count": 5
  }
}
```

**How it works:**
1. Queries DuckDuckGo's Instant Answer API for direct answers and summaries
2. Falls back to Wikipedia search if DuckDuckGo returns no results
3. Combines both sources when available for richer information

**Use Cases:**
- Learn about new tools/technologies
- Find documentation and quick references  
- Research problems and solutions
- Stay updated on trends and best practices

**Example Output:**
```
Wikipedia results for "golang concurrency patterns":

1. **[Go (programming language)](https://en.wikipedia.org/wiki/Go_(programming_language))**
   Go is a programming language developed at Google...
   
2. **[Goroutine](https://en.wikipedia.org/wiki/Goroutine)**
   A goroutine is a lightweight thread of execution...
```

---

### reddit Tool
Access Reddit's public API without authentication.

```json
{
  "name": "reddit",
  "arguments": {
    "action": "search",
    "query": "gog openclaw",
    "limit": 10
  }
}
```

**Actions:**
- `search` - Query Reddit globally
- `subreddit` - List posts from r/{name}
- `thread` - Get post + comments by ID

---

### gog Tool (Google Workspace)
Full Google Workspace integration via OAuth.

```json
{
  "name": "gog",
  "arguments": {
    "command": "gmail search 'inbox:unread newer_than:1d' --max 5"
  }
}
```

**Capabilities:**
- 📧 Gmail: Search, send, drafts, replies
- 📅 Calendar: Events CRUD, colors
- 📁 Drive: List, search files
- 👥 Contacts: List and search
- 📊 Sheets: Read/write cells
- 📝 Docs/Slides: Export and view

**Quick Commands:**
```bash
gog gmail search 'from:boss newer_than:2d' --max 10
gog calendar events primary --from 2026-03-10T00:00Z --to 2026-03-17T23:59Z
gog drive ls --max 20
gog contacts list --max 30
```

---

## 📧 Email Tools (send_email, send_report, check_inbox)

Full email system for yolo@b-haven.org with DKIM signing via Postfix.

### send_email - Send an Email
```json
{
  "name": "send_email",
  "arguments": {
    "to": "recipient@example.com",
    "subject": "Test Email",
    "body": "Hello from YOLO!"
  }
}
```
**Parameters:**
- `to` (optional): Recipient email (default: scott@stg.net)
- `subject` (required): Email subject line
- `body` (required): Email content

**How it works:** Uses `/usr/sbin/sendmail` for local delivery. Postfix automatically signs with DKIM. No SMTP authentication required!

### send_report - Send Progress Report
Convenience wrapper for reporting to scott@stg.net.
```json
{
  "name": "send_report", 
  "arguments": {
    "subject": "Weekly Update",
    "body": "Completed tasks: 1, 2, 3\n\nNext steps: A, B, C"
  }
}
```

### check_inbox - Read Incoming Emails
```json
{
  "name": "check_inbox",
  "arguments": {
    "mark_read": true
  }
}
```
**Parameters:**
- `mark_read` (optional): If true, move processed emails from `new/` to `cur/`

**How it works:** Reads Maildir at `/var/mail/b-haven.org/yolo/new/`. Parses RFC 2822 format with MIME support for multipart messages.

**Example Output:**
```
📬 Found 2 new email(s)
   Moved 2 email(s) to cur/ (marked as read)

--- Email 1 of 2 ---
From: user@example.com
Subject: Test Message
Date: Wed, 10 Mar 2026 22:36:27 +0000
Content-Type: text/plain; charset=utf-8

Body:
This is the email content...
```

For more details, see: [EMAIL-SYSTEM.md](./EMAIL-SYSTEM.md) and [EMAIL-OPERATIONS.md](./EMAIL-OPERATIONS.md)

---

## 💡 Best Practices

1. **Use web_search** before implementing new features to research best practices
2. **Check Reddit** for community discussions on tools/technologies  
3. **Leverage gog** for Gmail/Calendar automation tasks
4. **Use email tools** (send_email/send_report/check_inbox) for communication with scott@stg.net
5. **Spawn subagents** for parallel independent tasks
6. **Use think tool** for complex planning before action

---

## 📚 External Resources

- **GOG Docs**: https://gogcli.sh
- **GOG Source**: https://github.com/danielmiessler/gog  
- **Reddit API**: https://www.reddit.com/dev/api/
- **Wikipedia API**: https://www.mediawiki.org/wiki/API:Main_page

---

## 🔄 Self-Improvement Cycle

YOLO should:
1. Research new tools via web_search
2. Read documentation and learn usage patterns
3. Implement or integrate useful capabilities
4. Document additions for future use
5. Repeat continuously
