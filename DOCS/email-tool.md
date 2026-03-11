# Email Tools Documentation

## Overview

YOLO can send emails via SMTP from `yolo@b-haven.org` to notify developers about progress, issues, and discoveries. This enables two-way communication where YOLO reports status and developers can reply with feedback or new directions.

---

## Tools

### `send_email`

Send an email to any recipient via the configured SMTP server.

**Parameters:**
| Name | Type | Required | Description |
|------|------|----------|-------------|
| `to` | string | No | Recipient email address (default: scott@stg.net) |
| `subject` | string | Yes | Email subject |
| `body` | string | Yes | Email body content |

**Example:**
```json
{
  "name": "send_email",
  "arguments": {
    "to": "scott@stg.net",
    "subject": "YOLO Update: Concurrency Bug Fixed",
    "body": "Hi Scott,\n\nI've fixed the LimitedConcurrency deadlock issue by using semaphores correctly. All tests now pass.\n\nBest,\nYOLO"
  }
}
```

### `send_report`

Convenience wrapper for sending progress reports to scott@stg.net.

**Parameters:**
| Name | Type | Required | Description |
|------|------|----------|-------------|
| `subject` | string | No | Report subject (default: "YOLO Progress Report") |
| `body` | string | Yes | Report content |

**Example:**
```json
{
  "name": "send_report",
  "arguments": {
    "subject": "Daily Standup - March 10, 2026",
    "body": "## Completed\n- Fixed LimitedConcurrency deadlock\n- Added email tools for autonomous reporting\n\n## In Progress\n- Researching best practices for Go concurrency patterns\n\n## Blockers\n- Need SMTP password configured"
  }
}
```

---

## Configuration

### Prerequisites

Email sending uses the local Postfix MTA on b-haven.org. No authentication required when running on the server.

The following environment variables can be used to customize SMTP settings:

```bash
# Optional: Override defaults
export EMAIL_SMTP_HOST="localhost"        # Default: localhost
export EMAIL_USERNAME="yolo@b-haven.org"  # Default: yolo@b-haven.org
export EMAIL_FROM_ADDRESS="yolo@b-haven.org"  # Default: yolo@b-haven.org
export EMAIL_FROM_NAME="YOLO"             # Default: YOLO
```

### Default Configuration

| Setting | Default | Environment Variable |
|---------|---------|---------------------|
| SMTP Host | localhost | EMAIL_SMTP_HOST |
| SMTP Port | 25 | - |
| Username | yolo@b-haven.org | EMAIL_USERNAME |
| From Address | yolo@b-haven.org | EMAIL_FROM_ADDRESS |
| From Name | YOLO | EMAIL_FROM_NAME |

### Sending via Postfix

The email system connects to `localhost:25` and uses the local Postfix server which handles:
- DKIM signing automatically
- Message routing and delivery
- TLS encryption for outbound connections

No password authentication is needed when using the local MTA.

---

## Use Cases

### 1. Progress Reports
YOLO can send regular updates about what it's working on:

```json
{
  "name": "send_report",
  "arguments": {
    "subject": "Progress Report - Autonomous Work Session",
    "body": "## Today's Work\n\n- Researched Go concurrency patterns via web_search\n- Fixed 3 test failures in concurrency package\n- Added email tools for better communication\n\n## Next Steps\n\n1. Run full test suite\n2. Document changes\n3. Commit and restart"
  }
}
```

### 2. Alerting on Issues
Notify when something requires human attention:

```json
{
  "name": "send_email",
  "arguments": {
    "to": "scott@stg.net",
    "subject": "⚠️ Blocker: Need Guidance",
    "body": "Hi Scott,\n\nI've completed the concurrency improvements and all tests pass. Would you like me to:\n\n1. Research additional optimization opportunities?\n2. Work on new features?\n3. Document everything and wait for feedback?\n\n- YOLO"
  }
}
```

### 3. Sharing Discoveries
Report interesting findings from research:

```json
{
  "name": "send_report",
  "arguments": {
    "subject": "Discovery: Better Concurrency Pattern Found",
    "body": "I found a more idiomatic way to handle limited concurrency using context.Context with semaphore patterns. This could improve our codebase.\n\nWould you like me to implement this?"
  }
}
```

---

## Example Output

### Success
```
✅ Email sent successfully
   To: scott@stg.net
   From: yolo@b-haven.org
   Subject: YOLO Update
```

### Error (SMTP Failure)
```
Error sending email: dial tcp localhost:25: connection refused
```

---

## Best Practices

1. **Use `send_report` for routine updates** - It's pre-configured for the main recipient
2. **Use `send_email` for specific recipients** - When you need to send to different addresses
3. **Include context in subject lines** - Make it easy to understand what the email is about
4. **Format body with markdown** - Use headers, lists, and code blocks for readability
5. **Send reports after significant work** - After fixing bugs, adding features, or completing research

---

## Two-Way Communication Flow

```
YOLO → send_report → scott@stg.net
                    ↓
              Scott reads email
                    ↓
              Scott replies with feedback/directions
                    ↓
              Email arrives at yolo@b-haven.org inbox (/var/mail/b-haven.org/yolo/new/)
                    ↓
              YOLO checks Maildir for new messages (via gog gmail or custom tool)
                    ↓
              YOLO takes action based on feedback
```

This enables a semi-autonomous workflow where YOLO can report progress and receive guidance without direct console interaction.

### Reading Incoming Email

YOLO can read incoming emails via the `gog` tool (gmail integration) or by polling the Maildir directly:

```bash
# Check for new messages
ls /var/mail/b-haven.org/yolo/new/

# Read a message content
cat /var/mail/b-haven.org/yolo/new/1709932800.v35.b-haven.org.Pid.HASH
```

### Using gog Tool for Gmail

If the email account is also configured with Gmail/Google Workspace:
- `gog "gmail list"` - List recent messages
- `gog "gmail search query"` - Search inbox
- Full integration with Google Workspace tools available
