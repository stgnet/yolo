package errors

import (
	"errors"
	"fmt"
	"os"
	"testing"
)

// Test NewHTTPError
func TestNewHTTPError(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		url        string
		statusCode int
		message    string
		cause      error
	}{
		{
			name:       "with message and cause",
			method:     "GET",
			url:        "http://api.example.com/users",
			statusCode: 404,
			message:    "not found",
			cause:      fmt.Errorf("connection failed"),
		},
		{
			name:       "message only without cause",
			method:     "POST",
			url:        "http://api.example.com/submit",
			statusCode: 400,
			message:    "bad request",
			cause:      nil,
		},
		{
			name:       "no message with cause",
			method:     "DELETE",
			url:        "http://api.example.com/resource/123",
			statusCode: 500,
			message:    "",
			cause:      fmt.Errorf("internal error"),
		},
		{
			name:       "minimal parameters",
			method:     "GET",
			url:        "http://example.com",
			statusCode: 200,
			message:    "",
			cause:      nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewHTTPError(tt.method, tt.url, tt.statusCode, tt.message, tt.cause)
			
			if err == nil {
				t.Fatal("expected non-nil error")
			}
			
			if !IsNetworkError(err) {
				t.Error("should be a NetworkError")
			}
			
			var ne *NetworkError
			if !errors.As(err, &ne) {
				t.Fatal("should extract as NetworkError")
			}
			
			if ne.Method != tt.method {
				t.Errorf("expected method %s, got %s", tt.method, ne.Method)
			}
			
			if ne.URL != tt.url {
				t.Errorf("expected URL %s, got %s", tt.url, ne.URL)
			}
		})
	}
}

// Test NewToolExecError
func TestNewToolExecError(t *testing.T) {
	tests := []struct {
		name    string
		tool    string
		command string
		output  any
		message string
		cause   error
	}{
		{
			name:    "string output with message and cause",
			tool:    "docker",
			command: "run -it alpine",
			output:  "permission denied",
			message: "container failed to start",
			cause:   fmt.Errorf("exit code 125"),
		},
		{
			name:    "byte slice output",
			tool:    "git",
			command: "commit -m 'test'",
			output:  []byte("no changes to commit"),
			message: "",
			cause:   nil,
		},
		{
			name:    "nil output with message only",
			tool:    "npm",
			command: "install",
			output:  nil,
			message: "network error",
			cause:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewToolExecError(tt.tool, tt.command, tt.output, tt.message, tt.cause)
			
			if err == nil {
				t.Fatal("expected non-nil error")
			}
			
			if !IsToolExecutionError(err) {
				t.Error("should be a ToolExecutionError")
			}
			
			var tee *ToolExecutionError
			if !errors.As(err, &tee) {
				t.Fatal("should extract as ToolExecutionError")
			}
			
			if tee.Tool != tt.tool {
				t.Errorf("expected tool %s, got %s", tt.tool, tee.Tool)
			}
			
			if tee.Command != tt.command {
				t.Errorf("expected command %s, got %s", tt.command, tee.Command)
			}
		})
	}
}

// Test TodoValidationError
func TestNewTodoValidationError(t *testing.T) {
	err := NewTodoValidationError("title", "Test Todo", map[string]string{
		"title": "title cannot be empty",
	})
	
	if err == nil {
		t.Fatal("expected non-nil error")
	}
	
	if err.Field != "title" {
		t.Errorf("expected field 'title', got %s", err.Field)
	}
	
	if err.Title != "Test Todo" {
		t.Errorf("expected title 'Test Todo', got %s", err.Title)
	}
	
	if len(err.Errors) != 1 {
		t.Errorf("expected 1 error, got %d", len(err.Errors))
	}
}

func TestTodoValidationError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *TodoValidationError
		expected string
	}{
		{
			name: "with field and errors map",
			err: &TodoValidationError{
				Field:  "title",
				Title:  "",
				Errors: map[string]string{"title": "required"},
			},
			expected: "validation failed for field 'title': title=\"required\"",
		},
		{
			name: "with title and errors map",
			err: &TodoValidationError{
				Field:  "",
				Title:  "My Todo",
				Errors: map[string]string{"status": "invalid"},
			},
			expected: "validation failed for todo 'My Todo': status=\"invalid\"",
		},
		{
			name: "minimal error",
			err: &TodoValidationError{
				Field:  "",
				Title:  "",
				Errors: nil,
			},
			expected: "validation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := tt.err.Error()
			if msg != tt.expected {
				t.Errorf("expected message %q, got %q", tt.expected, msg)
			}
		})
	}
}

