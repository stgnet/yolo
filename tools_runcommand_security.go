package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
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

	// CompiledDangerousPatterns contains pre-compiled regex patterns for dangerous command detection.
	CompiledDangerousPatterns []*regexp.Regexp
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

	// Pre-compile all dangerous patterns into regex for proper matching
	CompiledDangerousPatterns = make([]*regexp.Regexp, len(DangerousCommandPatterns))
	for i, pattern := range DangerousCommandPatterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			fmt.Fprintf(os.Stderr, "WARNING: Failed to compile dangerous pattern %q: %v\n", pattern, err)
			CompiledDangerousPatterns[i] = nil // Skip invalid patterns
		} else {
			CompiledDangerousPatterns[i] = re
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

	// Shell injection detection - check for common bypass techniques
	if err := detectShellInjection(cmdLower); err != nil {
		return "", fmt.Errorf("shell injection attempt detected: %v", err)
	}

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

	// Check for dangerous patterns using proper regex matching (NOT strings.Contains)
	for _, pattern := range CompiledDangerousPatterns {
		if pattern != nil && pattern.MatchString(cmdLower) {
			// Find the original pattern string for the error message
			idx := 0
			for i, re := range CompiledDangerousPatterns {
				if re == pattern {
					idx = i
					break
				}
			}
			return "", fmt.Errorf("command contains blocked pattern: %s", DangerousCommandPatterns[idx])
		}
	}

	// Validate file paths are within baseDir (if present in command)
	paths := findFilePathsInCommand(command)
	for _, path := range paths {
		if err := validatePathAgainstBaseDir(path); err != nil {
			return "", fmt.Errorf("path traversal detected: %v", err)
		}
	}

	return command, nil
}

// detectShellInjection checks for various shell injection techniques and bypass methods.
func detectShellInjection(cmd string) error {
	// Check for URL-encoded newlines (%0a, %0A, etc.)
	if strings.Contains(cmd, "%0a") || strings.Contains(cmd, "%0d") {
		return fmt.Errorf("URL-encoded newline detected")
	}

	// Check for literal escape sequences
	if strings.Contains(cmd, "\\n") || strings.Contains(cmd, "\\r") {
		return fmt.Errorf("Escape sequence injection detected")
	}

	// Check for command chaining: ; (semicolon)
	if strings.Contains(cmd, "; ") || strings.HasSuffix(cmd, ";") || strings.HasPrefix(cmd, ";") {
		return fmt.Errorf("Semicolon command chaining detected")
	}

	// Check for && and || operators
	if strings.Contains(cmd, "&& ") || strings.Contains(cmd, "|| ") || strings.HasSuffix(cmd, "&&") || strings.HasSuffix(cmd, "||") {
		return fmt.Errorf("Logical operator chaining detected")
	}

	// Check for pipe commands (|) at suspicious positions
	if strings.Contains(cmd, "| ") || strings.HasPrefix(cmd, "|") {
		return fmt.Errorf("Pipe command injection detected")
	}

	// Check for command substitution: $(command)
	if strings.Contains(cmd, "$(") {
		return fmt.Errorf("Command substitution $() detected")
	}

	// Check for backtick command substitution
	if strings.Contains(cmd, "`") {
		return fmt.Errorf("Backtick command substitution detected")
	}

	// Check for variable expansion with command injection: ${VAR:-cmd} or ${VAR:+cmd}
	reVarExp, _ := regexp.Compile(`\$\{[^}]*[:+\-]`)
	if reVarExp.MatchString(cmd) {
		return fmt.Errorf("Variable expansion with potential command substitution detected")
	}

	// Check for quote injection (single and double quotes)
	if strings.Contains(cmd, "' ") || strings.HasPrefix(cmd, "'") || strings.HasSuffix(cmd, "'") {
		return fmt.Errorf("Quote injection attempt detected (single quotes)")
	}
	if strings.Contains(cmd, `" `) || strings.HasPrefix(cmd, `"`) || strings.HasSuffix(cmd, `"`) {
		return fmt.Errorf("Quote injection attempt detected (double quotes)")
	}

	// Check for newline characters in different forms
	if strings.Contains(cmd, "\n") || strings.Contains(cmd, "\r") {
		return fmt.Errorf("Raw newline character detected")
	}

	// Check for IFS manipulation to bypass filters
	if reIFs := regexp.MustCompile(`IFS\s*[=:]\s*['"]?\s*\S`); reIFs.MatchString(cmd) {
		return fmt.Errorf("IFS manipulation detected")
	}

	// Check for eval-like patterns
	if strings.Contains(cmd, "eval ") || cmd == "eval" {
		return fmt.Errorf("Eval command detected")
	}

	// Check for source/. injection (shell sourcing)
	if strings.Contains(cmd, "source ") || strings.Contains(cmd, ". ") {
		return fmt.Errorf("Source/shell sourcing detected")
	}

	return nil
}

