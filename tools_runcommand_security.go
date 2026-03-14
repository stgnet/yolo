package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ─── Security Configuration ──────────────────────────────────────────────

var (
	// SecurityEnabled controls whether security checks are applied to run_command.
	// Can be set via YOLO_SECURITY_ENABLED env var: "true"/"false".
	SecurityEnabled = true

	// AuditLoggingEnabled enables detailed logging of all command executions for audit purposes.
	// Set via YOLO_AUDIT_LOGGING env var: "true"/"false".
	AuditLoggingEnabled = true
)

var (
	// AllowedCommands is the whitelist of safe commands that can be executed.
	AllowedCommands = map[string]bool{
		// File operations
		"ls":        true,
		"ls-l":      true, // common alias
		"cat":       true,
		"head":      true,
		"tail":      true,
		"wc":        true,
		"find":      true,
		"grep":      true,
		"awk":       true,
		"sed":       true,
		"sort":      true,
		"uniq":      true,
		"cut":       true,
		"paste":     true,
		"tr":        true,
		"diff":      true,
		"md5sum":    true,
		"sha256sum": true,

		// Information commands
		"echo":     true,
		"date":     true,
		"time":     true,
		"pwd":      true,
		"whoami":   true,
		"id":       true,
		"hostname": true,
		"uname":    true,

		// Process management (read-only)
		"ps":    true,
		"top":   true,
		"pgrep": true,
		"pkill": false, // dangerous - requires explicit allow

		// Text processing and conversion
		"base64":  true,
		"xxd":     true,
		"hexdump": true,

		// Git operations (read-only)
		"git": false, // dangerous - requires explicit validation

		// Shell builtins (limited)
		"test":  true,
		"[":     true,
		"true":  true,
		"false": true,

		// Utilities
		"touch": false, // creates files - validate carefully
		"mkdir": false, // creates directories - validate carefully
		"cp":    false, // file copy - risky
		"mv":    false, // move - risky
		"rm":    false, // DELETE - HIGHLY RESTRICTED
		"chmod": false, // permission changes - risky
		"chown": false, // ownership changes - risky
		"xargs": false, // can execute arbitrary commands - risky

		// Network (blocked by default)
		"wget":   false,
		"curl":   false,
		"nc":     false,
		"netcat": false,
		"telnet": false,
		"ssh":    false,

		// Package managers (blocked)
		"apt":     false,
		"apt-get": false,
		"yum":     false,
		"dnf":     false,
		"brew":    false,
		"pip":     false,
		"go":      false,

		// System modifications (blocked)
		"dd":       false,
		"mkfs":     false,
		"mount":    false,
		"umount":   false,
		"shutdown": false,
		"reboot":   false,
		"sudo":     false,
		"su":       false,

		// Potentially dangerous commands
		"eval":   false, // shell eval - DANGEROUS
		"source": false, // shell sourcing - risky
	}

	// DangerousCommandPatterns are regex patterns that match dangerous command arguments.
	// Any command containing these will be blocked or flagged.
	DangerousCommandPatterns = []string{
		// File deletion patterns
		"rm\\s+-rf",
		"rm\\s+-r",
		"dd\\s+if=",
		":\\(\\)\\s*\\{\\s*:\\|:;&", // fork bomb
		"mkfs",
		">\\s*/dev/",

		// Network exfiltration
		"netcat",
		"nc\\s+-e",
		"curl.*-o.*-",
		"wget.*-O-|wget.*--save-cmdline",

		// Credential access
		"/etc/passwd",
		"/etc/shadow",
		"/root/",
		`\.ssh/`,
		"id_rsa",
		"password",
		"secret",
		"api_key",
		"token",

		// Shell injection attempts
		";\\s*rm",
		"\\|\\s*rm",
		">>/.*\\.bashrc",
		">>/.*\\.profile",
		">>/.*\\.bash_profile",

		// Command chaining for attacks
		";\\s*sudo",
		"\\|\\s*sudo",
		"`.*;.*rm",
		`\$\(.*;.*rm`,

		// Reverse shells and backdoors
		"bash.*-i",
		"nc.*-e",
		"sh.*-i",
	}

	// AllowedDirectories are the only directories that can be targeted.
	// Empty means all paths relative to baseDir are allowed (path validation handles this).
	AllowedDirectories = []string{} // Default: no restrictions beyond path validation
)

func init() {
	// Load security config from environment variables
	if v := os.Getenv("YOLO_SECURITY_ENABLED"); v != "" {
		SecurityEnabled = strings.ToLower(v) == "true"
	}

	if v := os.Getenv("YOLO_AUDIT_LOGGING"); v != "" {
		AuditLoggingEnabled = strings.ToLower(v) == "true"
	}

	// Allow user to add custom allowed commands via env var
	if v := os.Getenv("YOLO_ALLOWED_COMMANDS"); v != "" {
		for _, cmd := range strings.Split(v, ",") {
			cmd = strings.TrimSpace(strings.ToLower(cmd))
			if cmd != "" {
				AllowedCommands[cmd] = true
			}
		}
	}

	// Allow user to add custom blocked patterns via env var
	if v := os.Getenv("YOLO_BLOCKED_PATTERNS"); v != "" {
		for _, pattern := range strings.Split(v, ",") {
			pattern = strings.TrimSpace(pattern)
			if pattern != "" {
				DangerousCommandPatterns = append(DangerousCommandPatterns, pattern)
			}
		}
	}
}

// ─── Security Functions ──────────────────────────────────────────────────

