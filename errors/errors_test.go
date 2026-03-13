package errors

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
)

// Helper assertions
func assertEqual(t *testing.T, got, want interface{}, msg string) {
	t.Helper()
	if got != want {
		t.Errorf("%s: got %v, want %v", msg, got, want)
	}
}

func assertTrue(t *testing.T, cond bool, msg string) {
	t.Helper()
	if !cond {
		t.Errorf("%s: expected true, got false", msg)
	}
}

func assertFalse(t *testing.T, cond bool, msg string) {
	t.Helper()
	if cond {
		t.Errorf("%s: expected false, got true", msg)
	}
}

func assertContains(t *testing.T, s, substr string, msg string) {
	t.Helper()
	if !strings.Contains(s, substr) {
		t.Errorf("%s: %q does not contain %q", msg, s, substr)
	}
}

func assertNotNil(t *testing.T, v interface{}, msg string) {
	t.Helper()
	if v == nil {
		t.Errorf("%s: expected non-nil", msg)
	}
}

// Test error constructors
func TestNewFileNotFoundError(t *testing.T) {
	err := NewFileNotFoundError("read", "test.txt", os.ErrNotExist)
	assertNotNil(t, err, "error should not be nil")
	assertContains(t, err.Error(), "test.txt", "should contain filename")
	assertContains(t, err.Error(), "read", "should contain operation")
}

func TestNewToolExecutionError(t *testing.T) {
	err := NewToolExecutionError("ls", "ls -la", "permission denied", 1, fmt.Errorf("exec failed"))
	assertNotNil(t, err, "error should not be nil")
	assertContains(t, err.Error(), "ls", "should contain tool name")
	assertContains(t, err.Error(), "1", "should contain exit code")
}

func TestNewConfigurationError(t *testing.T) {
	err := NewConfigurationError("timeout", 30, fmt.Errorf("invalid value"))
	assertNotNil(t, err, "error should not be nil")
	assertContains(t, err.Error(), "timeout", "should contain field name")
	assertContains(t, err.Error(), "30", "should contain value")
}

func TestNewNetworkError(t *testing.T) {
	err := NewNetworkError("GET", "http://example.com", true, fmt.Errorf("connection refused"))
	assertNotNil(t, err, "error should not be nil")
	assertContains(t, err.Error(), "GET", "should contain method")
	assertContains(t, err.Error(), "http://example.com", "should contain URL")
}

func TestNewJSONError(t *testing.T) {
	data := map[string]string{"key": "value"}
	err := NewJSONError("unmarshal", data, fmt.Errorf("invalid char"))
	assertNotNil(t, err, "error should not be nil")
	assertContains(t, err.Error(), "unmarshal", "should contain operation")
}

// Test Is* functions
func TestIsFileNotFoundError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"file not found", NewFileNotFoundError("read", "test.txt", os.ErrNotExist), true},
		{"tool error", NewToolExecutionError("ls", "", "", 0, nil), false},
		{"regular error", fmt.Errorf("not file error"), false},
		{"nil error", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsFileNotFoundError(tt.err)
			assertEqual(t, result, tt.expected, "IsFileNotFoundError should return correct value")
		})
	}
}

func TestIsToolExecutionError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"tool error", NewToolExecutionError("my_tool", "cmd", "", 1, nil), true},
		{"file error", NewFileNotFoundError("open", "x.txt", os.ErrNotExist), false},
		{"regular error", fmt.Errorf("not tool"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsToolExecutionError(tt.err)
			assertEqual(t, result, tt.expected, "IsToolExecutionError should return correct value")
		})
	}
}

func TestIsConfigurationError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"config error", NewConfigurationError("field", "value", nil), true},
		{"regular error", fmt.Errorf("not config"), false},
		{"nil error", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsConfigurationError(tt.err)
			assertEqual(t, result, tt.expected, "IsConfigurationError should return correct value")
		})
	}
}

func TestIsNetworkError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"network error", NewNetworkError("GET", "http://x.com", false, nil), true},
		{"regular error", fmt.Errorf("not network"), false},
		{"nil error", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsNetworkError(tt.err)
			assertEqual(t, result, tt.expected, "IsNetworkError should return correct value")
		})
	}
}

func TestIsJSONError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"json error", NewJSONError("parse", nil, fmt.Errorf("invalid")), true},
		{"regular error", fmt.Errorf("not json"), false},
		{"nil error", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsJSONError(tt.err)
			assertEqual(t, result, tt.expected, "IsJSONError should return correct value")
		})
	}
}

