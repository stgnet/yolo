package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ─── Git Tools ──────────────────────────────────────────────

// gitStatus returns the current git status in a structured format.
func (t *ToolExecutor) gitStatus(args map[string]any) string {
	cmd := exec.Command("git", "status", "--short", "--branch", "--porcelain")
	cmd.Dir = t.baseDir
	output, err := cmd.Output()
	if err != nil {
		return errorMessage("git status failed: %v", err)
	}

	var sb strings.Builder
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")

	// Parse branch name from first line
	if len(lines) > 0 && strings.HasPrefix(lines[0], "## ") {
		branch := strings.TrimPrefix(lines[0], "## ")
		sb.WriteString(fmt.Sprintf("Branch: %s\n", branch))
		lines = lines[1:]
	}

	// Track changes summary
	var modified, added, deleted, untracked []string

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		status := line[:2]
		filePath := strings.TrimPrefix(line[3:], "\t")

		switch status {
		case " M", "MM":
			modified = append(modified, filePath)
		case "A ", "AA":
			added = append(added, filePath)
		case "D ", "DD":
			deleted = append(deleted, filePath)
		case "??", "A?":
			untracked = append(untracked, filePath)
		default:
			sb.WriteString(fmt.Sprintf("  %s\n", line))
		}
	}

	if len(modified) == 0 && len(added) == 0 && len(deleted) == 0 && len(untracked) == 0 {
		sb.WriteString("Working tree clean\n")
	} else {
		if len(modified) > 0 {
			sb.WriteString(fmt.Sprintf("\nModified (%d):\n", len(modified)))
			for _, f := range modified {
				sb.WriteString(fmt.Sprintf("  M %s\n", f))
			}
		}
		if len(added) > 0 {
			sb.WriteString(fmt.Sprintf("\nAdded (%d):\n", len(added)))
			for _, f := range added {
				sb.WriteString(fmt.Sprintf("  A %s\n", f))
			}
		}
		if len(deleted) > 0 {
			sb.WriteString(fmt.Sprintf("\nDeleted (%d):\n", len(deleted)))
			for _, f := range deleted {
				sb.WriteString(fmt.Sprintf("  D %s\n", f))
			}
		}
		if len(untracked) > 0 {
			sb.WriteString(fmt.Sprintf("\nUntracked (%d):\n", len(untracked)))
			for _, f := range untracked {
				sb.WriteString(fmt.Sprintf("  ? %s\n", f))
			}
		}
	}

	return sb.String()
}

// gitDiff shows the diff of changes.
func (t *ToolExecutor) gitDiff(args map[string]any) string {
	file := getStringArg(args, "file", "")

	var cmd *exec.Cmd
	if file != "" {
		fullPath := filepath.Join(t.baseDir, file)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			return errorMessage("file not found: %s", file)
		}
		cmd = exec.Command("git", "diff", "--no-color", file)
	} else {
		cmd = exec.Command("git", "diff", "--no-color")
	}

	cmd.Dir = t.baseDir
	output, err := cmd.Output()
	if err != nil {
		// No changes is OK
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return "No changes (working tree clean)"
		}
		return errorMessage("git diff failed: %v", err)
	}

	if len(output) == 0 {
		return "No changes (working tree clean)"
	}

	return string(output)
}

// gitLog shows recent commit history.
func (t *ToolExecutor) gitLog(args map[string]any) string {
	limit := getIntArg(args, "limit", 10)

	cmd := exec.Command("git", "log", fmt.Sprintf("-%d", limit), "--oneline", "--decorate")
	cmd.Dir = t.baseDir
	output, err := cmd.Output()
	if err != nil {
		return errorMessage("git log failed: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 0 {
		return "No commits found"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Recent commits (last %d):\n\n", limit))
	for _, line := range lines {
		sb.WriteString(fmt.Sprintf("  %s\n", line))
	}

	return sb.String()
}

// gitBranch lists all branches.
func (t *ToolExecutor) gitBranch(args map[string]any) string {
	cmd := exec.Command("git", "branch", "-v", "--sort=-committerdate")
	cmd.Dir = t.baseDir
	output, err := cmd.Output()
	if err != nil {
		return errorMessage("git branch failed: %v", err)
	}

	var sb strings.Builder
	sb.WriteString("Branches:\n\n")
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if strings.HasPrefix(line, "* ") {
			sb.WriteString(fmt.Sprintf("  *%s (current)\n", line[2:]))
		} else if strings.TrimSpace(line) != "" {
			sb.WriteString(fmt.Sprintf("   %s\n", line))
		}
	}

	return sb.String()
}

// gitCheckout checks out a branch or file.
func (t *ToolExecutor) gitCheckout(args map[string]any) string {
	branch := getStringArg(args, "branch", "")
	file := getStringArg(args, "file", "")

	if branch == "" && file == "" {
		return errorMessage("branch or file is required")
	}

	var cmd *exec.Cmd
	if file != "" {
		cmd = exec.Command("git", "checkout", "--", file)
	} else {
		cmd = exec.Command("git", "checkout", branch)
	}

	cmd.Dir = t.baseDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return errorMessage("git checkout failed: %v\n%s", err, string(output))
	}

	if file != "" {
		return fmt.Sprintf("Restored file from HEAD: %s", file)
	}
	return fmt.Sprintf("Switched to branch: %s", branch)
}

