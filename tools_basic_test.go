package main

import (
	"os"
	"path/filepath"
	"testing"
)

// TestExecuteReadFile tests the read_file action in Execute
func TestExecuteReadFile(t *testing.T) {
	dir := t.TempDir()
	executor := NewToolExecutor(dir, nil)

	// Create test file
	testFile := filepath.Join(dir, "test.txt")
	content := "Hello, World!"
	err := os.WriteFile(testFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	args := map[string]any{
		"path":   "test.txt",
		"limit":  10,
		"offset": 1,
	}

	result := executor.Execute("read_file", args)
	if result == "" {
		t.Error("Expected non-empty result from read_file")
	}
	t.Logf("read_file result: %s", result)
}

// TestExecuteWriteFile tests the write_file action in Execute
func TestExecuteWriteFile(t *testing.T) {
	dir := t.TempDir()
	executor := NewToolExecutor(dir, nil)

	testFile := filepath.Join(dir, "test_write.txt")
	content := "Test content for write_file"

	args := map[string]any{
		"path":    "test_write.txt",
		"content": content,
	}

	result := executor.Execute("write_file", args)
	if result == "" {
		t.Error("Expected non-empty result from write_file")
	}

	// Verify file was created and contains correct content
	createdContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read created file: %v", err)
	}
	if string(createdContent) != content {
		t.Errorf("Expected content '%s', got '%s'", content, string(createdContent))
	}
}

