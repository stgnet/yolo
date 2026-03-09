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

// Test cases for list_files tool edge cases
func TestListFilesToolEdgeCases(t *testing.T) {
	tests := []struct {
		name         string
		pattern      string
		expectCount  int // minimum expected count
		expectError  bool
	}{
		{
			name:        "list all go files",
			pattern:     "*.go",
			expectCount: 5,
			expectError: false,
		},
		{
			name:        "list main files",
			pattern:     "main.*",
			expectCount: 2,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := NewYoloAgent()

			result := agent.tools.listFiles(map[string]any{
				"pattern": tt.pattern,
			})

			if tt.expectError {
				if !isError(result) {
					t.Errorf("Expected error but got: %s", result)
				}
			} else {
				if isError(result) {
					t.Errorf("Unexpected error: %s", result)
				} else if result == "(no matching files or directories)" && tt.expectCount > 0 {
					t.Errorf("Expected at least %d matches but found none", tt.expectCount)
				}
			}
		})
	}
}

// Test cases for search_files tool edge cases
func TestSearchFilesToolEdgeCases(t *testing.T) {
	tests := []struct {
		name         string
		query        string
		pattern      string
		expectMatch  bool
		expectError  bool
	}{
		{
			name:      "search for package main",
			query:     "package main",
			pattern:   "*.go",
			expectMatch: true,
			expectError: false,
		},
		{
			name:      "search for non-existent pattern",
			query:     "THIS_UNIQUE_STRING_WILL_NOT_MATCH_ANYTHING_12345",
			pattern:   "*.go",
			expectMatch: false,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := NewYoloAgent()

			result := agent.tools.searchFiles(map[string]any{
				"query":   tt.query,
				"pattern": tt.pattern,
			})

			if tt.expectError {
				if !isError(result) {
					t.Errorf("Expected error but got: %s", result)
				}
			} else {
				if isError(result) {
					t.Errorf("Unexpected error: %s", result)
				} else if tt.expectMatch && strings.Contains(result, "No matches found") {
					t.Errorf("Expected match but none found")
				}
			}
		})
	}
}

// Test cases for run_command tool
func TestRunCommandToolEdgeCases(t *testing.T) {
	tests := []struct {
		name         string
		command      string
		expectError  bool
		expectOutput bool
	}{
		{
			name:        "run echo command",
			command:     "echo hello",
			expectError: false,
			expectOutput: true,
		},
		{
			name:        "run pwd command",
			command:     "pwd",
			expectError: false,
			expectOutput: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := NewYoloAgent()

			result := agent.tools.runCommand(map[string]any{
				"command": tt.command,
			})

			if tt.expectError {
				if !isError(result) {
					t.Errorf("Expected error but got: %s", result)
				}
			} else {
				if isError(result) {
					t.Errorf("Unexpected error: %s", result)
				} else if tt.expectOutput && strings.TrimSpace(result) == "" {
					t.Errorf("Expected output but got none")
				}
			}
		})
	}
}

// Helper function to check if a result is an error
func isError(result string) bool {
	return len(result) > 0 && (strings.HasPrefix(result, "Error") || strings.HasPrefix(result, "error"))
}
