package main

import (
	"strings"
	"testing"
)

// Test cases for read_file tool edge cases
func TestReadFileToolEdgeCases(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		content        string
		offset         int
		limit          int
		expectContains string
		expectError    bool
	}{
		{
			name:           "read existing file",
			path:           "main.go",
			offset:         1,
			limit:          50,
			expectContains: "// YOLO - Your Own Living Operator",
			expectError:    false,
		},
		{
			name:      "read with offset",
			path:      "main.go",
			offset:    100,
			limit:     50,
			expectContains: "", // Just check it doesn't error
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := NewYoloAgent()
		
			result := agent.tools.readFile(map[string]any{
				"path":   tt.path,
				"offset": float64(tt.offset),
				"limit":  float64(tt.limit),
			})
		
			if tt.expectError {
				if !isError(result) {
					t.Errorf("Expected error but got: %s", result)
				}
			} else {
				if isError(result) {
					t.Errorf("Unexpected error: %s", result)
				} else if tt.expectContains != "" && !strings.Contains(result, tt.expectContains) {
					t.Errorf("Result does not contain expected text '%s'. Got: %s", tt.expectContains, result)
				}
			}
		})
	}
}

// Test cases for write_file tool edge cases
func TestWriteFileToolEdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		content   string
		expectErr bool
	}{
		{
			name:      "write to valid path",
			path:      "test_output/test_write.txt",
			content:   "test content",
			expectErr: false,
		},
		{
			name:      "write empty content",
			path:      "test_output/empty.txt",
			content:   "",
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := NewYoloAgent()
		
			result := agent.tools.writeFile(map[string]any{
				"path":    tt.path,
				"content": tt.content,
			})
		
			if tt.expectErr {
				if !isError(result) {
					t.Errorf("Expected error but got: %s", result)
				}
			} else {
				if isError(result) {
					t.Errorf("Unexpected error: %s", result)
				}
			}
		})
	}
}

// Test cases for edit_file tool edge cases
func TestEditFileToolEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		oldText     string
		newText     string
		expectErr   bool
	}{
		{
			name:      "edit existing file",
			path:      "test_output/edit_test.txt",
			oldText:   "old content",
			newText:   "new content",
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := NewYoloAgent()
		
			// First create the file with old text
			agent.tools.writeFile(map[string]any{
				"path":    tt.path,
				"content": tt.oldText,
			})
		
			result := agent.tools.editFile(map[string]any{
				"path":     tt.path,
				"old_text": tt.oldText,
				"new_text": tt.newText,
			})
		
			if tt.expectErr {
				if !isError(result) {
					t.Errorf("Expected error but got: %s", result)
				}
			} else {
				if isError(result) {
					t.Errorf("Unexpected error: %s", result)
				}
			}
		})
	}
}

// Helper function to check if a result is an error
func isError(result string) bool {
	return len(result) > 0 && (strings.HasPrefix(result, "Error") || strings.HasPrefix(result, "error"))
}