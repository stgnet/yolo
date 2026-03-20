package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileExists(t *testing.T) {
	tmpDir := t.TempDir()
	existingFile := filepath.Join(tmpDir, "exists.txt")
	nonExistingFile := filepath.Join(tmpDir, "not_exists.txt")

	// Create a test file
	if err := os.WriteFile(existingFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test existing file
	if !FileExists(existingFile) {
		t.Error("Expected FileExists to return true for existing file")
	}

	// Test non-existing file
	if FileExists(nonExistingFile) {
		t.Error("Expected FileExists to return false for non-existing file")
	}

	// Test directory returns false (it's a dir, not a file)
	if FileExists(tmpDir) {
		t.Error("Expected FileExists to return false for directory")
	}
}

func TestIsDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	existingFile := filepath.Join(tmpDir, "exists.txt")

	// Create a test file
	if err := os.WriteFile(existingFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test existing directory
	if !IsDirectory(tmpDir) {
		t.Error("Expected IsDirectory to return true for existing directory")
	}

	// Test file returns false (it's a file, not a dir)
	if IsDirectory(existingFile) {
		t.Error("Expected IsDirectory to return false for file")
	}

	// Test non-existing path
	if IsDirectory(filepath.Join(tmpDir, "not_exists")) {
		t.Error("Expected IsDirectory to return false for non-existing path")
	}
}

func TestEnsureDir(t *testing.T) {
	tmpDir := t.TempDir()
	newDir := filepath.Join(tmpDir, "new", "nested", "dir")

	err := EnsureDir(newDir)
	if err != nil {
		t.Fatalf("EnsureDir failed: %v", err)
	}

	if !IsDirectory(newDir) {
		t.Error("Expected directory to be created by EnsureDir")
	}
}

func TestReadFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	expectedContent := []byte("Hello, World!")

	// Create test file
	if err := os.WriteFile(testFile, expectedContent, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test reading existing file
	content, err := ReadFile(testFile)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	if len(content) != len(expectedContent) || string(content) != string(expectedContent) {
		t.Errorf("Expected content %q, got %q", expectedContent, content)
	}

	// Test reading non-existing file
	_, err = ReadFile(filepath.Join(tmpDir, "not_exists.txt"))
	if err == nil {
		t.Error("Expected error when reading non-existing file")
	}
}

func TestReadFileString(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	expectedContent := "Hello, World!"

	// Create test file
	if err := os.WriteFile(testFile, []byte(expectedContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test reading as string
	content, err := ReadFileString(testFile)
	if err != nil {
		t.Fatalf("ReadFileString failed: %v", err)
	}

	if content != expectedContent {
		t.Errorf("Expected content %q, got %q", expectedContent, content)
	}
}

func TestWriteFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	content := []byte("Hello, World!")

	err := WriteFile(testFile, content, 0644)
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	if !FileExists(testFile) {
		t.Error("Expected file to exist after WriteFile")
	}

	readContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read written file: %v", err)
	}

	if string(readContent) != string(content) {
		t.Errorf("Expected content %q, got %q", content, readContent)
	}
}

func TestWriteFileString(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	content := "Hello, World!"

	err := WriteFileString(testFile, content, 0644)
	if err != nil {
		t.Fatalf("WriteFileString failed: %v", err)
	}

	readContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read written file: %v", err)
	}

	if string(readContent) != content {
		t.Errorf("Expected content %q, got %q", content, readContent)
	}
}