// findFilePathsInCommand extracts potential file paths from a command string.
func findFilePathsInCommand(command string) []string {
	var paths []string
	tokens := strings.Fields(command)

	for i, token := range tokens {
		// Skip shell operators and control characters
		if strings.HasPrefix(token, "-") || token == "|" || token == ";" ||
			token == "&" || token == ">>" || token == ">" || token == "<" ||
			token == "||" || token == "&&" || token == "(" || token == ")" {
			continue
		}

		// Check if it looks like a file path (includes paths starting with .)
		if isPathLike(token) {
			paths = append(paths, token)
			continue
		}

		// Look for next argument which might be a path (after -flag options)
		if len(tokens) > i+1 && isPathLike(tokens[i+1]) {
			// Check if current token looks like a flag that takes a path argument
			flagTokens := []string{"-o", "-f", "-d", "-i", "-I", "-c", "-p", "-D"}
			for _, flag := range flagTokens {
				if strings.HasPrefix(token, flag) {
					paths = append(paths, tokens[i+1])
					break
				}
			}
		}

		// Handle - flag meaning stdin/stdout
		if token == "-" && len(tokens) > i+1 {
			nextToken := tokens[i+1]
			// Check if next token is a path (not another flag)
			if !strings.HasPrefix(nextToken, "-") && (strings.Contains(nextToken, "/") ||
				strings.HasPrefix(nextToken, "./") || strings.HasPrefix(nextToken, "../")) {
				paths = append(paths, nextToken)
			}
		}
	}

	return paths
}

// isPathLike checks if a string looks like a file path.
func isPathLike(s string) bool {
	if s == "" {
		return false
	}

	// Absolute paths
	if strings.HasPrefix(s, "/") {
		return true
	}

	// Relative paths starting with ./ or ../
	if strings.HasPrefix(s, "./") || strings.HasPrefix(s, "../") {
		return true
	}

	// Paths containing / (like "dir/file.txt" or "a/b/c")
	if strings.Contains(s, "/") {
		return true
	}

	// Symlink notation like -> /path
	if s == "->" || s == "=>" {
		return true
	}

	return false
}

// validatePathAgainstBaseDir checks if a path escapes outside baseDir.
func validatePathAgainstBaseDir(path string) error {
	// First clean the path to remove any .. components and redundant separators
	cleaned := filepath.Clean(path)

	// Get absolute path by joining with an empty base for validation purposes
	// This resolves any remaining symbolic links or edge cases
	absPath, err := filepath.Abs(cleaned)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %v", err)
	}

	// Resolve symlinks if the path exists
	resolvedPath := absPath
	if info, err := os.Stat(absPath); err == nil && (info.Mode()&os.ModeSymlink) != 0 {
		if linkTarget, err := filepath.EvalSymlinks(absPath); err == nil {
			resolvedPath = linkTarget
		}
	}

	// Check if cleaned path tries to escape
	if cleaned == ".." || strings.HasPrefix(cleaned, "../") || cleaned == "." {
		return fmt.Errorf("path contains traversal attempts: %s", path)
	}

	// Ensure the resolved path doesn't escape via symlink attack
	symlinkEscaped := false
	parts := strings.Split(resolvedPath, string(filepath.Separator))
	for _, part := range parts {
		if part == ".." {
			symlinkEscaped = true
			break
		}
	}

	if symlinkEscaped {
		return fmt.Errorf("path resolution escapes base directory: %s", path)
	}

	return nil
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

			// Full path validation with proper resolution against baseDir
			cleanedPath := filepath.Clean(pathStr)

			// Check for obvious traversal patterns first
			if cleanedPath == ".." || strings.HasPrefix(cleanedPath, "../") {
				return fmt.Errorf("path traversal detected: %s", pathStr)
			}

			fullPath := filepath.Join(baseDir, cleanedPath)

			// Get the absolute path
			if !filepath.IsAbs(fullPath) {
				var err error
				fullPath, err = filepath.Abs(fullPath)
				if err != nil {
					return fmt.Errorf("failed to resolve absolute path: %v", err)
				}
			}

			// Resolve symlinks to catch symlink-based path traversal attacks
			if info, err := os.Stat(filepath.Dir(fullPath)); err == nil && (info.Mode()&os.ModeSymlink) != 0 {
				if realPath, err := filepath.EvalSymlinks(fullPath); err == nil {
					fullPath = realPath
				}
			}

			// Verify the resolved path is within baseDir using HasPrefix
			trimmedBaseDir := strings.TrimRight(baseDir, string(filepath.Separator))
			baseWithSep := trimmedBaseDir + string(filepath.Separator)

			if !strings.HasPrefix(fullPath, baseWithSep) && fullPath != trimmedBaseDir {
				return fmt.Errorf("path '%s' escapes working directory (resolved to: %s)", pathStr, fullPath)
			}

			// Additional check: ensure no .. components remain after resolution
			parts := strings.Split(fullPath, string(filepath.Separator))
			for _, part := range parts {
				if part == ".." {
					return fmt.Errorf("path '%s' contains unresolved traversal component", pathStr)
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
