# YOLO Email System

## Overview
YOLO can send and receive emails using `yolo@b-haven.org`. Outgoing emails are sent via the system's `/usr/sbin/sendmail` command, with Postfix automatically handling DKIM signing. Incoming emails are delivered to a Maildir inbox.

## Tools

### send_email
Send an email from `yolo@b-haven.org`.

**Parameters:**
- `subject` (required): Email subject line
- `body` (required): Email body content
- `to` (optional): Recipient address, defaults to `scott@stg.net`

**Example:**
```
[send_email]
{"subject": "Hello", "body": "World!", "to": "friend@example.com"}
```

### send_report
Send a progress report email to `scott@stg.net`.

**Parameters:**
- `body` (required): Report content
- `subject` (optional): Report subject, defaults to "YOLO Progress Report"

**Example:**
```
[send_report]
{"body": "Task completed successfully"}
```

### check_inbox
Read emails from the Maildir inbox.

**Parameters:**
- `mark_read` (optional, default: false): If true, move processed emails to `cur/` directory

**Example:**
```
[check_inbox]
{"mark_read": true}
```

## Implementation Details

### Sending Emails
- Uses `/usr/sbin/sendmail -f yolo@b-haven.org` command
- RFC 2822 email format with proper headers
- Postfix handles DKIM signing automatically
- No authentication required (uses local MTA)

### Receiving Emails
- Maildir location: `/var/mail/b-haven.org/yolo/`
- New messages: `new/` directory
- Processed messages: Move to `cur/` directory
- Supports multipart MIME emails
- Extracts plain text content automatically

## Testing Email Functionality

### Manual Test
```bash
cd /Users/sgriepentrog/src/yolo
go run test_email_send.go  # Creates and sends a test email
```

### Integration Tests
```bash
YOLO_TEST_EMAIL=1 go test -v -run TestSend ./email/...
```

## DKIM Configuration
DKIM signing is handled automatically by Postfix. The private key and selector are configured in `/etc/postfix/main.cf` via `milter_default_action` and `non_smtpd_milters`.
