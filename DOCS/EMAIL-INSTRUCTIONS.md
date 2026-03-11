# Email Inbox for yolo@b-haven.org

## How It Works
Inbound email to `yolo@b-haven.org` is delivered by Postfix to a Maildir on disk. Each email is a single file. No database, no IMAP — just files.

## Maildir Location
```
/var/mail/b-haven.org/yolo/
├── new/    ← unread messages land here
├── cur/    ← move messages here after processing
└── tmp/    ← ignore (used during delivery)
```

## Reading New Mail

### Check for new messages
```bash
ls /var/mail/b-haven.org/yolo/new/
```
If empty, no new mail.

### Read a message
```bash
cat /var/mail/b-haven.org/yolo/new/<filename>
```
The file contains the full RFC 2822 email — headers and body. Key headers to look for:
- `From:` — sender
- `Subject:` — subject line
- `Date:` — when sent
- `Content-Type:` — if `multipart/*`, the body has MIME parts

### Mark as read (move to cur/)
After processing a message, move it so you don't re-process it:
```bash
mv /var/mail/b-haven.org/yolo/new/<filename> /var/mail/b-haven.org/yolo/cur/
```

### Parse with Python (for structured access)
```python
import email
import os

MAILDIR = '/var/mail/b-haven.org/yolo'

for fname in os.listdir(os.path.join(MAILDIR, 'new')):
    path = os.path.join(MAILDIR, 'new', fname)
    with open(path) as f:
        msg = email.message_from_file(f)
    
    print(f"From: {msg['From']}")
    print(f"Subject: {msg['Subject']}")
    print(f"Date: {msg['Date']}")
    
    # Get plain text body
    if msg.is_multipart():
        for part in msg.walk():
            if part.get_content_type() == 'text/plain':
                print(part.get_payload(decode=True).decode())
                break
    else:
        print(msg.get_payload(decode=True).decode())
    
    # Move to cur/ after processing
    os.rename(path, os.path.join(MAILDIR, 'cur', fname))
```

## Sending Replies
To reply from any `@b-haven.org` address:
```bash
echo "Subject: Re: whatever
From: yolo@b-haven.org
To: recipient@example.com
Date: $(date -R)
MIME-Version: 1.0
Content-Type: text/plain; charset=utf-8

Your reply body here." | /usr/sbin/sendmail -f yolo@b-haven.org recipient@example.com
```
Postfix handles DKIM signing automatically.

## Polling Pattern
For periodic checking (e.g., in a heartbeat or cron):
```bash
# Quick check — nonzero exit if no new mail
test -n "$(ls -A /var/mail/b-haven.org/yolo/new/ 2>/dev/null)"
```
