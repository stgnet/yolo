// Package tools provides additional file operation tools
package tools

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/scottstg/yolo/config"
)

func TestCopyFileToolMissingSource(t *testing.T) {
	tool := &CopyFileTool{}
	
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"dest": "output.txt",
	})
	
	if err != nil {
		t.Fatalf("Execute returned unexpected error: %v", err)
	}
	
	if result.Success {
		t.Error("Expected failure when source is missing")
	}
	
	if result.Error != "source is required" {
		t.Errorf("Expected 'source is required' error, got '%s'", result.Error)
	}
}

func TestCopyFileToolMissingDest(t *testing.T) {
	tool := &CopyFileTool{}
	
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"source": "input.txt",
	})
	
	if err != nil {
		t.Fatalf("Execute returned unexpected error: %v", err)
	}
	
	if result.Success {
		t.Error("Expected failure when dest is missing")
	}
	
	if result.Error != "dest is required" {
		t.Errorf("Expected 'dest is required' error, got '%s'", result.Error)
	}
}

func TestCopyFileToolInvalidTypes(t *testing.T) {
	tool := &CopyFileTool{}
	
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"source": 123,
		"dest":   true,
	})
	
	if err != nil {
		t.Fatalf("Execute returned unexpected error: %v", err)
	}
	
	if result.Success {
		t.Error("Expected failure when types are invalid")
	}
}

func TestCopyFileToolSuccess(t *testing.T) {
	tmpDir := t.TempDir()
	// Temporarily change working directory for the test
	originalDir := config.WorkingDir()
	config.SetWorkingDir(tmpDir)
	defer config.SetWorkingDir(originalDir)
	
	// Create a source file
	sourcePath := filepath.Join(tmpDir, "source.txt")
	content := "Test content for copying"
	if err := os.WriteFile(sourcePath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}
	
	tool := &CopyFileTool{}
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"source": "source.txt",
		"dest":   "dest/subdir/copied.txt",
	})
	
	if err != nil {
		t.Fatalf("Execute returned unexpected error: %v", err)
	}
	
	if !result.Success {
		t.Errorf("Expected success, got error: %s", result.Error)
	}
	
	// Verify the file was copied
	destPath := filepath.Join(tmpDir, "dest/subdir/copied.txt")
	if !fileExists(destPath) {
		t.Error("Destination file does not exist")
	}
	
	copiedContent, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("Failed to read destination file: %v", err)
	}
	
	if string(copiedContent) != content {
		t.Errorf("Expected '%s', got '%s'", content, string(copiedContent))
	}
	
	// Verify source still exists
	if !fileExists(sourcePath) {
		t.Error("Source file was deleted (should only happen with move)")
	}
}

func TestCopyFileToolSourceNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	originalDir := config.WorkingDir()
	config.SetWorkingDir(tmpDir)
	defer config.SetWorkingDir(originalDir)
	
	tool := &CopyFileTool{}
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"source": "nonexistent.txt",
		"dest":   "output.txt",
	})
	
	if err != nil {
		t.Fatalf("Execute returned unexpected error: %v", err)
	}
	
	if result.Success {
		t.Error("Expected failure when source file doesn't exist")
	}
}

func TestMoveFileToolMissingSource(t *testing.T) {
	tool := &MoveFileTool{}
	
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"dest": "output.txt",
	})
	
	if err != nil {
		t.Fatalf("Execute returned unexpected error: %v", err)
	}
	
	if result.Success {
		t.Error("Expected failure when source is missing")
	}
	
	if result.Error != "source is required" {
		t.Errorf("Expected 'source is required' error, got '%s'", result.Error)
	}
}

func TestMoveFileToolMissingDest(t *testing.T) {
	tool := &MoveFileTool{}
	
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"source": "input.txt",
	})
	
	if err != nil {
		t.Fatalf("Execute returned unexpected error: %v", err)
	}
	
	if result.Success {
		t.Error("Expected failure when dest is missing")
	}
	
	if result.Error != "dest is required" {
		t.Errorf("Expected 'dest is required' error, got '%s'", result.Error)
	}
}