func TestCopyFile(t *testing.T) {
	tmpDir := t.TempDir()
	srcFile := filepath.Join(tmpDir, "source.txt")
	dstFile := filepath.Join(tmpDir, "dest.txt")
	content := []byte("Hello, World!")

	// Create source file
	if err := os.WriteFile(srcFile, content, 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	// Copy file
	err := CopyFile(srcFile, dstFile)
	if err != nil {
		t.Fatalf("CopyFile failed: %v", err)
	}

	// Verify destination exists and has correct content
	readContent, err := os.ReadFile(dstFile)
	if err != nil {
		t.Fatalf("Failed to read copied file: %v", err)
	}

	if string(readContent) != string(content) {
		t.Errorf("Expected content %q, got %q", content, readContent)
	}

	// Test copying non-existing file
	err = CopyFile(filepath.Join(tmpDir, "not_exists.txt"), filepath.Join(tmpDir, "dest2.txt"))
	if err == nil {
		t.Error("Expected error when copying non-existing file")
	}
}

func TestMoveFile(t *testing.T) {
	tmpDir := t.TempDir()
	srcFile := filepath.Join(tmpDir, "source.txt")
	dstFile := filepath.Join(tmpDir, "dest.txt")
	content := []byte("Hello, World!")

	// Create source file
	if err := os.WriteFile(srcFile, content, 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	// Move file
	err := MoveFile(srcFile, dstFile)
	if err != nil {
		t.Fatalf("MoveFile failed: %v", err)
	}

	// Verify source no longer exists
	if FileExists(srcFile) {
		t.Error("Expected source file to not exist after MoveFile")
	}

	// Verify destination exists and has correct content
	if !FileExists(dstFile) {
		t.Error("Expected destination file to exist after MoveFile")
	}

	readContent, err := os.ReadFile(dstFile)
	if err != nil {
		t.Fatalf("Failed to read moved file: %v", err)
	}

	if string(readContent) != string(content) {
		t.Errorf("Expected content %q, got %q", content, readContent)
	}
}

func TestDeleteFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	// Create test file
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Delete file
	err := DeleteFile(testFile)
	if err != nil {
		t.Fatalf("DeleteFile failed: %v", err)
	}

	if FileExists(testFile) {
		t.Error("Expected file to not exist after DeleteFile")
	}

	// Test deleting non-existing file
	err = DeleteFile(filepath.Join(tmpDir, "not_exists.txt"))
	if err == nil {
		t.Error("Expected error when deleting non-existing file")
	}

	// Test deleting directory should fail
	err = DeleteFile(tmpDir)
	if err == nil {
		t.Error("Expected error when trying to delete a directory with DeleteFile")
	}
}

func TestSafetyConfigProtectedPaths(t *testing.T) {
	config := DefaultSafetyConfig()

	// Test protected paths - must match exactly or start with "protected_path/"
	protectedPaths := []string{
		"/etc",
		"/etc/passwd",
		"/bin/sh",
		".git/HEAD",
		"go.mod",
		".git/index",
	}

	for _, path := range protectedPaths {
		if !config.IsProtectedPath(path) {
			t.Errorf("Expected %q to be protected", path)
		}
	}

	// Test non-protected paths
	nonProtectedPaths := []string{
		"tmp/test.txt",
		"src/app.go",
		"data/output.json",
		"binaries/sh",  // doesn't match /bin because of path.Clean behavior
	}

	for _, path := range nonProtectedPaths {
		if config.IsProtectedPath(path) {
			t.Errorf("Expected %q to not be protected", path)
		}
	}
}

func TestSafetyConfigCustomSettings(t *testing.T) {
	config := &SafetyConfig{
		EnableSizeCheck: false,
		MaxDeleteSize:   1024,
		CreateBackup:    false,
		ProtectedPaths:  []string{"/custom/protected"},
	}

	if config.EnableSizeCheck != false {
		t.Error("Expected EnableSizeCheck to be false")
	}
	if config.MaxDeleteSize != 1024 {
		t.Errorf("Expected MaxDeleteSize to be 1024, got %d", config.MaxDeleteSize)
	}
	if config.CreateBackup != false {
		t.Error("Expected CreateBackup to be false")
	}

	if !config.IsProtectedPath("/custom/protected/file.txt") {
		t.Error("Expected /custom/protected/file.txt to be protected")
	}
	if config.IsProtectedPath("/other/path") {
		t.Error("Expected /other/path to not be protected")
	}
}

func TestWriteFileCreatesParentDirectories(t *testing.T) {
	tmpDir := t.TempDir()
	nestedFile := filepath.Join(tmpDir, "a", "b", "c", "test.txt")
	content := []byte("Hello, World!")

	err := WriteFile(nestedFile, content, 0644)
	if err != nil {
		t.Fatalf("WriteFile failed to create parent directories: %v", err)
	}

	if !FileExists(nestedFile) {
		t.Error("Expected file to exist after WriteFile with nested path")
	}

	readContent, err := os.ReadFile(nestedFile)
	if err != nil {
		t.Fatalf("Failed to read written file: %v", err)
	}

	if string(readContent) != string(content) {
		t.Errorf("Expected content %q, got %q", content, readContent)
	}
}
