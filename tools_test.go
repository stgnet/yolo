package main

import (
	"os"
	"path/filepath"
	"strings"
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

// TestMoveFile tests the move_file tool implementation
func TestMoveFile(t *testing.T) {
	tests := []struct {
		name        string
		setupAction func(string) // Setup before test
		args        map[string]any
		expectError bool
		checkFunc   func(*testing.T, string, string) // Additional validation
	}{
		{
			name: "move file to new name in same directory",
			setupAction: func(dir string) {
				srcFile := filepath.Join(dir, "source.txt")
				os.WriteFile(srcFile, []byte("test content"), 0644)
			},
			args: map[string]any{
				"source": "source.txt",
				"dest":   "destination.txt",
			},
			expectError: false,
			checkFunc: func(t *testing.T, tmpDir, result string) {
				// Verify source no longer exists
				srcPath := filepath.Join(tmpDir, "source.txt")
				if _, err := os.Stat(srcPath); !os.IsNotExist(err) {
					t.Errorf("Source file still exists: %s", srcPath)
				}
				
				// Verify destination exists with content
				destPath := filepath.Join(tmpDir, "destination.txt")
				content, err := os.ReadFile(destPath)
				if err != nil {
					t.Errorf("Destination file does not exist: %s", destPath)
					return
				}
				if string(content) != "test content" {
					t.Errorf("Content mismatch after move")
				}
			},
		},
		{
			name: "move file to different directory",
			setupAction: func(dir string) {
				srcFile := filepath.Join(dir, "source.txt")
				os.WriteFile(srcFile, []byte("test content"), 0644)
				os.MkdirAll(filepath.Join(dir, "subdir"), 0755)
			},
			args: map[string]any{
				"source": "source.txt",
				"dest":   "subdir/moved.txt",
			},
			expectError: false,
			checkFunc: func(t *testing.T, tmpDir, result string) {
				// Verify source no longer exists
				srcPath := filepath.Join(tmpDir, "source.txt")
				if _, err := os.Stat(srcPath); !os.IsNotExist(err) {
					t.Errorf("Source file still exists: %s", srcPath)
				}
				
				// Verify destination exists
				destPath := filepath.Join(tmpDir, "subdir/moved.txt")
				if _, err := os.Stat(destPath); os.IsNotExist(err) {
					t.Errorf("Destination file does not exist: %s", destPath)
				}
			},
		},
		{
			name: "move file with auto-create destination directory",
			setupAction: func(dir string) {
				srcFile := filepath.Join(dir, "source.txt")
				os.WriteFile(srcFile, []byte("test content"), 0644)
			},
			args: map[string]any{
				"source": "source.txt",
				"dest":   "new/nested/dir/file.txt",
			},
			expectError: false,
			checkFunc: func(t *testing.T, tmpDir, result string) {
				destPath := filepath.Join(tmpDir, "new/nested/dir/file.txt")
				if _, err := os.Stat(destPath); os.IsNotExist(err) {
					t.Errorf("Destination file does not exist: %s", destPath)
				}
			},
		},
		{
			name: "missing source argument",
			setupAction: func(dir string) {},
			args: map[string]any{
				"dest": "destination.txt",
			},
			expectError: true,
			checkFunc: nil,
		},
		{
			name: "missing dest argument",
			setupAction: func(dir string) {
				srcFile := filepath.Join(dir, "source.txt")
				os.WriteFile(srcFile, []byte("test content"), 0644)
			},
			args: map[string]any{
				"source": "source.txt",
			},
			expectError: true,
			checkFunc: nil,
		},
		{
			name: "move non-existent source file",
			setupAction: func(dir string) {},
			args: map[string]any{
				"source": "non_existent.txt",
				"dest":   "destination.txt",
			},
			expectError: true,
			checkFunc: nil,
		},
		{
			name: "move directory instead of file should fail",
			setupAction: func(dir string) {
				os.MkdirAll(filepath.Join(dir, "mydir"), 0755)
			},
			args: map[string]any{
				"source": "mydir",
				"dest":   "moved_dir",
			},
			expectError: true,
			checkFunc: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
		
			if tt.setupAction != nil {
				tt.setupAction(tmpDir)
			}
		
			executor := &ToolExecutor{baseDir: tmpDir}
			result := executor.moveFile(tt.args)
		
			if tt.expectError {
				if !isError(result) {
					t.Errorf("Expected error but got: %s", result)
				}
			} else {
				if isError(result) {
					t.Errorf("Unexpected error: %s", result)
					return
				}
				
				// Check that result contains move confirmation
				if !strings.Contains(strings.ToLower(result), "moved") {
					t.Errorf("Result should contain 'moved': %s", result)
				}
			
				if tt.checkFunc != nil {
					tt.checkFunc(t, tmpDir, result)
				}
			}
		})
	}
}

// Helper function to detect error messages
func isError(msg string) bool {
	return len(msg) >= 6 && msg[:5] == "Error"
}

