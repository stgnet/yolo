# YOLO CORRECTIONS - IMPORTANT

## File Path Corrections
- **Source code location:** `.` (current directory), NOT `yolo/`
- Example: `tools_inbox.go` not `yolo/tools_inbox.go`
- Working directory is `/Users/sgriepentrog/src/yolo`

## Restart Procedure
- **USE THE `restart` TOOL** - do NOT use os.Exit() or kill yourself
- The `restart` tool rebuilds and restarts YOLO properly
- Using os.Exit() terminates the process instead of restarting it

## Email Response Testing
To test email responses without actually sending:
1. Simulate an inbound email in a test
2. Check what response would be generated  
3. Prevent actual email from being sent during the test
