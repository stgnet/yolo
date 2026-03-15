package main

import (
	"fmt"
	"os/exec"
	"reflect"
)

// GitExecutor provides git command execution capabilities
type GitExecutor struct {
	path string
}

// NewToolExecutor creates a GitExecutor instance with the given base path
func NewToolExecutor(basePath string, config interface{}) *GitExecutor {
	return &GitExecutor{path: basePath}
}

// GetPath returns the base path of this executor
func (ge *GitExecutor) GetPath() string {
	return ge.path
}

// gitListBranches lists all branches in the repository
func (ge *GitExecutor) gitListBranches(params map[string]any) (interface{}, error) {
	cmd := exec.Command("git", "branch")
	if ge.path != "" {
		cmd.Dir = ge.path
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, &ExecutionError{Err: err, Output: string(output)}
	}
	return string(output), nil
}

// gitDiff shows differences in the repository
func (ge *GitExecutor) gitDiff(params map[string]any) (interface{}, error) {
	args := []string{"diff"}
	if opt, ok := params["options"].(string); ok && opt != "" {
		args = append(args, opt)
	}
	cmd := exec.Command("git", args...)
	if ge.path != "" {
		cmd.Dir = ge.path
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, &ExecutionError{Err: err, Output: string(output)}
	}
	return string(output), nil
}

// gitStatus shows repository status
func (ge *GitExecutor) gitStatus(params map[string]any) (interface{}, error) {
	args := []string{"status"}
	cmd := exec.Command("git", args...)
	if ge.path != "" {
		cmd.Dir = ge.path
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, &ExecutionError{Err: err, Output: string(output)}
	}
	return string(output), nil
}

// asExitError converts interface{} to *ExitError if possible
func (ge *GitExecutor) asExitError(v interface{}) *ExitError {
	if v == nil {
		return nil
	}
	var err error
	switch val := v.(type) {
	case error:
		err = val
	case string:
		err = fmt.Errorf("%s", val)
	default:
		valType := reflect.TypeOf(v)
		valValue := reflect.ValueOf(v)
		if valType != nil && valValue.IsValid() {
			err = fmt.Errorf("unexpected type %v value %v", valType, valValue.Interface())
		} else {
			err = fmt.Errorf("nil or invalid value")
		}
	}
	return &ExitError{Err: err, ExitCode: 1}
}

// toInt converts interface{} to int64
func (ge *GitExecutor) toInt(v interface{}) (int64, error) {
	if v == nil {
		return 0, fmt.Errorf("cannot convert nil to int")
	}
	switch val := v.(type) {
	case int:
		return int64(val), nil
	case int8:
		return int64(val), nil
	case int16:
		return int64(val), nil
	case int32:
		return int64(val), nil
	case int64:
		return val, nil
	case uint:
		return int64(val), nil
	case uint8:
		return int64(val), nil
	case uint16:
		return int64(val), nil
	case uint32:
		return int64(val), nil
	case uint64:
		if val > 9223372036854775807 {
			return 0, fmt.Errorf("uint64 %d overflows int64", val)
		}
		return int64(val), nil
	case float32:
		return int64(val), nil
	case float64:
		if val > 9223372036854775807 || val < -9223372036854775808 {
			return 0, fmt.Errorf("float64 %f overflows int64", val)
		}
		return int64(val), nil
	case string:
		// Try to parse as number
		// (simplified - would use strconv.ParseInt in production)
		return 0, fmt.Errorf("string '%s' cannot be converted to int", val)
	default:
		return 0, fmt.Errorf("cannot convert type %T to int64", v)
	}
}
