# Email System Documentation

## Overview

YOLO can send and receive emails from `yolo@b-haven.org` using the local Postfix MTA with automatic DKIM signing.

## Sending Emails

### Via Sendmail (Default)
The email system uses `/usr/sbin/sendmail` by default:

```go
import "yolo/email"

cfg := email.DefaultConfig()  // Uses sendmail at /usr/sbin/sendmail
client := email.New(cfg)

msg := &email.Message{
    To:      []string{"recipient@example.com"},
    Subject: "Test Email",
    Body:    "Hello from YOLO!",
}

err := client.Send(msg)
```

### Using Tools
```bash
# Send a simple email
send_email subject="Test" body="Hello!" to="scott@stg.net"

# Send a progress report
send_report body="Progress update..."
```

## Receiving Emails

### Maildir Structure
```
/var/mail/b-haven.org/yolo/
├── new/   # Unread messages
├── cur/   # Processed messages  
└── tmp/   # In-progress deliveries (ignore)
```

### Check for New Email
```bash
# Tool-based
check_inbox

# Command-line
ls /var/mail/b-haven.org/yolo/new/
```

### Read and Mark as Read
The `check_inbox` tool automatically:
1. Reads all messages from `/var/mail/b-haven.org/yolo/new/`
2. Parses headers (From, Subject, Date, Body)
3. Moves processed messages to `cur/`

## Configuration

### Default Config (`email.DefaultConfig()`)
- **Sendmail Path**: `/usr/sbin/sendmail`
- **SMTP Host**: `localhost` (fallback only)
- **SMTP Port**: `25` (fallback only)
- **TLS**: Disabled (not needed for local relay)
- **Authentication**: None (local MTA handles this)

### Custom Config
```go
cfg := &email.Config{
    SendmailPath: "/usr/sbin/sendmail",  // Change if needed
    SMTPHost:     "localhost",            // Fallback SMTP
    SMTPPort:     25,                     // Fallback port
}
```

## DKIM Signing

Postfix automatically signs all outgoing emails from `@b-haven.org` using the configured DKIM keys. No additional configuration needed in YOLO's code.

## Troubleshooting

### Email Not Sending
```bash
# Check if sendmail is available
which sendmail

# Test manually
echo "Subject: Test\nFrom: yolo@b-haven.org\nTo: test@example.com\n\nTest body" | /usr/sbin/sendmail -f yolo@b-haven.org test@example.com

# Check Postfix logs
tail -f /var/log/mail.log
```

### Email Not Arriving
```bash
# Check Maildir for received emails
ls -la /var/mail/b-haven.org/yolo/new/

# Check mail logs for delivery status
grep "yolo@b-haven.org" /var/log/mail.log
```

## Testing

Send a test email:
```bash
send_email subject="[TEST] YOLO Email Test" body="This is a test from YOLO" to="scott@stg.net"
```

Verify receipt via Gmail:
```bash
gog "gmail search newer_than:1m from:yolo@b-haven.org"
```

## See Also
- `docs/EMAIL_INSTRUCTIONS.md` - Detailed Maildir instructions
- `email/email.go` - Email client implementation
- `tools_email.go` - Tool implementations
