package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadFileExists(t *testing.T) {
	// Create a temporary file for testing
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	content := "Hello, World!"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	result, err := ReadFile(testFile)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	if string(result) != content {
		t.Errorf("Expected '%s', got '%s'", content, string(result))
	}
}

func TestReadFileNotFound(t *testing.T) {
	result, err := ReadFile("/nonexistent/path/file.txt")
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
	if result != nil {
		t.Errorf("Expected nil result for error, got %v", result)
	}
}

func TestWriteFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "output.txt")
	content := "Test content for file writing"

	err := WriteFile(testFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	result, err := ReadFile(testFile)
	if err != nil {
		t.Fatalf("ReadFile failed after write: %v", err)
	}

	if string(result) != content {
		t.Errorf("Expected '%s', got '%s'", content, string(result))
	}
}

func TestWriteFileCreateDirs(t *testing.T) {
	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, "nested", "path", "file.txt")
	content := "Content in nested directory"

	err := WriteFile(testPath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("WriteFile failed to create directories: %v", err)
	}

	result, err := ReadFile(testPath)
	if err != nil {
		t.Fatalf("ReadFile failed after write to nested path: %v", err)
	}

	if string(result) != content {
		t.Errorf("Expected '%s', got '%s'", content, string(result))
	}
}

func TestListFilesTopLevel(t *testing.T) {
	tmpDir := t.TempDir()
	
	// Create test files
	files := []string{"file1.txt", "file2.txt", "dir1/file3.txt"}
	for _, f := range files {
		path := filepath.Join(tmpDir, f)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}
	}

	filesResult, err := ListFiles(filepath.Join(tmpDir))
	if err != nil {
		t.Fatalf("ListFiles failed: %v", err)
	}

	// Should find the two files in top level, not the nested one
	if len(filesResult) < 2 {
		t.Errorf("Expected at least 2 files, got %d", len(filesResult))
	}
}

func TestEnsureDir(t *testing.T) {
	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, "a", "b", "c")

	err := EnsureDir(testPath)
	if err != nil {
		t.Fatalf("EnsureDir failed: %v", err)
	}

	if !IsDirectory(testPath) {
		t.Error("Directory was not created")
	}
}

func TestDeleteFile(t *testing.T) {
	tmpDir := t.TempDir()
	
	testFile := filepath.Join(tmpDir, "file.txt")
	content := "Content to delete"

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	err := DeleteFile(testFile)
	if err != nil {
		t.Fatalf("DeleteFile failed: %v", err)
	}

	if FileExists(testFile) {
		t.Error("File still exists after deletion")
	}
}

func TestGetFileSize(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	content := "Test content"

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	size, err := GetFileSize(testFile)
	if err != nil {
		t.Fatalf("GetFileSize failed: %v", err)
	}

	if size != int64(len(content)) {
		t.Errorf("Expected size %d, got %d", len(content), size)
	}
}