// Test WithContext
func TestWithContext_PathError(t *testing.T) {
	pe := &os.PathError{Op: "open", Path: "/test/file.txt", Err: os.ErrNotExist}
	wrapped := WithContext(pe, "ignored_context", "/ignored_path")

	assertTrue(t, IsFileNotFoundError(wrapped), "should wrap PathError as FileNotFoundError")
	var fnfe *FileNotFoundError
	if errors.As(wrapped, &fnfe) {
		assertEqual(t, fnfe.Op, "open", "should use PathError.Op, not context param")
		assertEqual(t, fnfe.Path, "/test/file.txt", "should use PathError.Path, not context param")
	}
}

func TestWithContext_NonPathError(t *testing.T) {
	err := fmt.Errorf("something went wrong")
	wrapped := WithContext(err, "context", "path")

	assertFalse(t, IsFileNotFoundError(wrapped), "should not be FileNotFoundError")
	assertContains(t, wrapped.Error(), "context", "should contain context")
}

// Test Wrap function
func TestWrap_FileType(t *testing.T) {
	baseErr := fmt.Errorf("original error")
	wrapped := Wrap(baseErr, FileType, map[string]any{
		"path": "/test/file.txt",
		"op":   "read",
	})

	assertTrue(t, IsFileNotFoundError(wrapped), "should be FileNotFoundError")
	var fnfe *FileNotFoundError
	if errors.As(wrapped, &fnfe) {
		assertEqual(t, fnfe.Path, "/test/file.txt", "should have correct path")
		assertEqual(t, fnfe.Op, "read", "should have correct op")
	}
}

func TestWrap_ToolType(t *testing.T) {
	baseErr := fmt.Errorf("command failed")
	wrapped := Wrap(baseErr, ToolType, map[string]any{
		"tool":     "docker",
		"command":  "run -it alpine",
		"output":   "permission denied",
		"exitCode": int64(125), // Test with int64 to cover getInt type conversion
	})

	assertTrue(t, IsToolExecutionError(wrapped), "should be ToolExecutionError")
	var tee *ToolExecutionError
	if errors.As(wrapped, &tee) {
		assertEqual(t, tee.Tool, "docker", "should have correct tool")
		assertEqual(t, tee.ExitCode, 125, "should have correct exit code")
	}
}

func TestWrap_ToolType_MissingExitCode(t *testing.T) {
	baseErr := fmt.Errorf("command failed")
	wrapped := Wrap(baseErr, ToolType, map[string]any{
		"tool":    "ls",
		"command": "ls -la",
	})

	assertTrue(t, IsToolExecutionError(wrapped), "should be ToolExecutionError")
	var tee *ToolExecutionError
	if errors.As(wrapped, &tee) {
		assertEqual(t, tee.ExitCode, -1, "should use default exit code when not provided")
	}
}

func TestWrap_ToolType_Int32ExitCode(t *testing.T) {
	baseErr := fmt.Errorf("command failed")
	wrapped := Wrap(baseErr, ToolType, map[string]any{
		"tool":     "grep",
		"command":  "grep pattern file.txt",
		"exitCode": int32(0), // Test int32 type conversion
	})

	assertTrue(t, IsToolExecutionError(wrapped), "should be ToolExecutionError")
	var tee *ToolExecutionError
	if errors.As(wrapped, &tee) {
		assertEqual(t, tee.ExitCode, 0, "should handle int32 exit code")
	}
}

func TestWrap_NetworkType(t *testing.T) {
	baseErr := fmt.Errorf("connection refused")
	wrapped := Wrap(baseErr, NetworkType, map[string]any{
		"url":     "http://example.com",
		"method":  "POST",
		"timeout": true, // Test getBool extraction
	})

	assertTrue(t, IsNetworkError(wrapped), "should be NetworkError")
	var ne *NetworkError
	if errors.As(wrapped, &ne) {
		assertEqual(t, ne.URL, "http://example.com", "should have correct URL")
		assertEqual(t, ne.Method, "POST", "should have correct method")
		assertTrue(t, ne.Timeout, "should have correct timeout flag")
	}
}

func TestWrap_NetworkType_MissingTimeout(t *testing.T) {
	baseErr := fmt.Errorf("connection refused")
	wrapped := Wrap(baseErr, NetworkType, map[string]any{
		"url":    "http://example.com",
		"method": "GET",
	})

	assertTrue(t, IsNetworkError(wrapped), "should be NetworkError")
	var ne *NetworkError
	if errors.As(wrapped, &ne) {
		assertFalse(t, ne.Timeout, "should use default timeout (false) when not provided")
	}
}

func TestWrap_JSONType(t *testing.T) {
	data := []byte(`{"invalid": json}`)
	baseErr := fmt.Errorf("unexpected end of JSON input")
	wrapped := Wrap(baseErr, JSONType, map[string]any{
		"operation": "unmarshal",
		"data":      data,
	})

	assertTrue(t, IsJSONError(wrapped), "should be JSONError")
	var je *JSONError
	if errors.As(wrapped, &je) {
		assertEqual(t, je.Operation, "unmarshal", "should have correct operation")
		assertNotNil(t, je.Data, "should preserve data")
	}
}