// TestExecuteEditFile tests the edit_file action in Execute
func TestExecuteEditFile(t *testing.T) {
	dir := t.TempDir()
	executor := NewToolExecutor(dir, nil)

	testFile := filepath.Join(dir, "test_edit.txt")
	initialContent := "Hello World\nThis is a test\nGoodbye"
	err := os.WriteFile(testFile, []byte(initialContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	args := map[string]any{
		"path":     "test_edit.txt",
		"old_text": "World",
		"new_text": "Universe",
	}

	result := executor.Execute("edit_file", args)
	if result == "" {
		t.Error("Expected non-empty result from edit_file")
	}

	// Verify file was edited correctly
	editedContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read edited file: %v", err)
	}
	expectedContent := "Hello Universe\nThis is a test\nGoodbye"
	if string(editedContent) != expectedContent {
		t.Errorf("Expected content '%s', got '%s'", expectedContent, string(editedContent))
	}
}

// TestExecuteMakeDir tests the make_dir action in Execute
func TestExecuteMakeDir(t *testing.T) {
	dir := t.TempDir()
	executor := NewToolExecutor(dir, nil)

	args := map[string]any{
		"path": "nested/deep/directory",
	}

	result := executor.Execute("make_dir", args)
	if result == "" {
		t.Error("Expected non-empty result from make_dir")
	}

	// Verify directory was created
	newDir := filepath.Join(dir, "nested/deep/directory")
	info, err := os.Stat(newDir)
	if err != nil {
		t.Fatalf("Failed to stat created directory: %v", err)
	}
	if !info.IsDir() {
		t.Error("Expected directory, got file")
	}
}

// TestExecuteRemoveDir tests the remove_dir action in Execute
func TestExecuteRemoveDir(t *testing.T) {
	dir := t.TempDir()
	executor := NewToolExecutor(dir, nil)

	// Create directory to remove
	testDir := filepath.Join(dir, "to_remove")
	nestedDir := filepath.Join(testDir, "nested")
	err := os.MkdirAll(nestedDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	// Create a file in the directory
	testFile := filepath.Join(testDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	args := map[string]any{
		"path": "to_remove",
	}

	result := executor.Execute("remove_dir", args)
	if result == "" {
		t.Error("Expected non-empty result from remove_dir")
	}

	// Verify directory was removed
	_, err = os.Stat(testDir)
	if !os.IsNotExist(err) {
		t.Errorf("Expected directory to be removed, but it still exists")
	}
}

// TestExecuteCopyFile tests the copy_file action in Execute
func TestExecuteCopyFile(t *testing.T) {
	dir := t.TempDir()
	executor := NewToolExecutor(dir, nil)

	// Create source file
	srcFile := filepath.Join(dir, "source.txt")
	content := "Content to copy"
	err := os.WriteFile(srcFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	args := map[string]any{
		"source": "source.txt",
		"dest":   "subdir/dest.txt",
	}

	result := executor.Execute("copy_file", args)
	if result == "" {
		t.Error("Expected non-empty result from copy_file")
	}

	// Verify file was copied
	destFile := filepath.Join(dir, "subdir/dest.txt")
	_, err = os.Stat(destFile)
	if err != nil {
		t.Fatalf("Failed to stat copied file: %v", err)
	}
}

// TestExecuteMoveFile tests the move_file action in Execute
func TestExecuteMoveFile(t *testing.T) {
	dir := t.TempDir()
	executor := NewToolExecutor(dir, nil)

	// Create source file
	srcFile := filepath.Join(dir, "source.txt")
	content := "Content to move"
	err := os.WriteFile(srcFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	args := map[string]any{
		"source": "source.txt",
		"dest":   "subdir/moved.txt",
	}

	result := executor.Execute("move_file", args)
	if result == "" {
		t.Error("Expected non-empty result from move_file")
	}

	// Verify source file was moved
	_, err = os.Stat(srcFile)
	if !os.IsNotExist(err) {
		t.Errorf("Expected source file to be moved, but it still exists")
	}

	// Verify destination file exists
	destFile := filepath.Join(dir, "subdir/moved.txt")
	_, err = os.Stat(destFile)
	if err != nil {
		t.Fatalf("Failed to stat moved file: %v", err)
	}
}

// TestExecuteListFiles tests the list_files action in Execute
func TestExecuteListFiles(t *testing.T) {
	dir := t.TempDir()
	executor := NewToolExecutor(dir, nil)

	// Create test files
	files := []string{"file1.txt", "file2.md", "data.json"}
	for _, f := range files {
		err := os.WriteFile(filepath.Join(dir, f), []byte("test"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", f, err)
		}
	}

	args := map[string]any{
		"pattern": "*.txt",
	}

	result := executor.Execute("list_files", args)
	if result == "" {
		t.Error("Expected non-empty result from list_files")
	}

	// Verify result contains expected file
	if !containsString(result, "file1.txt") {
		t.Errorf("Expected result to contain 'file1.txt', got: %s", result)
	}
}

func containsString(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || (len(s) > len(substr) && (s[:len(substr)] == substr || containsInJSON(s, substr))))
}

func containsInJSON(jsonStr, substr string) bool {
	// Simple check for substring in JSON output
	for i := 0; i <= len(jsonStr)-len(substr); i++ {
		if jsonStr[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestExecuteSearchFiles tests the search_files action in Execute
func TestExecuteSearchFiles(t *testing.T) {
	dir := t.TempDir()
	executor := NewToolExecutor(dir, nil)

	// Create test file with searchable content
	testFile := filepath.Join(dir, "search_me.txt")
	content := "This is a test file\nIt contains some text to search\nHello world"
	err := os.WriteFile(testFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	args := map[string]any{
		"query":   "search",
		"pattern": "*.txt",
	}

	result := executor.Execute("search_files", args)
	if result == "" {
		t.Error("Expected non-empty result from search_files")
	}

	// Verify result mentions the file or contains matches
	if !containsString(result, "search_me.txt") {
		t.Errorf("Expected result to mention 'search_me.txt', got: %s", result)
	}
}

// TestExecuteRunCommand tests the run_command action in Execute
func TestExecuteRunCommand(t *testing.T) {
	dir := t.TempDir()
	executor := NewToolExecutor(dir, nil)

	args := map[string]any{
		"command": "echo 'Hello from command'",
	}

	result := executor.Execute("run_command", args)
	if result == "" {
		t.Error("Expected non-empty result from run_command")
	}

	// Verify command output contains expected text
	if !containsString(result, "Hello from command") {
		t.Errorf("Expected result to contain 'Hello from command', got: %s", result)
	}
}

// TestExecuteListSubagents tests the list_subagents action in Execute
func TestExecuteListSubagents(t *testing.T) {
	dir := t.TempDir()
	executor := NewToolExecutor(dir, nil)

	args := map[string]any{}

	result := executor.Execute("list_subagents", args)
	if result == "" {
		t.Error("Expected non-empty result from list_subagents")
	}
}

// TestExecuteRestart tests the restart action in Execute
func TestExecuteRestart(t *testing.T) {
	dir := t.TempDir()
	_ = dir // unused - restart requires TTY
	// Skip actual restart test as it requires TTY
	t.Log("Skipping restart test - requires interactive terminal")
}

// TestExecuteSpawnSubagent tests the spawn_subagent action in Execute
func TestExecuteSpawnSubagent(t *testing.T) {
	dir := t.TempDir()
	executor := NewToolExecutor(dir, nil)

	args := map[string]any{
		"prompt": "test prompt",
	}

	result := executor.Execute("spawn_subagent", args)
	if result == "" {
		t.Error("Expected non-empty result from spawn_subagent")
	}
}
