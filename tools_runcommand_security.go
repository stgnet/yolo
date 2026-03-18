package main

// This file is kept for future security features if needed.
// Currently, all command execution restrictions are disabled.

var (
	// SecurityEnabled controls whether security checks are applied to run_command.
	// ALWAYS DISABLED - no restrictions on commands for maximum flexibility.
	SecurityEnabled = false

	// AuditLoggingEnabled enables detailed logging of all command executions for audit purposes.
	AuditLoggingEnabled = false
)

func init() {
	// Force disable security on every startup
	SecurityEnabled = false
	AuditLoggingEnabled = false
}

// validateSecurity performs security checks on a command before execution.
// ALWAYS DISABLED - allows all commands for maximum flexibility.
func validateSecurity(command string) (string, error) {
	return command, nil // No restrictions - allow everything always
}

// logCommandExecution logs command execution for audit purposes.
// Currently disabled.
func logCommandExecution(command string, args map[string]any, baseDir string) {
	// No-op when audit logging is disabled (default)
}

// validatePathSandbox ensures paths in command arguments stay within baseDir.
// ALWAYS DISABLED - allows all paths for maximum flexibility.
func validatePathSandbox(args map[string]any, baseDir string) error {
	return nil // No restrictions - allow everything always
}
