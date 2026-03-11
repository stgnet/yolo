# Email Integration Guide

## Overview

YOLO can now send emails autonomously from `yolo@b-haven.org` using the server's local SMTP relay (postfix/sendmail). No authentication is required - emails are sent directly through the configured MTA.

## Configuration

No configuration needed! The email system uses:
- **Sendmail binary**: `/usr/sbin/sendmail` (default)
- **Fallback SMTP**: `localhost:25` (optional)
- **From address**: `yolo@b-haven.org` (hardcoded default)

## Tools Available

### send_email

Send custom emails to any recipient.

```json
{
  "name": "send_email",
  "arguments": {
    "to": "recipient@example.com",
    "subject": "Email subject line",
    "body": "Email body content"
  }
}
```

**Parameters:**
- `to` (optional): Recipient email address. Defaults to `scott@stg.net` if not specified
- `subject` (required): Email subject line
- `body` (required): Email body text (plain text)

### send_report

Send progress reports directly to scott@stg.net.

```json
{
  "name": "send_report", 
  "arguments": {
    "subject": "Progress Report: Task Update",
    "body": "Detailed progress report content..."
  }
}
```

**Parameters:**
- `subject` (optional): Report subject. Defaults to "YOLO Progress Report"
- `body` (required): Report content

## Use Cases

1. **Progress Reporting**: Send status updates during long-running tasks
2. **Task Notifications**: Alert about completed or failed operations  
3. **Self-Diagnostic**: Notify about system issues or improvements discovered
4. **Research Results**: Share findings from autonomous learning sessions

## Testing

Run the email integration tests:

```bash
go test -v ./email/...
go test -v -run TestSendEmailIntegration .
go test -v -run TestSendReportIntegration .
```

## How It Works

1. YOLO creates an RFC 5322 compliant email message
2. Message is piped to `/usr/sbin/sendmail` with recipient arguments
3. Postfix handles SMTP delivery, DKIM signing, and routing
4. Recipients receive emails from `yolo@b-haven.org`

## Troubleshooting

**Issue**: "sendmail failed" errors
- **Solution**: Verify postfix is running: `/usr/sbin/postfix status`
- **Fix**: Start postfix: `sudo postfix start`

**Issue**: Emails not arriving  
- **Solution**: Check mail logs: `tail -f /var/log/mail.log`
- **Check**: Verify MX records for recipient domain are configured

## Security Notes

- Emails use the server's configured postfix settings
- DKIM/SPF handled by postfix automatically
- No credentials stored or transmitted
- All emails originate from `yolo@b-haven.org`
