package yolo

import (
	"fmt"
	"os/exec"
	"reflect"
)

// GitExecutor implements git operations through the tool executor interface
type GitExecutor struct {
	path string
}

// NewToolExecutor creates a new GitExecutor instance
func NewToolExecutor(basePath string, config interface{}) *GitExecutor {
	return &GitExecutor{path: basePath}
}

// gitListBranches lists all git branches
func (g *GitExecutor) gitListBranches(args map[string]any) (map[string]any, bool) {
	cmd := exec.Command("git", "branch")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return map[string]any{
			"error":   err.Error(),
			"output":  string(output),
			"success": false,
		}, false
	}

	return map[string]any{
		"branches":  string(output),
		"success":   true,
		"command":   "git branch",
		"exit_code": 0,
	}, true
}

// gitDiff shows the differences between commits and the working directory
func (g *GitExecutor) gitDiff(args map[string]any) (map[string]any, bool) {
	var cmd *exec.Cmd
	if args["unified"] != nil {
		unified, ok := args["unified"].(int)
		if !ok {
			return map[string]any{"error": "invalid unified parameter", "success": false}, false
		}
		cmd = exec.Command("git", "diff", fmt.Sprintf("-U%d", unified))
	} else {
		cmd = exec.Command("git", "diff")
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return map[string]any{
			"error":   err.Error(),
			"output":  string(output),
			"success": false,
		}, false
	}

	exitCode := 0
	if len(output) == 0 {
		exitCode = 1
	}

	return map[string]any{
		"diff":      string(output),
		"success":   exitCode == 0,
		"command":   "git diff",
		"exit_code": exitCode,
	}, true
}

// gitStatus shows the working tree status
func (g *GitExecutor) gitStatus(args map[string]any) (map[string]any, bool) {
	cmd := exec.Command("git", "status")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return map[string]any{
			"error":   err.Error(),
			"output":  string(output),
			"success": false,
		}, false
	}

	return map[string]any{
		"status":    string(output),
		"success":   true,
		"command":   "git status",
		"exit_code": 0,
	}, true
}

// gitCommand executes a generic git command based on the provided arguments
func (g *GitExecutor) gitCommand(args map[string]any) (map[string]any, bool) {
	command, ok := args["command"].(string)
	if !ok || command == "" {
		return map[string]any{"error": "command parameter required", "success": false}, false
	}

	var cmd *exec.Cmd
	switch command {
	case "init":
		cmd = exec.Command("git", "init")
	case "clone":
		if repo, ok := args["repo"].(string); ok {
			cmd = exec.Command("git", "clone", repo)
		} else {
			return map[string]any{"error": "repo parameter required for clone", "success": false}, false
		}
	case "status":
		cmd = exec.Command("git", "status")
	case "log":
		if lines, ok := args["lines"].(int); ok && lines > 0 {
			cmd = exec.Command("git", "log", fmt.Sprintf("--max-count=%d", lines))
		} else {
			cmd = exec.Command("git", "log")
		}
	case "diff":
		if args["unified"] != nil {
			unified, ok := args["unified"].(int)
			if !ok {
				return map[string]any{"error": "invalid unified parameter", "success": false}, false
			}
			cmd = exec.Command("git", "diff", fmt.Sprintf("-U%d", unified))
		} else {
			cmd = exec.Command("git", "diff")
		}
	default:
		return map[string]any{"error": fmt.Sprintf("unknown command: %s", command), "success": false}, false
	}

	output, err := cmd.CombinedOutput()
	exitCode := 0
	if err != nil {
		var exitErr *exec.ExitError
		if ok := asExitError(err, &exitErr); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 127
		}
	}

	return map[string]any{
		"output":    string(output),
		"success":   err == nil,
		"command":   fmt.Sprintf("git %s", command),
		"exit_code": exitCode,
	}, true
}

// asExitError extracts ExitError from error interface if present
func asExitError(err error, exitErr **exec.ExitError) bool {
	if exitErr == nil || err == nil {
		return false
	}
	var e *exec.ExitError
	if ok := okAsType(err, &e); ok {
		*exitErr = e
		return true
	}
	return false
}

// okAsType is a type assertion helper that works for any error interface type
func okAsType(err error, out interface{}) bool {
	v := reflect.ValueOf(out).Elem()
	e := reflect.TypeOf((*error)(nil)).Elem()
	if v.Type().AssignableTo(e) {
		defer func() {
			if r := recover(); r != nil {
				v.Set(reflect.Zero(v.Type()))
			}
		}()
		if ok, _ := okType(err, v.Interface()); ok {
			return true
		}
	}
	return false
}

// okType checks if error matches a specific type using reflection
func okType(err error, target interface{}) (bool, interface{}) {
	targetType := reflect.TypeOf(target)
	errType := reflect.TypeOf(err)
	return errType.AssignableTo(targetType), reflect.ValueOf(target).Interface()
}