func TestMoveFileToolSuccess(t *testing.T) {
	tmpDir := t.TempDir()
	originalDir := config.WorkingDir()
	config.SetWorkingDir(tmpDir)
	defer config.SetWorkingDir(originalDir)
	
	// Create a source file
	sourcePath := filepath.Join(tmpDir, "source.txt")
	content := "Test content for moving"
	if err := os.WriteFile(sourcePath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}
	
	tool := &MoveFileTool{}
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"source": "source.txt",
		"dest":   "moved.txt",
	})
	
	if err != nil {
		t.Fatalf("Execute returned unexpected error: %v", err)
	}
	
	if !result.Success {
		t.Errorf("Expected success, got error: %s", result.Error)
	}
	
	// Verify the file was moved
	destPath := filepath.Join(tmpDir, "moved.txt")
	if !fileExists(destPath) {
		t.Error("Destination file does not exist after move")
	}
	
	movedContent, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("Failed to read destination file: %v", err)
	}
	
	if string(movedContent) != content {
		t.Errorf("Expected '%s', got '%s'", content, string(movedContent))
	}
	
	// Verify source no longer exists
	if fileExists(sourcePath) {
		t.Error("Source file still exists after move")
	}
}

func TestMoveFileToolCreatesDirectories(t *testing.T) {
	tmpDir := t.TempDir()
	originalDir := config.WorkingDir()
	config.SetWorkingDir(tmpDir)
	defer config.SetWorkingDir(originalDir)
	
	// Create a source file
	sourcePath := filepath.Join(tmpDir, "source.txt")
	content := "Test content"
	if err := os.WriteFile(sourcePath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}
	
	tool := &MoveFileTool{}
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"source": "source.txt",
		"dest":   "nested/deep/path/moved.txt",
	})
	
	if err != nil {
		t.Fatalf("Execute returned unexpected error: %v", err)
	}
	
	if !result.Success {
		t.Errorf("Expected success, got error: %s", result.Error)
	}
	
	// Verify the file was moved to nested directory
	destPath := filepath.Join(tmpDir, "nested/deep/path/moved.txt")
	if !fileExists(destPath) {
		t.Error("Destination file does not exist in nested directory")
	}
}

func TestCopyFileHelper(t *testing.T) {
	tmpDir := t.TempDir()
	originalDir := config.WorkingDir()
	config.SetWorkingDir(tmpDir)
	defer config.SetWorkingDir(originalDir)
	
	sourcePath := filepath.Join(tmpDir, "source.txt")
	destPath := filepath.Join(tmpDir, "dest", "copied.txt")
	content := "Test content"
	
	if err := os.WriteFile(sourcePath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}
	
	err := copyFile("source.txt", "dest/copied.txt")
	if err != nil {
		t.Fatalf("copyFile failed: %v", err)
	}
	
	if !fileExists(destPath) {
		t.Error("Destination file does not exist")
	}
	
	copiedContent, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("Failed to read copied file: %v", err)
	}
	
	if string(copiedContent) != content {
		t.Errorf("Content mismatch: expected '%s', got '%s'", content, string(copiedContent))
	}
}

func TestMoveFileHelper(t *testing.T) {
	tmpDir := t.TempDir()
	originalDir := config.WorkingDir()
	config.SetWorkingDir(tmpDir)
	defer config.SetWorkingDir(originalDir)
	
	sourcePath := filepath.Join(tmpDir, "source.txt")
	destPath := filepath.Join(tmpDir, "moved.txt")
	content := "Test content"
	
	if err := os.WriteFile(sourcePath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}
	
	err := moveFile("source.txt", "moved.txt")
	if err != nil {
		t.Fatalf("moveFile failed: %v", err)
	}
	
	if !fileExists(destPath) {
		t.Error("Destination file does not exist after move")
	}
	
	if fileExists(sourcePath) {
		t.Error("Source file still exists after move")
	}
}

func TestCopyFileHelperSourceNotFound(t *testing.T) {
	err := copyFile("nonexistent.txt", "output.txt")
	if err == nil {
		t.Error("Expected error when source file doesn't exist")
	}
}

func TestMoveFileHelperSourceNotFound(t *testing.T) {
	err := moveFile("nonexistent.txt", "output.txt")
	if err == nil {
		t.Error("Expected error when source file doesn't exist")
	}
}

// Helper function for tests
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}