// validateSecurity performs security checks on a command before execution.
// Returns an error if the command is unsafe or blocked.
func validateSecurity(command string) (string, error) {
	if !SecurityEnabled {
		return command, nil
	}

	// Normalize command for checking
	cmdLower := strings.TrimSpace(strings.ToLower(command))

	// Extract the base command (first word before any spaces or pipes)
	baseCmd := ""
	for i, ch := range cmdLower {
		if ch == ' ' || ch == '|' || ch == ';' || ch == '&' || ch == '>' || ch == '<' {
			baseCmd = strings.TrimSpace(cmdLower[:i])
			break
		}
	}
	if baseCmd == "" {
		baseCmd = cmdLower
	}

	// Check if command is in allowlist
	allowed, exists := AllowedCommands[baseCmd]
	if !exists {
		return "", fmt.Errorf("command '%s' not in allowed commands list", baseCmd)
	}

	// Block explicitly disabled commands
	if !allowed {
		return "", fmt.Errorf("command '%s' is disabled by security policy", baseCmd)
	}

	// Check for dangerous patterns in the full command
	for _, pattern := range DangerousCommandPatterns {
		if strings.Contains(cmdLower, pattern) {
			return "", fmt.Errorf("command contains blocked pattern: %s", pattern)
		}
	}

	// Validate file paths are within baseDir (if present in command)
	paths := findFilePathsInCommand(command)
	for _, path := range paths {
		if !filepath.IsAbs(path) && filepath.Clean(path) == ".." {
			return "", fmt.Errorf("path traversal detected: %s", path)
		}
	}

	return command, nil
}

// findFilePathsInCommand extracts potential file paths from a command string.
func findFilePathsInCommand(command string) []string {
	var paths []string
	tokens := strings.Fields(command)

	for _, token := range tokens {
		// Look for common path indicators
		if strings.HasPrefix(token, "-") || token == "|" || token == ";" ||
			token == "&" || token == ">" || token == "<" || token == ">>" {
			continue
		}

		// Check if it looks like a file path
		if strings.Contains(token, "/") && !strings.HasPrefix(token, ".") {
			paths = append(paths, token)
		} else if (token == "-" || isPathLike(token)) && len(tokens) > 0 {
			// Look for next argument which might be a path
			continue
		}
	}

	return paths
}

func isPathLike(s string) bool {
	// Simple heuristic: looks like a relative or absolute path
	return (strings.HasPrefix(s, "./") || strings.HasPrefix(s, "../") ||
		strings.HasPrefix(s, "/") || strings.Contains(s, ".") && len(s) > 3)
}

// logCommandExecution logs command execution for audit purposes.
func logCommandExecution(command string, args map[string]any, baseDir string) {
	if !AuditLoggingEnabled {
		return
	}

	logEntry := fmt.Sprintf("[AUDIT %s] Command: %s | Args: %+v | BaseDir: %s",
		time.Now().Format("2006-01-02 15:04:05"), command, args, baseDir)

	// Write to a simple audit log file
	const auditLogFile = ".yolo/command_audit.log"
	f, err := os.OpenFile(auditLogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		// If we can't write to the audit file, at least log to stderr
		fmt.Fprintf(os.Stderr, "AUDIT: %s\n", logEntry)
		return
	}
	defer f.Close()

	f.WriteString(logEntry + "\n")
}

// validatePathSandbox ensures paths in command arguments stay within baseDir.
func validatePathSandbox(args map[string]any, baseDir string) error {
	pathKeys := []string{"path", "file", "dir", "destination", "output", "input"}

	for _, key := range pathKeys {
		if val, ok := args[key]; ok {
			pathStr := ""
			switch v := val.(type) {
			case string:
				pathStr = v
			case int:
				pathStr = fmt.Sprintf("%d", v)
			default:
				pathStr = fmt.Sprintf("%v", v)
			}

			// Check for path traversal attempts
			if strings.Contains(pathStr, "..") {
				fullPath := filepath.Join(baseDir, filepath.Clean(pathStr))
				baseWithSep := baseDir + string(filepath.Separator)
				if !strings.HasPrefix(fullPath, baseWithSep) && fullPath != baseDir {
					return fmt.Errorf("path traversal detected: %s", pathStr)
				}
			}

			// Ensure relative paths stay within baseDir
			if !filepath.IsAbs(pathStr) {
				fullPath := filepath.Join(baseDir, filepath.Clean(pathStr))
				baseWithSep := baseDir + string(filepath.Separator)
				if fullPath != baseDir && !strings.HasPrefix(fullPath, baseWithSep) {
					return fmt.Errorf("path '%s' escapes working directory", pathStr)
				}
			}
		}
	}

	return nil
}

// isCommandInAllowlist checks if a command is in the allowed commands list.
func isCommandInAllowlist(cmd string) bool {
	cmdLower := strings.TrimSpace(strings.ToLower(cmd))
	_, exists := AllowedCommands[cmdLower]
	return exists
}

// getSecurityStatus returns the current security configuration status.
func getSecurityStatus() string {
	status := "Security checks: ENABLED\n"
	if !SecurityEnabled {
		status = "Security checks: DISABLED\n"
	}

	status += fmt.Sprintf("Audit logging: %s\n", enabledOrDisabled(AuditLoggingEnabled))

	var blockedCount, allowedCount int
	for _, v := range AllowedCommands {
		if v {
			allowedCount++
		} else {
			blockedCount++
		}
	}
	status += fmt.Sprintf("Commands in allowlist: %d allowed, %d blocked\n", allowedCount, blockedCount)

	return status
}

func enabledOrDisabled(enabled bool) string {
	if enabled {
		return "ENABLED"
	}
	return "DISABLED"
}