func TestWrap_ConfigType(t *testing.T) {
	baseErr := fmt.Errorf("invalid duration")
	wrapped := Wrap(baseErr, ConfigType, map[string]any{
		"field": "timeout",
		"value": "30s",
	})

	assertTrue(t, IsConfigurationError(wrapped), "should be ConfigurationError")
	var ce *ConfigurationError
	if errors.As(wrapped, &ce) {
		assertEqual(t, ce.Field, "timeout", "should have correct field")
		assertEqual(t, ce.Value, "30s", "should have correct value")
	}
}

func TestWrap_UnknownType(t *testing.T) {
	baseErr := fmt.Errorf("original error")
	wrapped := Wrap(baseErr, ErrorType("unknown"), map[string]any{})

	assertFalse(t, wrapped == nil, "should not return nil for unknown type")
	assertTrue(t, wrapped == baseErr, "should return original error for unknown type")
}

func TestWrap_NilError(t *testing.T) {
	wrapped := Wrap(nil, FileType, map[string]any{
		"path": "/test/file.txt",
		"op":   "read",
	})

	assertTrue(t, wrapped == nil, "should return nil for nil error")
}

// Test error chain unwrapping
func TestErrorsIsCompatibility(t *testing.T) {
	fileErr := NewFileNotFoundError("read", "test.txt", os.ErrNotExist)
	wrapped := fmt.Errorf("operation failed: %w", fileErr)

	assertTrue(t, errors.Is(wrapped, fileErr), "errors.Is should work")
	assertTrue(t, IsFileNotFoundError(wrapped), "IsFileNotFoundError should find wrapped error")
}

func TestErrorsAsCompatibility(t *testing.T) {
	fileErr := NewFileNotFoundError("read", "test.txt", os.ErrNotExist)
	wrapped := fmt.Errorf("operation failed: %w", fileErr)

	var target *FileNotFoundError
	if !errors.As(wrapped, &target) {
		t.Fatal("errors.As should find wrapped error")
	}
	assertEqual(t, target.Path, "test.txt", "should extract correct path")
}

func TestDeepWrapping(t *testing.T) {
	baseErr := NewNetworkError("GET", "http://api.example.com", true, fmt.Errorf("timeout"))
	wrapped1 := Wrap(baseErr, ToolType, map[string]any{
		"tool":    "curl",
		"command": "curl http://api.example.com",
	})
	wrapped2 := fmt.Errorf("outer: %w", wrapped1)

	assertTrue(t, IsNetworkError(wrapped2), "should detect network error through wraps")
	var ne *NetworkError
	if errors.As(wrapped2, &ne) {
		assertEqual(t, ne.URL, "http://api.example.com", "should extract URL")
	}
}

func TestUnwrapConfigurationError(t *testing.T) {
	baseErr := fmt.Errorf("original config error")
	configErr := NewConfigurationError("timeout", 30, baseErr)

	unwrapped := errors.Unwrap(configErr)
	assertTrue(t, unwrapped == baseErr, "Unwrap should return original error")
}

func TestUnwrapNetworkError(t *testing.T) {
	baseErr := fmt.Errorf("original network error")
	networkErr := NewNetworkError("POST", "http://api.com", false, baseErr)

	unwrapped := errors.Unwrap(networkErr)
	assertTrue(t, unwrapped == baseErr, "Unwrap should return original error")
}

func TestUnwrapJSONError(t *testing.T) {
	baseErr := fmt.Errorf("original json error")
	jsonErr := NewJSONError("marshal", map[string]string{}, baseErr)

	unwrapped := errors.Unwrap(jsonErr)
	assertTrue(t, unwrapped == baseErr, "Unwrap should return original error")
}

func TestUnwrapNilCause(t *testing.T) {
	configErr := NewConfigurationError("field", "value", nil)
	networkErr := NewNetworkError("GET", "http://x.com", false, nil)
	jsonErr := NewJSONError("parse", nil, nil)

	assertFalse(t, errors.Unwrap(configErr) != nil, "Unwrap with nil cause should return nil")
	assertFalse(t, errors.Unwrap(networkErr) != nil, "Unwrap with nil cause should return nil")
	assertFalse(t, errors.Unwrap(jsonErr) != nil, "Unwrap with nil cause should return nil")
}

// Test As* functions
func TestAsFileNotFoundError(t *testing.T) {
	fileErr := NewFileNotFoundError("read", "test.txt", os.ErrNotExist)

	result, ok := AsFileNotFoundError(fileErr)
	assertTrue(t, ok, "should find error")
	assertNotNil(t, result, "result should not be nil")
	assertEqual(t, result.Path, "test.txt", "should have correct path")
}

