# Email Processing System for YOLO

## Overview
YOLO can automatically process inbound emails to `yolo@b-haven.org` with intelligent response generation and automatic deletion.

## Features

### 1. Email Inbox Reading (`check_inbox`)
- Reads all new emails from Maildir (`/var/mail/b-haven.org/yolo/new/`)
- Displays full email details (From, Subject, Date, Content-Type, Body)
- Optional `mark_read` parameter to move processed emails to `cur/` directory

### 2. Automated Email Processing (`process_inbox_with_response`)
This tool implements the complete workflow: **Read → Respond → Delete**

**Workflow:**
1. Reads all new emails from inbox
2. Analyzes each email using intelligent heuristics to determine if a response is needed
3. Composes context-aware auto-responses
4. Sends responses back to original sender
5. Deletes the original email after successful response

### 3. Smart Response Heuristics

The system intelligently determines which emails need responses:

**Responds to:**
- Emails with questions (subject/body contains "?")
- Emails with requests (subject/body contains "please", "help", "need", "when")
- Emails from Scott (@stg.net) - prioritized human sender
- Short messages (< 5000 chars) that appear to be human communication

**Does NOT respond to:**
- Automated/system messages (build completed, test run, CI check, job finished, etc.)
- Long system logs or notifications
- Automated notifications

### 4. Response Generation
Responses include:
- Acknowledgment of received message
- Current timestamp
- Information about autonomous operation mode
- Priority handling for Scott's emails
- Professional closing

## Usage

**Check inbox:**
```
check_inbox(mark_read=false)
```

**Process all emails with auto-responses and deletion:**
```
process_inbox_with_response()
```

## Implementation Details

### Email Directory Structure
- **new/** - Incoming emails waiting for processing
- **cur/** - Processed/read emails
- **tmp/** - Temporary files during processing

### Files Modified
- `tools_inbox.go` - Core email processing logic with improved heuristics
- `tools_email.go` - Email sending functionality
- `tools_inbox_integration_test.go` - Integration tests for email processing

## Testing

All unit and integration tests pass:
- ✅ Email inbox reading tests
- ✅ Email response heuristic tests
- ✅ System log filtering tests
- ✅ Scott's emails prioritization tests
- ✅ Auto-deletion after response tests

## Future Enhancements

Potential improvements:
1. Customizable email keywords for auto-response triggers
2. Whitelist of trusted senders
3. Scheduled inbox processing (cron jobs)
4. Email categorization and tagging
5. Support for email attachments