// gitCommit commits changes with a message.
func (t *ToolExecutor) gitCommit(args map[string]any) string {
	message := getStringArg(args, "message", "")
	all := getBoolArg(args, "all", false) // auto-stage all changes

	if message == "" {
		return errorMessage("message is required")
	}

	var cmdArgs []string
	if all {
		cmdArgs = append(cmdArgs, "-am")
	} else {
		cmdArgs = append(cmdArgs, "-m")
	}
	cmdArgs = append(cmdArgs, message)

	cmd := exec.Command("git", append([]string{"commit"}, cmdArgs...)...)
	cmd.Dir = t.baseDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return errorMessage("git commit failed: %v\n%s", err, string(output))
	}

	// Extract commit hash from output
	var commitHash string
	if len(output) > 0 {
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			if strings.Contains(line, "Files changed:") {
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					commitHash = parts[1]
					break
				}
			}
		}
	}

	return fmt.Sprintf("Committed: %s (%s)", message, commitHash)
}

// gitAdd stages files for commit.
func (t *ToolExecutor) gitAdd(args map[string]any) string {
	file := getStringArg(args, "file", "")

	var cmdArgs []string
	if file != "" {
		cmdArgs = append(cmdArgs, file)
	} else {
		cmdArgs = append(cmdArgs, ".")
	}

	cmd := exec.Command("git", append([]string{"add"}, cmdArgs...)...)
	cmd.Dir = t.baseDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return errorMessage("git add failed: %v\n%s", err, string(output))
	}

	if file != "" {
		return fmt.Sprintf("Staged file: %s", file)
	}
	return "Staged all changes"
}

// gitShow shows commit details.
func (t *ToolExecutor) gitShow(args map[string]any) string {
	commit := getStringArg(args, "commit", "HEAD")

	cmd := exec.Command("git", "show", "--stat", commit)
	cmd.Dir = t.baseDir
	output, err := cmd.Output()
	if err != nil {
		return errorMessage("git show failed: %v", err)
	}

	return string(output)
}

// gitRemote shows configured remotes.
func (t *ToolExecutor) gitRemote(args map[string]any) string {
	cmd := exec.Command("git", "remote", "-v")
	cmd.Dir = t.baseDir
	output, err := cmd.Output()
	if err != nil {
		return errorMessage("git remote failed: %v", err)
	}

	var sb strings.Builder
	sb.WriteString("Remotes:\n\n")
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if strings.TrimSpace(line) != "" {
			sb.WriteString(fmt.Sprintf("  %s\n", line))
		}
	}

	return sb.String()
}

// ─── Git Status Helpers (structured output for parsing) ────────

// GitStatusJSON provides structured git status for programmatic access.
func (t *ToolExecutor) gitStatusJSON(args map[string]any) string {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = t.baseDir
	output, err := cmd.Output()
	if err != nil {
		return errorMessage("git status failed: %v", err)
	}

	type GitFileStatus struct {
		Status string `json:"status"`
		File   string `json:"file"`
	}

	var files []GitFileStatus

	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}

		status := line[:2]
		filePath := strings.TrimPrefix(line[3:], "\t")

		files = append(files, GitFileStatus{
			Status: status,
			File:   filePath,
		})
	}

	jsonOutput, _ := json.MarshalIndent(files, "", "  ")
	return string(jsonOutput)
}
