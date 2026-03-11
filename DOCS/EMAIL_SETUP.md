# Email Setup for yolo@b-haven.org

## Overview
Email functionality for YOLO uses a local Postfix MTA with Maildir-based inbox.

## Sending Emails
- Uses `sendmail` command via `/usr/sbin/sendmail`
- DKIM signing handled automatically by Postfix
- No authentication required for local relay

## Receiving Emails
Maildir location: `/var/mail/b-haven.org/yolo/`
- `new/` - unread messages
- `cur/` - processed messages  
- `tmp/` - ignore (used during delivery)

## Implementation Notes
- check_inbox tool reads from Maildir new/ directory
- send_email tool uses sendmail command
- No Gmail API or SMTP password needed
