// Package errors provides custom error types and utilities for consistent error handling.
package errors

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

// Error types for different categories of failures
type (
	// FileNotFoundError indicates a file or directory was not found
	FileNotFoundError struct {
		Path   string
		Op     string
		Cause  error
	}

	// ToolExecutionError indicates a tool command failed to execute
	ToolExecutionError struct {
		Tool    string
		Command string
		Output  string
		ExitCode int
		Cause   error
	}

	// ConfigurationError indicates a configuration issue
	ConfigurationError struct {
		Field string
		Value any
		Cause error
	}

	// NetworkError indicates a network-related failure
	NetworkError struct {
		URL     string
		Method  string
		Timeout bool
		Cause   error
	}

	// JSONError indicates JSON marshaling/unmarshaling failure
	JSONError struct {
		Operation string
		Data      any
		Cause     error
	}
)

// Error implements the error interface for FileNotFoundError
func (e *FileNotFoundError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s on %s: %v", e.Op, e.Path, e.Cause)
	}
	return fmt.Sprintf("%s on %s: file not found", e.Op, e.Path)
}

// Unwrap returns the wrapped error for errors.Is/As support
func (e *FileNotFoundError) Unwrap() error { return e.Cause }

// Error implements the error interface for ToolExecutionError
func (e *ToolExecutionError) Error() string {
	msg := fmt.Sprintf("tool %s failed: exit code %d", e.Tool, e.ExitCode)
	if e.Command != "" {
		msg += fmt.Sprintf(" (%s)", e.Command)
	}
	if e.Output != "" {
		lines := strings.Split(strings.TrimSpace(e.Output), "\n")
		if len(lines) > 0 {
			msg += ": " + lines[0]
		}
	}
	return msg
}

// Unwrap returns the wrapped error
func (e *ToolExecutionError) Unwrap() error { return e.Cause }

// Error implements the error interface for ConfigurationError
func (e *ConfigurationError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("configuration error for %s=%v: %v", e.Field, e.Value, e.Cause)
	}
	return fmt.Sprintf("configuration error for %s=%v", e.Field, e.Value)
}

// Unwrap returns the wrapped error
func (e *ConfigurationError) Unwrap() error { return e.Cause }

// Error implements the error interface for NetworkError
func (e *NetworkError) Error() string {
	msg := fmt.Sprintf("network error")
	if e.Method != "" && e.URL != "" {
		msg += fmt.Sprintf(" %s %s", e.Method, e.URL)
	}
	if e.Timeout {
		msg += " (timeout)"
	}
	if e.Cause != nil {
		msg += fmt.Sprintf(": %v", e.Cause)
	}
	return msg
}

// Unwrap returns the wrapped error
func (e *NetworkError) Unwrap() error { return e.Cause }

// Error implements the error interface for JSONError
func (e *JSONError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("JSON %s failed: %v", e.Operation, e.Cause)
	}
	return fmt.Sprintf("JSON %s failed", e.Operation)
}

// Unwrap returns the wrapped error
func (e *JSONError) Unwrap() error { return e.Cause }

// NewFileNotFoundError creates a new FileNotFoundError
func NewFileNotFoundError(op, path string, cause error) *FileNotFoundError {
	return &FileNotFoundError{Op: op, Path: path, Cause: cause}
}

// NewToolExecutionError creates a new ToolExecutionError
func NewToolExecutionError(tool, command, output string, exitCode int, cause error) *ToolExecutionError {
	return &ToolExecutionError{Tool: tool, Command: command, Output: output, ExitCode: exitCode, Cause: cause}
}

// NewConfigurationError creates a new ConfigurationError
func NewConfigurationError(field string, value any, cause error) *ConfigurationError {
	return &ConfigurationError{Field: field, Value: value, Cause: cause}
}

// NewNetworkError creates a new NetworkError
func NewNetworkError(method, url string, timeout bool, cause error) *NetworkError {
	return &NetworkError{Method: method, URL: url, Timeout: timeout, Cause: cause}
}

// NewJSONError creates a new JSONError
func NewJSONError(operation string, data any, cause error) *JSONError {
	return &JSONError{Operation: operation, Data: data, Cause: cause}
}

// Wrap wraps an error with additional context using custom error types
func Wrap(err error, typ ErrorType, context map[string]any) error {
	if err == nil {
		return nil
	}

	switch typ {
	case FileType:
		return &FileNotFoundError{
			Path:  getString(context, "path"),
			Op:    getString(context, "op"),
			Cause: err,
		}
	case ToolType:
		return &ToolExecutionError{
			Tool:     getString(context, "tool"),
			Command:  getString(context, "command"),
			Output:   getString(context, "output"),
			ExitCode: getInt(context, "exitCode", -1),
			Cause:    err,
		}
	case ConfigType:
		return &ConfigurationError{
			Field: getString(context, "field"),
			Value: context["value"],
			Cause: err,
		}
	case NetworkType:
		return &NetworkError{
			URL:     getString(context, "url"),
			Method:  getString(context, "method"),
			Timeout: getBool(context, "timeout", false),
			Cause:   err,
		}
	case JSONType:
		return &JSONError{
			Operation: getString(context, "operation"),
			Data:      context["data"],
			Cause:     err,
		}
	default:
		return err
	}
}

// ErrorType represents the category of error
type ErrorType string

const (
	FileType    ErrorType = "file"
	ToolType    ErrorType = "tool"
	ConfigType  ErrorType = "config"
	NetworkType ErrorType = "network"
	JSONType    ErrorType = "json"
)

// Helper functions for context map extraction
func getString(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getInt(m map[string]any, key string, defaultVal int) int {
	if v, ok := m[key]; ok {
		switch n := v.(type) {
		case int:
			return n
		case int64:
			return int(n)
		case int32:
			return int(n)
		}
	}
	return defaultVal
}

func getBool(m map[string]any, key string, defaultVal bool) bool {
	if v, ok := m[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return defaultVal
}

// Is* functions for errors.Is compatibility
func IsFileNotFoundError(err error) bool {
	var fnfe *FileNotFoundError
	return errors.As(err, &fnfe)
}

func IsToolExecutionError(err error) bool {
	var tee *ToolExecutionError
	return errors.As(err, &tee)
}

func IsConfigurationError(err error) bool {
	var ce *ConfigurationError
	return errors.As(err, &ce)
}

func IsNetworkError(err error) bool {
	var ne *NetworkError
	return errors.As(err, &ne)
}

func IsJSONError(err error) bool {
	var je *JSONError
	return errors.As(err, &je)
}

// As* functions for extracting typed errors
func AsFileNotFoundError(err error) (*FileNotFoundError, bool) {
	var fnfe *FileNotFoundError
	if errors.As(err, &fnfe) {
		return fnfe, true
	}
	return nil, false
}

func AsToolExecutionError(err error) (*ToolExecutionError, bool) {
	var tee *ToolExecutionError
	if errors.As(err, &tee) {
		return tee, true
	}
	return nil, false
}

// WithContext wraps an os.PathError as a FileNotFoundError
func WithContext(err error, op, path string) error {
	if pe, ok := err.(*os.PathError); ok {
		return NewFileNotFoundError(pe.Op, pe.Path, pe.Err)
	}
	return fmt.Errorf("%s %s: %w", op, path, err)
}