func TestTodoValidationError_Unwrap(t *testing.T) {
	err := NewTodoValidationError("field", "title", map[string]string{"field": "error"})
	unwrapped := err.Unwrap()
	
	if unwrapped != nil {
		t.Errorf("expected nil unwrap, got %v", unwrapped)
	}
}

// Test TodoNotFoundError
func TestNewTodoNotFoundError(t *testing.T) {
	existing := []string{"Todo 1", "Todo 2"}
	err := NewTodoNotFoundError("Non-existent Todo", existing)
	
	if err == nil {
		t.Fatal("expected non-nil error")
	}
	
	if err.Title != "Non-existent Todo" {
		t.Errorf("expected title 'Non-existent Todo', got %s", err.Title)
	}
	
	if len(err.Existing) != 2 {
		t.Errorf("expected 2 existing todos, got %d", len(err.Existing))
	}
}

func TestTodoNotFoundError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *TodoNotFoundError
		expected string
	}{
		{
			name: "with existing todos",
			err: &TodoNotFoundError{
				Title:    "Missing Todo",
				Existing: []string{"Todo 1", "Todo 2"},
			},
			expected: "todo not found: Missing Todo. Existing todos: Todo 1 Todo 2",
		},
		{
			name:     "without existing todos",
			err:      &TodoNotFoundError{Title: "Missing Todo", Existing: nil},
			expected: "todo not found: Missing Todo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := tt.err.Error()
			if msg != tt.expected {
				t.Errorf("expected message %q, got %q", tt.expected, msg)
			}
		})
	}
}

// Test Wrap with additional edge cases
func TestWrap_FileType_MissingFields(t *testing.T) {
	baseErr := fmt.Errorf("original error")
	wrapped := Wrap(baseErr, FileType, map[string]any{})
	
	if !IsFileNotFoundError(wrapped) {
		t.Error("should be FileNotFoundError even with missing fields")
	}
	
	var fnfe *FileNotFoundError
	if errors.As(wrapped, &fnfe) {
		if fnfe.Path != "" {
			t.Errorf("expected empty path, got %q", fnfe.Path)
		}
		if fnfe.Op != "" {
			t.Errorf("expected empty op, got %q", fnfe.Op)
		}
	}
}

// Test ToolExecutionError with various output formats
func TestToolExecutionError_VariousOutputs(t *testing.T) {
	tests := []struct {
		name   string
		output string
	}{
		{"empty output", ""},
		{"single line", "error message"},
		{"multiline", "line1\nline2\nline3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewToolExecutionError("test_tool", "command", tt.output, 1, nil)
			msg := err.Error()
			
			if !containsString(msg, "test_tool") {
				t.Errorf("error should contain tool name: %q", msg)
			}
			
			if tt.output != "" && len(tt.output) > 0 {
				firstLine := splitFirstLine(tt.output)
				if !containsString(msg, firstLine) {
					t.Errorf("error should contain first line of output: got %q", msg)
				}
			}
		})
	}
}

// Helper functions for tests
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || 
		(len(s) > 0 && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || 
		findSubstring(s, substr))))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func splitFirstLine(s string) string {
	for i, c := range s {
		if c == '\n' {
			return s[:i]
		}
	}
	return s
}

// Test Wrap ConfigType with different value types
func TestWrap_ConfigType_ValueTypes(t *testing.T) {
	tests := []struct {
		name  string
		value any
	}{
		{"string value", "30s"},
		{"int value", 42},
		{"bool value", true},
		{"float value", 3.14},
		{"nil value", nil},
		{"map value", map[string]string{"key": "val"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseErr := fmt.Errorf("invalid config")
			wrapped := Wrap(baseErr, ConfigType, map[string]any{
				"field": "test_field",
				"value": tt.value,
			})
			
			if !IsConfigurationError(wrapped) {
				t.Error("should be ConfigurationError")
			}
			
			var ce *ConfigurationError
			if errors.As(wrapped, &ce) {
				if ce.Field != "test_field" {
					t.Errorf("expected field 'test_field', got %s", ce.Field)
				}
			}
		})
	}
}

// Test Wrap with os.PathError wrapped in fmt.Errorf
func TestWrap_PathErrorWrapped(t *testing.T) {
	pe := &os.PathError{Op: "open", Path: "/test.txt", Err: os.ErrNotExist}
	wrapped := fmt.Errorf("failed to open: %w", pe)
	
	result := Wrap(wrapped, FileType, map[string]any{
		"path": "/new-path.txt",
		"op":   "read",
	})
	
	if !IsFileNotFoundError(result) {
		t.Error("should be FileNotFoundError")
	}
	
	var fnfe *FileNotFoundError
	if errors.As(result, &fnfe) {
		// The wrap should preserve the original error's chain
		if fnfe.Cause != wrapped {
			t.Error("cause should be the wrapped error")
		}
	}
}
