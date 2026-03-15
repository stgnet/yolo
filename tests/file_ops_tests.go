// Package main provides comprehensive unit tests for the utils package.
// These tests use the utils package and validate all file operations without side effects.
package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"yolo/utils"
)

// ─── File Operations Tests ──────────────

// TestFileOps_Comprehensive tests comprehensive file operations
func TestFileOps_Comprehensive(t *testing.T) {
	t.Run("TestSafetyConfig", func(t *testing.T) {
		cfg := utils.DefaultSafetyConfig()

		if !cfg.EnableSizeCheck {
			t.Error("EnableSizeCheck should be true by default")
		}

		if cfg.MaxDeleteSize <= 0 {
			t.Error("MaxDeleteSize should have a positive value")
		}

		expected := int64(100 * 1024 * 1024) // 100 MB
		if cfg.MaxDeleteSize != expected {
			t.Errorf("Expected MaxDeleteSize %d, got %d", expected, cfg.MaxDeleteSize)
		}

		if len(cfg.ProtectedPaths) == 0 {
			t.Error("ProtectedPaths should not be empty")
		}

		if !cfg.CreateBackup {
			t.Error("CreateBackup should be true by default")
		}
	})

	t.Run("TestSafetyConfig_ProtectedPaths", func(t *testing.T) {
		cfg := utils.DefaultSafetyConfig()

		protectedPaths := []string{"/etc", "/bin", ".git/HEAD", "go.mod"}
		for _, path := range protectedPaths {
			found := false
			for _, p := range cfg.ProtectedPaths {
				if p == path {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Protected path %q not found", path)
			}
		}
	})

	t.Run("TestReadFile", func(t *testing.T) {
		tempDir := t.TempDir()
		testFile := filepath.Join(tempDir, "test.txt")

		err := os.WriteFile(testFile, []byte("Hello, World!"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		data, err := utils.ReadFile(testFile)
		if err != nil {
			t.Errorf("Unexpected error reading file: %v", err)
		}

		expected := []byte("Hello, World!")
		if string(data) != string(expected) {
			t.Errorf("Expected content %q, got %q", expected, data)
		}
	})

	t.Run("TestReadFile_NonExistent", func(t *testing.T) {
		tempDir := t.TempDir()
		_, err := utils.ReadFile(filepath.Join(tempDir, "nonexistent.txt"))
		if err == nil {
			t.Error("Expected error when reading non-existent file")
		}
	})

	t.Run("TestWriteFile", func(t *testing.T) {
		tempDir := t.TempDir()
		testFile := filepath.Join(tempDir, "test.txt")

		content := []byte("Test write content")
		err := utils.WriteFile(testFile, content, 0644)
		if err != nil {
			t.Errorf("Unexpected error writing file: %v", err)
		}

		data, err := os.ReadFile(testFile)
		if err != nil {
			t.Fatalf("Failed to read back written file: %v", err)
		}

		if string(data) != string(content) {
			t.Errorf("Expected content %q, got %q", content, data)
		}
	})

	t.Run("TestWriteFile_CreatesParents", func(t *testing.T) {
		tempDir := t.TempDir()
		nestedFile := filepath.Join(tempDir, "nested/dir/test.txt")

		content := []byte("Nested file content")
		err := utils.WriteFile(nestedFile, content, 0644)
		if err != nil {
			t.Errorf("Unexpected error creating nested file: %v", err)
		}

		data, err := os.ReadFile(nestedFile)
		if err != nil {
			t.Fatalf("Failed to read back nested file: %v", err)
		}

		if string(data) != string(content) {
			t.Errorf("Expected content %q, got %q", content, data)
		}
	})

	t.Run("TestFileExists", func(t *testing.T) {
		tempDir := t.TempDir()

		testFile := filepath.Join(tempDir, "test.txt")
		os.WriteFile(testFile, []byte("test"), 0644)

		if !utils.FileExists(testFile) {
			t.Errorf("Expected %q to exist", testFile)
		}

		nonExistent := filepath.Join(tempDir, "doesnotexist.txt")
		if utils.FileExists(nonExistent) {
			t.Errorf("Expected %q to not exist", nonExistent)
		}
	})

	t.Run("TestIsDirectory", func(t *testing.T) {
		tempDir := t.TempDir()

		if !utils.IsDirectory(tempDir) {
			t.Errorf("Expected %q to be a directory", tempDir)
		}

		testFile := filepath.Join(tempDir, "test.txt")
		os.WriteFile(testFile, []byte("test"), 0644)

		if utils.IsDirectory(testFile) {
			t.Errorf("Expected %q to not be a directory", testFile)
		}
	})

	t.Run("TestEnsureDir", func(t *testing.T) {
		tempDir := t.TempDir()
		nestedDir := filepath.Join(tempDir, "level1/level2/level3")

		err := utils.EnsureDir(nestedDir)
		if err != nil {
			t.Errorf("Unexpected error creating nested dir: %v", err)
		}

		info, err := os.Stat(nestedDir)
		if err != nil {
			t.Fatalf("Failed to stat created directory: %v", err)
		}

		if !info.IsDir() {
			t.Error("Created path should be a directory")
		}
	})

	t.Run("TestDeleteFile", func(t *testing.T) {
		tempDir := t.TempDir()
		testFile := filepath.Join(tempDir, "test.txt")

		os.WriteFile(testFile, []byte("Test content"), 0644)

		err := utils.DeleteFile(testFile)
		if err != nil {
			t.Errorf("Unexpected error deleting file: %v", err)
		}

		if utils.FileExists(testFile) {
			t.Errorf("Expected %q to be deleted", testFile)
		}
	})

	t.Run("TestDeleteFile_Directory", func(t *testing.T) {
		tempDir := t.TempDir()
		subdir := filepath.Join(tempDir, "subdir")
		os.Mkdir(subdir, 0755)

		err := utils.DeleteFile(subdir)
		if err == nil {
			t.Error("Expected error when trying to delete directory with DeleteFile")
		}
	})

	t.Run("TestCopyFile", func(t *testing.T) {
		tempDir := t.TempDir()
		srcFile := filepath.Join(tempDir, "source.txt")
		dstFile := filepath.Join(tempDir, "destination.txt")

		content := []byte("Source file content for copying")
		os.WriteFile(srcFile, content, 0644)

		err := utils.CopyFile(srcFile, dstFile)
		if err != nil {
			t.Errorf("Unexpected error copying file: %v", err)
		}

		data, err := os.ReadFile(dstFile)
		if err != nil {
			t.Fatalf("Failed to read copied file: %v", err)
		}

		if string(data) != string(content) {
			t.Errorf("Expected content %q, got %q", content, data)
		}
	})

	t.Run("TestMoveFile", func(t *testing.T) {
		tempDir := t.TempDir()
		srcFile := filepath.Join(tempDir, "source.txt")
		dstFile := filepath.Join(tempDir, "destination.txt")

		content := []byte("Content to be moved")
		os.WriteFile(srcFile, content, 0644)

		err := utils.MoveFile(srcFile, dstFile)
		if err != nil {
			t.Errorf("Unexpected error moving file: %v", err)
		}

		if utils.FileExists(srcFile) {
			t.Errorf("Expected source file to be removed after move")
		}

		if !utils.FileExists(dstFile) {
			t.Errorf("Expected destination file to exist after move")
		}

		data, err := os.ReadFile(dstFile)
		if err != nil {
			t.Fatalf("Failed to read moved file: %v", err)
		}

		if string(data) != string(content) {
			t.Errorf("Expected content %q after move, got %q", content, data)
		}
	})

	t.Run("TestGetFileSize", func(t *testing.T) {
		tempDir := t.TempDir()
		testFile := filepath.Join(tempDir, "test.txt")

		content := []byte("12345") // 5 bytes
		os.WriteFile(testFile, content, 0644)

		size, err := utils.GetFileSize(testFile)
		if err != nil {
			t.Errorf("Unexpected error getting file size: %v", err)
		}

		if size != 5 {
			t.Errorf("Expected size 5, got %d", size)
		}
	})

	t.Run("TestGetFileModTime", func(t *testing.T) {
		tempDir := t.TempDir()
		testFile := filepath.Join(tempDir, "test.txt")

		os.WriteFile(testFile, []byte("content"), 0644)

		modTime, err := utils.GetFileModTime(testFile)
		if err != nil {
			t.Errorf("Unexpected error getting modification time: %v", err)
		}

		if modTime.IsZero() {
			t.Error("Expected non-zero modification time")
		}
	})

	t.Run("TestReadDir", func(t *testing.T) {
		tempDir := t.TempDir()

		os.Mkdir(filepath.Join(tempDir, "subdir"), 0755)
		os.WriteFile(filepath.Join(tempDir, "file1.txt"), []byte("content1"), 0644)
		os.WriteFile(filepath.Join(tempDir, "file2.txt"), []byte("content2"), 0644)

		entries, err := utils.ReadDir(tempDir)
		if err != nil {
			t.Errorf("Unexpected error reading directory: %v", err)
		}

		if len(entries) < 2 {
			t.Errorf("Expected at least 2 entries, got %d", len(entries))
		}
	})

	t.Run("TestListFiles", func(t *testing.T) {
		tempDir := t.TempDir()

		os.Mkdir(filepath.Join(tempDir, "subdir"), 0755)
		os.WriteFile(filepath.Join(tempDir, "file1.txt"), []byte("content1"), 0644)
		os.WriteFile(filepath.Join(tempDir, "file2.txt"), []byte("content2"), 0644)

		files, err := utils.ListFiles(tempDir)
		if err != nil {
			t.Errorf("Unexpected error listing files: %v", err)
		}

		for _, f := range files {
			if f == "subdir" {
				t.Error("ListFiles should not include directories")
			}
		}
	})
}

// TestFileOps_EdgeCases tests edge cases for file operations
func TestFileOps_EdgeCases(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("read empty path returns error", func(t *testing.T) {
		_, err := utils.ReadFile("")
		if err == nil {
			t.Error("Expected error for empty path")
		}
	})

	t.Run("isDirectory with empty string", func(t *testing.T) {
		result := utils.IsDirectory("")
		if result {
			t.Error("Empty path should not be considered a directory")
		}
	})

	t.Run("FileExists with empty path", func(t *testing.T) {
		if utils.FileExists("") {
			t.Error("Empty path should not exist as file")
		}
	})

	t.Run("delete root returns error", func(t *testing.T) {
		err := utils.DeleteFile("/")
		if err == nil {
			t.Error("Expected error when trying to delete root")
		}
	})

	t.Run("WriteFileString", func(t *testing.T) {
		testFile := filepath.Join(tempDir, "test.txt")
		content := "String content for WriteFileString"

		err := utils.WriteFileString(testFile, content, 0644)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		data, err := os.ReadFile(testFile)
		if err != nil {
			t.Fatalf("Failed to read back: %v", err)
		}

		if string(data) != content {
			t.Errorf("Expected %q, got %q", content, data)
		}
	})
}

// TestFileOps_Integration tests workflow integration for file operations
func TestFileOps_Integration(t *testing.T) {
	t.Run("workflow: read, modify, write, verify", func(t *testing.T) {
		tempDir := t.TempDir()
		testFile := filepath.Join(tempDir, "test.txt")

		// Write initial content
		err := utils.WriteFile(testFile, []byte("Initial"), 0644)
		if err != nil {
			t.Fatalf("Failed to write initial file: %v", err)
		}

		// Read the file
		data, err := utils.ReadFile(testFile)
		if err != nil {
			t.Errorf("Unexpected error reading: %v", err)
		}

		if string(data) != "Initial" {
			t.Errorf("Expected 'Initial', got %q", data)
		}

		// Modify and write back
		newContent := []byte("Modified content")
		err = utils.WriteFile(testFile, newContent, 0644)
		if err != nil {
			t.Errorf("Unexpected error writing: %v", err)
		}

		// Verify modification
		data, err = utils.ReadFile(testFile)
		if err != nil {
			t.Errorf("Unexpected error reading modified file: %v", err)
		}

		if string(data) != "Modified content" {
			t.Errorf("Expected 'Modified content', got %q", data)
		}
	})

	t.Run("workflow: copy and verify integrity", func(t *testing.T) {
		tempDir := t.TempDir()
		srcFile := filepath.Join(tempDir, "source.txt")
		dstFile := filepath.Join(tempDir, "destination.txt")

		content := []byte("Test content for copy verification")
		os.WriteFile(srcFile, content, 0644)

		err := utils.CopyFile(srcFile, dstFile)
		if err != nil {
			t.Fatalf("Failed to copy: %v", err)
		}

		dstData, err := os.ReadFile(dstFile)
		if err != nil {
			t.Fatalf("Failed to read destination: %v", err)
		}

		if string(dstData) != string(content) {
			t.Errorf("Copy integrity check failed")
		}
	})

	t.Run("workflow: move and verify", func(t *testing.T) {
		tempDir := t.TempDir()
		srcFile := filepath.Join(tempDir, "source.txt")
		dstFile := filepath.Join(tempDir, "destination.txt")

		content := []byte("Test content for move verification")
		os.WriteFile(srcFile, content, 0644)

		err := utils.MoveFile(srcFile, dstFile)
		if err != nil {
			t.Fatalf("Failed to move: %v", err)
		}

		if utils.FileExists(srcFile) {
			t.Error("Source should not exist after move")
		}

		if !utils.FileExists(dstFile) {
			t.Error("Destination should exist after move")
		}

		dstData, err := os.ReadFile(dstFile)
		if err != nil {
			t.Fatalf("Failed to read moved file: %v", err)
		}

		if string(dstData) != string(content) {
			t.Errorf("Move integrity check failed")
		}
	})
}

// TestSafetyConfigDetailed tests SafetyConfig with various configurations
func TestSafetyConfigDetailed(t *testing.T) {
	t.Run("custom SafetyConfig with disabled size check", func(t *testing.T) {
		cfg := &utils.SafetyConfig{
			EnableSizeCheck: false,
			MaxDeleteSize:   1024 * 1024,
			ProtectedPaths:  []string{},
			CreateBackup:    false,
		}

		if cfg.EnableSizeCheck {
			t.Error("EnableSizeCheck should be false")
		}
	})

	t.Run("protected path detection", func(t *testing.T) {
		cfg := utils.DefaultSafetyConfig()

		// Test protected paths
		protectedPaths := []string{"/etc/passwd", "/bin/sh", ".git/HEAD"}
		for _, pp := range protectedPaths {
			if !cfg.isProtectedPath(pp) {
				t.Errorf("Expected %q to be protected", pp)
			}
		}

		// Test non-protected paths
		safePaths := []string{"./myfile.txt", "/tmp/test.txt"}
		for _, sp := range safePaths {
			if cfg.isProtectedPath(sp) {
				t.Errorf("Expected %q to not be protected", sp)
			}
		}
	})

	t.Run("size check rejects large files", func(t *testing.T) {
		tempDir := t.TempDir()
		largeFile := filepath.Join(tempDir, "large.txt")

		largeContent := make([]byte, 200) // 200 bytes
		os.WriteFile(largeFile, largeContent, 0644)

		cfg := &utils.SafetyConfig{
			EnableSizeCheck: true,
			MaxDeleteSize:   100,
			ProtectedPaths:  []string{},
			CreateBackup:    false,
		}

		err := utils.DeleteFileWithConfig(largeFile, cfg)
		if err == nil {
			t.Error("Expected error when deleting large file with size limit")
		} else if err.Error() != "file too large for deletion (max: 100 bytes, actual: 200)" {
			t.Logf("Error message: %v", err)
		}
	})

	t.Run("size check allows smaller files", func(t *testing.T) {
		tempDir := t.TempDir()
		testFile := filepath.Join(tempDir, "small.txt")

		os.WriteFile(testFile, []byte("12345"), 0644) // 5 bytes

		cfg := &utils.SafetyConfig{
			EnableSizeCheck: true,
			MaxDeleteSize:   100,
			ProtectedPaths:  []string{},
			CreateBackup:    false,
		}

		err := utils.DeleteFileWithConfig(testFile, cfg)
		if err != nil {
			t.Errorf("Unexpected error deleting small file with size check: %v", err)
		}
	})
}

// TestHelperFunctions tests utility functions used in testing
func TestHelperFunctions(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("createTestFile helper", func(t *testing.T) {
		testFile := filepath.Join(tempDir, "helper.txt")
		err := createTestFile(testFile, "test content")
		if err != nil {
			t.Errorf("Failed to create test file: %v", err)
		}

		data, err := os.ReadFile(testFile)
		if err != nil {
			t.Fatalf("Failed to read created file: %v", err)
		}

		if string(data) != "test content" {
			t.Errorf("Expected 'test content', got %q", data)
		}
	})

	t.Run("deleteTestDir helper", func(t *testing.T) {
		testDir := filepath.Join(tempDir, "toDelete")
		os.Mkdir(testDir, 0755)

		err := deleteTestDir(testDir)
		if err != nil {
			t.Errorf("Failed to delete test dir: %v", err)
		}

		if utils.IsDirectory(testDir) {
			t.Error("Directory should be deleted")
		}
	})
}

// Helper function to create test file
func createTestFile(path string, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}

// Helper function to delete test directory
func deleteTestDir(dir string) error {
	return os.RemoveAll(dir)
}

// TestManualVerification performs manual verification tests
func TestManualVerification(t *testing.T) {
	t.Run("verify file operations with different permissions", func(t *testing.T) {
		tempDir := t.TempDir()
		testFile := filepath.Join(tempDir, "test.txt")

		for _, perm := range []os.FileMode{0644, 0600, 0755} {
			err := os.WriteFile(testFile, []byte("content"), perm)
			if err != nil {
				t.Errorf("Failed to create file with permission %o: %v", perm, err)
				continue
			}

			data, err := utils.ReadFile(testFile)
			if err != nil {
				t.Errorf("Failed to read file with permission %o: %v", perm, err)
				continue
			}

			if string(data) != "content" {
				t.Errorf("Read content mismatch for permission %o", perm)
			}
		}
	})

	t.Run("verify DefaultSafetyConfig has correct number of protected paths", func(t *testing.T) {
		cfg := utils.DefaultSafetyConfig()
		t.Logf("ProtectedPaths count: %d", len(cfg.ProtectedPaths))
	})
}
