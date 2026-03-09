package main

import (
	"os"
	"path/filepath"
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
			name:           "read with offset",
			path:           "main.go",
			offset:         100,
			limit:          50,
			expectContains: "", // Just check it doesn't error
			expectError:    false,
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
		name      string
		path      string
		oldText   string
		newText   string
		expectErr bool
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
		name        string
		pattern     string
		expectCount int // minimum expected count
		expectError bool
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
		name        string
		query       string
		pattern     string
		expectMatch bool
		expectError bool
	}{
		{
			name:        "search for package main",
			query:       "package main",
			pattern:     "*.go",
			expectMatch: true,
			expectError: false,
		},
		{
			name:        "search for non-existent pattern",
			query:       "THIS_UNIQUE_STRING_WILL_NOT_MATCH_ANYTHING_12345",
			pattern:     "*.go",
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
			name:         "run echo command",
			command:      "echo hello",
			expectError:  false,
			expectOutput: true,
		},
		{
			name:         "run pwd command",
			command:      "pwd",
			expectError:  false,
			expectOutput: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			agent := NewYoloAgent()
			agent.baseDir = tmpDir  // Update base dir for test isolation

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

// TestMakeDir tests the makeDir tool functionality
func TestMakeDir(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		expectError bool
	}{
		{"create simple dir", "test_dir_12345", false},
		{"create nested dirs", "nested/deep/dir/67890", false},
		{"missing path param", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			agent := NewYoloAgent()
			agent.baseDir = tmpDir  // Update base dir for test isolation
			agent.tools = NewToolExecutor(tmpDir, agent)  // Recreate tool executor with new baseDir

			result := agent.tools.makeDir(map[string]any{
				"path": tt.path,
			})

			if tt.expectError {
				if !isError(result) {
					t.Errorf("Expected error but got: %s", result)
				}
			} else {
				if isError(result) {
					t.Errorf("Unexpected error: %s", result)
				}
				
				// Verify directory was created using agent's baseDir
				fullPath := filepath.Join(agent.baseDir, tt.path)
				info, err := os.Stat(fullPath)
				if err != nil {
					t.Errorf("Directory not created at %s: %v", fullPath, err)
				} else if !info.IsDir() {
					t.Errorf("Created file instead of directory")
				}
			}
		})
	}
}

// TestRemoveDir tests the removeDir tool functionality
func TestRemoveDir(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func(tmpDir string, agent *YoloAgent) string
		expectError bool
	}{
		{
			name: "remove empty dir",
			setupFunc: func(tmpDir string, agent *YoloAgent) string {
				dirPath := "empty_dir_abcde"
				agent.tools.makeDir(map[string]any{"path": dirPath})
				return dirPath
			},
			expectError: false,
		},
		{
			name: "remove dir with files",
			setupFunc: func(tmpDir string, agent *YoloAgent) string {
				dirPath := "dir_with_files_fghij"
				agent.tools.makeDir(map[string]any{"path": dirPath})
				filePath := filepath.Join(dirPath, "file.txt")
				agent.tools.writeFile(map[string]any{
					"path": filePath,
					"data": []string{"test content"},
				})
				return dirPath
			},
			expectError: false,
		},
		{
			name:        "remove non-existent dir",
			setupFunc:   func(tmpDir string, agent *YoloAgent) string { return "non_existent_xyz" },
			expectError: true,
		},
		{
			name:        "missing path param",
			setupFunc:   func(tmpDir string, agent *YoloAgent) string { return "" },
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			agent := NewYoloAgent()
			agent.baseDir = tmpDir  // Update base dir for test isolation
			agent.tools = NewToolExecutor(tmpDir, agent)  // Recreate tool executor with new baseDir

			dirPath := tt.setupFunc(tmpDir, agent)
			
			result := agent.tools.removeDir(map[string]any{
				"path": dirPath,
			})

			if tt.expectError {
				if !isError(result) {
					t.Errorf("Expected error but got: %s", result)
				}
			} else {
				if isError(result) {
					t.Errorf("Unexpected error: %s", result)
				}
				
				// Verify directory was removed using agent's baseDir
				fullPath := filepath.Join(agent.baseDir, dirPath)
				if _, err := os.Stat(fullPath); !os.IsNotExist(err) {
					t.Errorf("Directory still exists after removal")
				}
			}
		})
	}
}

// TestMakeAndRemoveDirIntegration tests makeDir and removeDir working together
func TestMakeAndRemoveDirIntegration(t *testing.T) {
	tmpDir := t.TempDir()
	agent := NewYoloAgent()
	agent.baseDir = tmpDir  // Update base dir for test isolation
	agent.tools = NewToolExecutor(tmpDir, agent)  // Recreate tool executor with new baseDir

	// Create nested directory structure
	testPath := "integration_test_klmno/deep/nested/dir"
	result := agent.tools.makeDir(map[string]any{"path": testPath})
	if isError(result) {
		t.Fatalf("Failed to create directory: %s", result)
	}

	// Verify it exists using agent's baseDir
	fullPath := filepath.Join(agent.baseDir, testPath)
	if _, err := os.Stat(fullPath); err != nil {
		t.Fatalf("Directory not created: %v", err)
	}

	// Add a file in nested dir
	agent.tools.writeFile(map[string]any{
		"path": filepath.Join(testPath, "test.txt"),
		"data": []string{"integration test"},
	})

	// Remove parent directory (should remove everything)
	result = agent.tools.removeDir(map[string]any{"path": filepath.Dir(filepath.Dir(testPath))})
	if isError(result) {
		t.Fatalf("Failed to remove directory: %s", result)
	}

	// Verify removal
	parentPath := filepath.Join(tmpDir, filepath.Dir(filepath.Dir(testPath)))
	if _, err := os.Stat(parentPath); !os.IsNotExist(err) {
		t.Errorf("Parent directory and contents not fully removed")
	}
}
