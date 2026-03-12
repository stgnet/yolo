# YOLO Critical Notes

## Source Code Location
- **Your source code is in `.` (current directory), NOT in `yolo/`**
- Working directory: /Users/sgriepentrog/src/yolo
- Files are directly in the root, e.g., `tools_inbox.go`, not `yolo/tools_inbox.go`

## Restarting YOLO
- **Use the `restart` tool to restart YOLO - DO NOT use os.Exit() or kill yourself**
- The restart tool properly rebuilds and restarts the agent

## Email Testing Guidelines
- Unit tests for composeResponseToEmail should test the function directly WITHOUT sending emails
- Tests that send real emails MUST be gated behind `YOLO_TEST_EMAIL=1` environment variable
- Tests must call `skipUnlessEmailEnabled(t)` helper if they could send real emails
- DO NOT add test cases with valid subject+body in unit tests - they will send real emails via sendmail

## Email Response Generation
- composeResponseToEmail calls LLM directly for ALL emails (no pattern matching)
- The function generates response text but does NOT send the email
- processInboxWithResponse calls composeResponseToEmail then sendEmail separately
- This separation allows testing response generation without sending emails
