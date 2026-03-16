package toolexecutor

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// GitExecutor implements git operations
type GitExecutor struct {
	basePath string
}

// NewGitExecutor creates a new GitExecutor instance
func NewGitExecutor() *GitExecutor {
	return &GitExecutor{basePath: "."}
}

// gitListBranches lists all branches in the repository
func (e *GitExecutor) gitListBranches(args map[string]any) (map[string]any, error) {
	cmd := e.buildCmd("branch", "-a")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git branch failed: %v, output: %s", err, string(output))
	}

	result := map[string]any{
		"output":   string(output),
		"success":  true,
		"branches": splitLines(string(output)),
	}
	return result, nil
}

// gitDiff shows differences between commits and working tree
func (e *GitExecutor) gitDiff(args map[string]any) (map[string]any, error) {
	var cmdArgs []string
	if opts, ok := args["options"].([]string); ok {
		cmdArgs = opts
	}

	cmd := e.buildCmd(append([]string{"diff"}, cmdArgs...)...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git diff failed: %v, output: %s", err, string(output))
	}

	result := map[string]any{
		"output":  string(output),
		"success": true,
	}
	return result, nil
}

// gitStatus shows the working tree status
func (e *GitExecutor) gitStatus(args map[string]any) (map[string]any, error) {
	var cmdArgs []string
	if opts, ok := args["options"].([]string); ok {
		cmdArgs = opts
	}

	cmd := e.buildCmd(append([]string{"status"}, cmdArgs...)...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git status failed: %v, output: %s", err, string(output))
	}

	result := map[string]any{
		"output":  string(output),
		"success": true,
	}
	return result, nil
}

// buildCmd creates a command for the executor's base path
func (e *GitExecutor) buildCmd(args ...string) *exec.Cmd {
	cmd := exec.Command("git", args...)
	cmd.Dir = e.basePath
	return cmd
}

// splitLines splits a string by newlines and filters empty lines
func splitLines(s string) []string {
	lines := []string{}
	for _, line := range splitString(s, "\n") {
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

// splitString splits s into all substrings separated by sep and returns a slice of the substrings between delimiters.
func splitString(s string, sep string) []string {
	if sep == "" {
		return []string{}
	}
	count := 0
	i := 0
	for len(s) > i {
		if s[i:i+len(sep)] == sep {
			count++
			i += len(sep)
		} else {
			i++
		}
	}

	a := make([]string, count+1)
	j := 0
	i = 0
	for len(s) > i {
		if s[i:i+len(sep)] == sep {
			a[j] = s[:i]
			j++
			s = s[i+len(sep):]
			i = 0
		} else {
			i++
		}
	}
	a[j] = s
	return a
}

// ReadDir reads the contents of a directory
func (e *GitExecutor) ReadDir(name string) ([]os.FileInfo, error) {
	path := filepath.Join(e.basePath, name)
	dir, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer dir.Close()
	return dir.Readdir(-1)
}

// Stat returns information about a file or directory
func (e *GitExecutor) Stat(name string) (os.FileInfo, error) {
	path := filepath.Join(e.basePath, name)
	return os.Stat(path)
}
