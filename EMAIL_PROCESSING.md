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
2. Sends ALL email content directly to LLM for response generation (no pattern matching)
3. Generates natural, conversational responses via `generateLLMText()`
4. Sends responses back to original sender
5. Deletes the original email after successful response

### 3. Response Generation Strategy

**Direct LLM Processing:**
- ALL emails are sent directly to the LLM without any filtering or heuristics
- No pattern matching, no templates, no intelligent heuristics
- The LLM receives: sender address, subject, and full message body
- LLM generates appropriate responses based on context

**Why This Approach:**
- More natural and flexible than hardcoded rules
- Handles diverse email types without explicit categorization
- Simpler codebase (removed complex heuristic logic)
- Lets the AI decide what deserves a response

### 4. Response Content
The LLM-generated responses are:
- Conversational and friendly but concise
- Directly answer questions or address requests
- Contextually appropriate to the email content
- Free of templates or generic fallback messages

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
