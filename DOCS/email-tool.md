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

The `EMAIL_PASSWORD` environment variable must be set with an app-specific password for the b-haven.org email account.

```bash
export EMAIL_PASSWORD="your-app-password-here"
```

### Default Configuration

| Setting | Default | Environment Variable |
|---------|---------|---------------------|
| SMTP Host | b-haven.org | EMAIL_SMTP_HOST |
| SMTP Port | 587 | - |
| Username | yolo@b-haven.org | EMAIL_USERNAME |
| From Address | yolo@b-haven.org | EMAIL_FROM_ADDRESS |
| From Name | YOLO | EMAIL_FROM_NAME |

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
    "subject": "⚠️ Blocker: Missing SMTP Credentials",
    "body": "Hi Scott,\n\nI need the EMAIL_PASSWORD environment variable to be set so I can send emails. Without this, I cannot use the email tools.\n\nPlease configure and let me know when ready.\n\n- YOLO"
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

### Error (Missing Password)
```
Error: EMAIL_PASSWORD not configured. Set the environment variable or configure SMTP credentials.
```

### Error (SMTP Failure)
```
Error sending email: dial tcp b-haven.org:587: i/o timeout
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
              YOLO reads inbox (via gog gmail)
                    ↓
              YOLO takes action based on feedback
```

This enables a semi-autonomous workflow where YOLO can report progress and receive guidance without direct console interaction.
