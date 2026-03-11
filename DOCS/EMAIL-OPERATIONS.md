# Email Operations for yolo@b-haven.org

## Maildir Inbox
- **Location**: `/var/mail/b-haven.org/yolo/`
- **New messages**: `new/` directory
- **Processed messages**: Move to `cur/` directory
- **Format**: RFC 2822 raw email files (headers + body)

## Sending Email
Use the sendmail command for outgoing emails:
```bash
echo "Subject: Your Subject
From: yolo@b-haven.org
To: recipient@example.com
Date: $(date -R)
MIME-Version: 1.0
Content-Type: text/plain; charset=utf-8

Email body here." | /usr/sbin/sendmail -f yolo@b-haven.org recipient@example.com
```

Postfix handles DKIM signing automatically.

## Checking Inbox
```bash
# Check for new messages
ls /var/mail/b-haven.org/yolo/new/

# Read a message
cat /var/mail/b-haven.org/yolo/new/<filename>

# Mark as read (move to cur)
mv /var/mail/b-haven.org/yolo/new/<filename> /var/mail/b-haven.org/yolo/cur/
```
