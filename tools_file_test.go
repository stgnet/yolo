package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setupFileTest creates a temporary directory and returns executor + temp dir path
func setupFileTest(t *testing.T, filename string, content string) (*ToolExecutor, string) {
	dir := t.TempDir()
	te := &ToolExecutor{baseDir: dir}
	if content != "" {
		fullPath := filepath.Join(dir, filename)
		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}
	return te, dir
}

// Helper to read test file content
func readTestFileContent(t *testing.T, dir, filename string) string {
	data, err := os.ReadFile(filepath.Join(dir, filename))
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}
	return string(data)
}

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ─── Test read_file ──────────────────────────────────────────────────

func TestReadFile_MissingPath(t *testing.T) {
	executor := &ToolExecutor{baseDir: "."}
	result := executor.readFile(map[string]any{})

	if !strings.HasPrefix(result, "Error:") || !strings.Contains(result, "path is required") {
		t.Errorf("Expected error about missing path, got: %s", result)
	}
}

func TestReadFile_InvalidPath(t *testing.T) {
	te, _ := setupFileTest(t, "", "")
	result := te.readFile(map[string]any{"path": "nonexistent_file.txt"})

	if !strings.HasPrefix(result, "Error:") {
		t.Errorf("Expected error for non-existent file, got: %s", result)
	}
}

func TestReadFile_ValidFile(t *testing.T) {
	testContent := `Line 1 of test content
Line 2 of test content
Line 3 of test content`
	te, _ := setupFileTest(t, "test_read.txt", testContent)

	result := te.readFile(map[string]any{
		"path":   "test_read.txt",
		"offset": 1,
		"limit":  200,
	})

	if strings.HasPrefix(result, "Error:") {
		t.Errorf("Unexpected error reading file: %s", result)
	}

	// Verify content is in result
	if !strings.Contains(result, "Line 1 of test content") || !strings.Contains(result, "Line 2 of test content") {
		maxLen := min(100, len(result))
		t.Errorf("Expected content in result, got: %s", result[:maxLen])
	}

	// Verify header format includes line range
	if !strings.Contains(result, "lines 1-3") || !strings.Contains(result, "of 3]") {
		maxLen := min(150, len(result))
		t.Errorf("Expected file header with line counts, got: %s", result[:maxLen])
	}
}

func TestReadFile_WithOffset(t *testing.T) {
	testContent := `Line 1
Line 2
Line 3
Line 4
Line 5`
	te, _ := setupFileTest(t, "test_offset.txt", testContent)

	result := te.readFile(map[string]any{
		"path":   "test_offset.txt",
		"offset": 3,
		"limit":  2,
	})

	if strings.HasPrefix(result, "Error:") {
		t.Errorf("Unexpected error reading file with offset: %s", result)
	}

	// Should contain only lines 3-4 based on offset and limit
	if !strings.Contains(result, "Line 3") || !strings.Contains(result, "Line 4") {
		t.Errorf("Expected offset to work correctly, got: %s", result)
	}
}

func TestReadFile_BinaryFile(t *testing.T) {
	dir := t.TempDir()
	tempFile := filepath.Join(dir, "binary.bin")
	binaryContent := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD}

	err := os.WriteFile(tempFile, binaryContent, 0644)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	te := &ToolExecutor{baseDir: dir}
	result := te.readFile(map[string]any{"path": "binary.bin"})

	if !strings.Contains(result, "binary file") {
		t.Errorf("Expected error about binary file, got: %s", result)
	}
}

// ─── Test write_file ──────────────────────────────────────────────────

func TestWriteFile_MissingPath(t *testing.T) {
	executor := &ToolExecutor{baseDir: "."}
	result := executor.writeFile(map[string]any{})

	if !strings.HasPrefix(result, "Error:") || !strings.Contains(result, "path is required") {
		t.Errorf("Expected error about missing path, got: %s", result)
	}
}

func TestWriteFile_Valid(t *testing.T) {
	te, dir := setupFileTest(t, "", "")
	testContent := "Test content for write operation with comprehensive data"

	result := te.writeFile(map[string]any{
		"path":    "write_test.txt",
		"content": testContent,
	})

	if strings.HasPrefix(result, "Error:") {
		t.Errorf("Unexpected error writing file: %s", result)
	}

	if !strings.Contains(result, "Wrote") || !strings.Contains(result, "write_test.txt") {
		t.Errorf("Expected success message with file path, got: %s", result)
	}

	// Verify file was created with correct content
	data := readTestFileContent(t, dir, "write_test.txt")
	if data != testContent {
		t.Errorf("File content mismatch: expected '%s', got '%s'", testContent, data)
	}
}

func TestWriteFile_CreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	testContent := "Test content with automatic directory creation"

	te := &ToolExecutor{baseDir: dir}
	result := te.writeFile(map[string]any{
		"path":    "subdir/nested/write_with_dir.txt",
		"content": testContent,
	})

	if strings.HasPrefix(result, "Error:") {
		t.Errorf("Unexpected error writing file with directory creation: %s", result)
	}

	// Verify file was created with correct content
	data := readTestFileContent(t, dir, "subdir/nested/write_with_dir.txt")
	if data != testContent {
		t.Errorf("File content mismatch after directory creation")
	}
}

func TestWriteFile_EmptyContent(t *testing.T) {
	dir := t.TempDir()
	te := &ToolExecutor{baseDir: dir}

	result := te.writeFile(map[string]any{
		"path":    "empty_test.txt",
		"content": "",
	})

	if strings.HasPrefix(result, "Error:") {
		t.Errorf("Unexpected error writing empty file: %s", result)
	}

	// Verify file was created (empty)
	data := readTestFileContent(t, dir, "empty_test.txt")
	if len(data) != 0 {
		t.Errorf("Expected empty file, got %d bytes", len(data))
	}
}

func TestWriteFile_MultilineContent(t *testing.T) {
	dir := t.TempDir()
	testContent := `Line 1
Line 2
Line 3 with special chars: !@#$%^&*()
Line 4 with unicode: café naïve`

	te := &ToolExecutor{baseDir: dir}
	result := te.writeFile(map[string]any{
		"path":    "multiline_test.txt",
		"content": testContent,
	})

	if strings.HasPrefix(result, "Error:") {
		t.Errorf("Unexpected error writing multiline file: %s", result)
	}

	// Verify file content matches exactly
	data := readTestFileContent(t, dir, "multiline_test.txt")
	if data != testContent {
		t.Errorf("Multiline content mismatch: got '%s'", data)
	}
}
