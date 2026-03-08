#!/bin/bash
# send_email.sh - Send email via system mail command
# Usage: send_email.sh "subject" "body" "recipients"
# Recipients should be comma-separated

SUBJECT="$1"
BODY="$2"
RECIPIENTS="$3"

if [ -z "$SUBJECT" ] || [ -z "$BODY" ] || [ -z "$RECIPIENTS" ]; then
    echo "Usage: send_email.sh \"subject\" \"body\" \"recipients\"" >&2
    exit 1
fi

# Replace commas with spaces for mail command recipient list
RECIPIENT_LIST=$(echo "$RECIPIENTS" | tr ',' ' ')

# Try to use the system mail command
if command -v mail &> /dev/null; then
    echo "$BODY" | mail -s "$SUBJECT" $RECIPIENT_LIST
    exit $?
elif command -v sendmail &> /dev/null; then
    # Fallback to sendmail if available
    {
        echo "To: $RECIPIENTS"
        echo "Subject: $SUBJECT"
        echo ""
        echo "$BODY"
    } | sendmail -t
    exit $?
else
    echo "Error: Neither 'mail' nor 'sendmail' command found. Please install a mail client." >&2
    exit 1
fi
