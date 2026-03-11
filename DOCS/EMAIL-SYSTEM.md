# Email System Documentation

## Overview
YOLO has full email capabilities via `yolo@b-haven.org` using Postfix with DKIM signing.

## Features

### 1. Send Emails (send_email tool)
Send emails from `yolo@b-haven.org` to any recipient.

**Parameters:**
- `subject` (required): Email subject line
- `body` (required): Email body content
- `to` (optional): Recipient email (defaults to scott@stg.net)

**Example:**
```
[tool activity]
[send_email => {"subject": "Test", "body": "Hello", "to": "user@example.com"}]
```

### 2. Send Progress Reports (send_report tool)
Quickly send progress reports to scott@stg.net.

**Parameters:**
- `body` (required): Report content
- `subject` (optional): Report subject (defaults to "YOLO Progress Report")

**Example:**
```
[tool activity]
[send_report => {"body": "Task completed successfully"}]
```

### 3. Check Inbox (check_inbox tool)
Read emails from the Maildir inbox at `/var/mail/b-haven.org/yolo/new/`

**Parameters:**
- `mark_read` (optional): If true, move processed emails to cur/ directory

**Example:**
```
[tool activity]
[check_inbox => {"mark_read": false}]
```

## Technical Details

### DKIM Configuration
- Domain: b-haven.org
- Selector: mail._domainkey.b-haven.org
- Private key: Located in /etc/postfix/dkim/ (auto-loaded by OpenDKIM)
- Signing: Automatic for all outgoing mail via Postfix

### Maildir Structure
```
/var/mail/b-haven.org/yolo/
├── new/        # New unread emails
├── cur/        # Read/processed emails
└── tmp/        # Temporary files (during delivery)
```

### Email Format
- Supports RFC 2822 format
- Handles multipart MIME messages
- Extracts text/plain content automatically

## Testing

Run email tests:
```bash
go test -v -run "Email|Inbox"
```

Enable integration tests:
```bash
YOLO_TEST_EMAIL=1 go test -v -run TestSend_Integration
```

## Troubleshooting

### Emails not sending
- Check Postfix is running: `sudo systemctl status postfix`
- Verify DKIM service: `sudo systemctl status opendkim`
- Check mail logs: `tail -f /var/log/mail.log`

### Emails going to spam
- Ensure SPF records are configured for b-haven.org
- DKIM signing should be automatic via Postfix
- Domain reputation takes time to build

### Inbox not working
- Verify Maildir exists: `ls -la /var/mail/b-haven.org/yolo/`
- Check postfix delivery in `/var/log/mail.log`