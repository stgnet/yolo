package main

import (
	"os"
	"path/filepath"
	"testing"
)

// TestMakeDir tests the make_dir tool implementation
func TestMakeDir(t *testing.T) {
	tests := []struct {
		name        string
		args        map[string]any
		expectError bool
	}{
		{
			name:        "create simple directory",
			args:        map[string]any{"path": "test_new_dir"},
			expectError: false,
		},
		{
			name:        "create nested directory",
			args:        map[string]any{"path": "level1/level2/level3"},
			expectError: false,
		},
		{
			name:        "create directory with spaces",
			args:        map[string]any{"path": "my new dir"},
			expectError: false,
		},
		{
			name:        "missing path argument",
			args:        map[string]any{},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			executor := &ToolExecutor{baseDir: tmpDir}
			
			result := executor.makeDir(tt.args)
			
			if tt.expectError {
				if !isError(result) {
					t.Errorf("Expected error but got: %s", result)
				}
			} else {
				if isError(result) {
					t.Errorf("Unexpected error: %s", result)
					return
				}
				
				path := getStringArg(tt.args, "path", "")
				fullPath := filepath.Join(tmpDir, path)
				
				if _, err := os.Stat(fullPath); os.IsNotExist(err) {
					t.Errorf("Directory was not created: %s", fullPath)
				} else {
					info, _ := os.Stat(fullPath)
					if !info.IsDir() {
						t.Errorf("Created path is not a directory: %s", fullPath)
					}
				}
				
				// Verify .gitignore was created
				gitignorePath := filepath.Join(fullPath, ".gitignore")
				if _, err := os.Stat(gitignorePath); os.IsNotExist(err) {
					t.Errorf(".gitignore was not created in: %s", fullPath)
				}
			}
		})
	}
}

// TestRemoveDir tests the remove_dir tool implementation  
func TestRemoveDir(t *testing.T) {
	tests := []struct {
		name        string
		setupAction func(string) // Setup before removal
		args        map[string]any
		expectError bool
	}{
		{
			name:        "remove empty directory",
			setupAction: func(dir string) { os.MkdirAll(filepath.Join(dir, "empty"), 0755) },
			args:        map[string]any{"path": "empty"},
			expectError: false,
		},
		{
			name:        "remove directory with files",
			setupAction: func(dir string) {
				dirPath := filepath.Join(dir, "with_files")
				os.MkdirAll(dirPath, 0755)
				os.WriteFile(filepath.Join(dirPath, "file.txt"), []byte("test"), 0644)
			},
			args:        map[string]any{"path": "with_files"},
			expectError: false,
		},
		{
			name:        "remove nested directory",
			setupAction: func(dir string) {
				dirPath := filepath.Join(dir, "level1/level2")
				os.MkdirAll(dirPath, 0755)
				os.WriteFile(filepath.Join(dirPath, "nested.txt"), []byte("test"), 0644)
			},
			args:        map[string]any{"path": "level1"},
			expectError: false,
		},
		{
			name:        "missing path argument",
			setupAction: func(dir string) {},
			args:        map[string]any{},
			expectError: true,
		},
		{
			name:        "remove non-existent directory",
			setupAction: func(dir string) {},
			args:        map[string]any{"path": "does_not_exist"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			
			if tt.setupAction != nil {
				tt.setupAction(tmpDir)
			}
			
			executor := &ToolExecutor{baseDir: tmpDir}
			result := executor.removeDir(tt.args)
			
			if tt.expectError {
				if !isError(result) {
					t.Errorf("Expected error but got: %s", result)
				}
			} else {
				if isError(result) {
					t.Errorf("Unexpected error: %s", result)
					return
				}
				
				path := getStringArg(tt.args, "path", "")
				fullPath := filepath.Join(tmpDir, path)
				
				if _, err := os.Stat(fullPath); !os.IsNotExist(err) {
					t.Errorf("Directory was not removed: %s", fullPath)
				}
			}
		})
	}
}

// Helper function to detect error messages
func isError(msg string) bool {
	return len(msg) >= 6 && msg[:5] == "Error"
}