func TestAsToolExecutionError(t *testing.T) {
	toolErr := NewToolExecutionError("docker", "run", "output", 1, nil)

	result, ok := AsToolExecutionError(toolErr)
	assertTrue(t, ok, "should find error")
	assertNotNil(t, result, "result should not be nil")
	assertEqual(t, result.Tool, "docker", "should have correct tool")
}

func TestAsFileNotFoundError_WrongType(t *testing.T) {
	toolErr := NewToolExecutionError("docker", "run", "", 0, nil)

	result, ok := AsFileNotFoundError(toolErr)
	assertFalse(t, ok, "should not find error")
	assertNotNil(t, result == nil, "result should be nil")
}

// Test error type constants
func TestErrorTypeConstants(t *testing.T) {
	expectedTypes := []ErrorType{
		FileType, ToolType, ConfigType, NetworkType, JSONType,
	}

	for _, typ := range expectedTypes {
		assertContains(t, string(typ), "", "type should not be empty")
	}
}

// Test error messages
func TestErrorMessages(t *testing.T) {
	tests := []struct {
		err    error
		expect string
	}{
		{NewFileNotFoundError("read", "file.txt", os.ErrNotExist), "read on file.txt"},
		{NewToolExecutionError("tool", "cmd", "output", 1, nil), "tool tool failed: exit code 1"},
		{NewConfigurationError("field", "value", fmt.Errorf("cause")), "configuration error for field=value"},
		{NewNetworkError("GET", "http://x.com", true, nil), "network error GET http://x.com (timeout)"},
		{NewJSONError("marshal", nil, fmt.Errorf("invalid")), "JSON marshal failed: invalid"},
	}

	for _, tt := range tests {
		t.Run(tt.expect, func(t *testing.T) {
			assertContains(t, tt.err.Error(), tt.expect, "error message should contain expected text")
		})
	}
}

func TestErrorMessages_NoCause(t *testing.T) {
	tests := []struct {
		err    error
		expect string
	}{
		{NewFileNotFoundError("read", "file.txt", nil), "file not found"},
		{NewConfigurationError("field", "value", nil), "configuration error for field=value"},
		{NewJSONError("parse", nil, nil), "failed"},
	}

	for _, tt := range tests {
		t.Run(tt.expect, func(t *testing.T) {
			assertContains(t, tt.err.Error(), tt.expect, "error message should contain expected text")
		})
	}
}

// Test helper function edge cases for getInt
func TestGetInt_Int32Conversion(t *testing.T) {
	m := map[string]any{"code": int32(42)}
	result := getInt(m, "code", -1)
	assertEqual(t, result, 42, "should convert int32 correctly")
}

func TestGetInt_MissingKey_ReturnsDefault(t *testing.T) {
	m := map[string]any{"other": 10}
	result := getInt(m, "missing", -99)
	assertEqual(t, result, -99, "should return default value when key missing")
}

func TestGetInt_WrongType_ReturnsDefault(t *testing.T) {
	m := map[string]any{"code": "not a number"}
	result := getInt(m, "code", 0)
	assertEqual(t, result, 0, "should return default when type mismatch")
}

// Test AsToolExecutionError false case
func TestAsToolExecutionError_FalseCase(t *testing.T) {
	fileErr := NewFileNotFoundError("read", "test.txt", nil)
	result, ok := AsToolExecutionError(fileErr)
	assertFalse(t, ok, "should not find ToolExecutionError")
	assertEqual(t, result == nil, true, "result should be nil")
}

// Test getInt with int type (direct, not through Wrap)
func TestGetInt_IntType(t *testing.T) {
	m := map[string]any{"count": 42}
	result := getInt(m, "count", -1)
	assertEqual(t, result, 42, "should handle int type directly")
}

// Test getBool with false value
func TestGetBool_FalseValue(t *testing.T) {
	m := map[string]any{"flag": false}
	result := getBool(m, "flag", true)
	assertEqual(t, result, false, "should return actual false value, not default")
}

// Test getBool missing key returns default
func TestGetBool_MissingKey(t *testing.T) {
	m := map[string]any{"other": "value"}
	result := getBool(m, "flag", true)
	assertEqual(t, result, true, "should return default when key missing")
}

// Test getString with value
func TestGetString_WithValue(t *testing.T) {
	m := map[string]any{"name": "test"}
	result := getString(m, "name")
	assertEqual(t, result, "test", "should return string value")
}

// Test getString missing key
func TestGetString_MissingKey(t *testing.T) {
	m := map[string]any{"other": 123}
	result := getString(m, "missing")
	assertEqual(t, result, "", "should return empty string when key missing")
}

// Test getString wrong type
func TestGetString_WrongType(t *testing.T) {
	m := map[string]any{"count": 42}
	result := getString(m, "count")
	assertEqual(t, result, "", "should return empty string when type mismatch")
}
