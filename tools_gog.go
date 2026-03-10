// GOG (Google CLI) Tool Implementation
// https://github.com/danielmiessler/gog - Google CLI for OpenClaw agents

package main

import "strings"

func (t *ToolExecutor) gog(args map[string]any) string {
	command := getStringArg(args, "command", "")
	if command == "" {
		return "Error: command parameter is required. Examples:\n  - 'gmail search inbox:unread --max 5'\n  - 'calendar list events'\n  - 'drive list'\n  - 'contacts list'"
	}

	// Use the runCommand tool to execute gog with JSON output
	fullCommand := map[string]any{"command": "gog --json " + command}
	result := t.runCommand(fullCommand)
	
	if result == "" {
		return "Error: gog command returned no output"
	}
	
	// Check for common error patterns
	if strings.Contains(result, "Error") || strings.Contains(result, "error") {
		return result
	}
	
	return result
}
