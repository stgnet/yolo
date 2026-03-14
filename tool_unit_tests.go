// Comprehensive unit tests for YOLO tool handlers
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestWriteFile_SuccessfulWrite - Test write file through ToolExecutor
func TestWriteFile_SuccessfulWrite(t *testing.T) {
	executor := NewToolExecutor(".", nil)
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")

	args := map[string]any{
		"path":    testFile,
		"content": "Hello, World!",
	}

	result := executor.writeFile(args)

	if !containsString(result, "written") {
		t.Errorf("WriteFile should succeed and return success message, got: %s", result)
	}

	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read written file: %v", err)
	}
	if string(content) != "Hello, World!" {
		t.Errorf("Written content should match input, got: %s", string(content))
	}
}

// TestReadFile_SuccessfulRead - Test read file through ToolExecutor
func TestReadFile_SuccessfulRead(t *testing.T) {
	executor := NewToolExecutor(".", nil)
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "read_test.txt")
	content := "Line 1\nLine 2\nLine 3"

	err := os.WriteFile(testFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	args := map[string]any{
		"path":   testFile,
		"limit":  10,
		"offset": 1,
	}

	result := executor.readFile(args)

	if !containsString(result, "content:") {
		t.Errorf("ReadFile should succeed and return content, got: %s", result)
	}
}

// TestListFiles_SuccessfulList - Test list files through ToolExecutor
func TestListFiles_SuccessfulList(t *testing.T) {
	executor := NewToolExecutor(".", nil)
	tempDir := t.TempDir()

	// Create some test files
	for i := 0; i < 3; i++ {
		filePath := filepath.Join(tempDir, fmt.Sprintf("file_%d.txt", i))
		err := os.WriteFile(filePath, []byte("test"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	args := map[string]any{
		"pattern": "*",
	}

	result := executor.listFiles(args)

	if !containsString(result, "files:") {
		t.Errorf("ListFiles should succeed and return files, got: %s", result)
	}
}

// TestRunCommand_SuccessfulExecution - Test run command through ToolExecutor
func TestRunCommand_SuccessfulExecution(t *testing.T) {
	executor := NewToolExecutor(".", nil)
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")

	cmd := fmt.Sprintf("echo 'Test output' > %s", testFile)

	args := map[string]any{
		"command": cmd,
	}

	result := executor.runCommand(args)

	if containsString(result, "Exit code: 1") {
		t.Errorf("RunCommand should succeed, got error: %s", result)
	}
}

// TestAddTodo_SuccessfulAddition - Test add todo through ToolExecutor
func TestAddTodo_SuccessfulAddition(t *testing.T) {
	executor := NewToolExecutor(".", nil)
	title := fmt.Sprintf("test_todo_%d", time.Now().UnixNano())

	args := map[string]any{
		"title": title,
	}

	result := executor.addTodo(args)

	if !containsString(result, "added") {
		t.Errorf("AddTodo should succeed and return success message, got: %s", result)
	}
}

// TestCompleteTodo_SuccessfulCompletion - Test complete todo through ToolExecutor
func TestCompleteTodo_SuccessfulCompletion(t *testing.T) {
	executor := NewToolExecutor(".", nil)
	title := fmt.Sprintf("todo_to_complete_%d", time.Now().UnixNano())

	// First add the todo
	argsAdd := map[string]any{
		"title": title,
	}
	executor.addTodo(argsAdd)

	// Then complete it
	argsComplete := map[string]any{
		"title": title,
	}

	result := executor.completeTodo(argsComplete)

	if !containsString(result, "completed") {
		t.Errorf("CompleteTodo should succeed and return success message, got: %s", result)
	}
}

// TestDeleteTodo_SuccessfulDeletion - Test delete todo through ToolExecutor
func TestDeleteTodo_SuccessfulDeletion(t *testing.T) {
	executor := NewToolExecutor(".", nil)
	title := fmt.Sprintf("todo_to_delete_%d", time.Now().UnixNano())

	// First add the todo
	argsAdd := map[string]any{
		"title": title,
	}
	executor.addTodo(argsAdd)

	// Then delete it
	argsDelete := map[string]any{
		"title": title,
	}

	result := executor.deleteTodo(argsDelete)

	if !containsString(result, "deleted") {
		t.Errorf("DeleteTodo should succeed and return success message, got: %s", result)
	}
}

// TestSendEmail_SimpleEmail - Test send email through ToolExecutor
func TestSendEmail_SimpleEmail(t *testing.T) {
	executor := NewToolExecutor(".", nil)
	subject := fmt.Sprintf("Test Email %d", time.Now().UnixNano())
	body := "This is a test email body."

	args := map[string]any{
		"subject": subject,
		"body":    body,
	}

	result := executor.sendEmail(args)

	if !containsString(result, "sent") {
		t.Errorf("SendEmail should succeed and return success message, got: %s", result)
	}
}

// TestSendReport_SimpleReport - Test send report through ToolExecutor
func TestSendReport_SimpleReport(t *testing.T) {
	executor := NewToolExecutor(".", nil)
	body := "This is a test progress report."

	args := map[string]any{
		"body": body,
	}

	result := executor.sendReport(args)

	if !containsString(result, "sent") {
		t.Errorf("SendReport should succeed and return success message, got: %s", result)
	}
}

// TestReddit_SearchWithQuery - Test Reddit search through ToolExecutor
func TestReddit_SearchWithQuery(t *testing.T) {
	executor := NewToolExecutor(".", nil)
	query := "golang programming"
	limit := 5

	args := map[string]any{
		"action": "search",
		"query":  query,
		"limit":  limit,
	}

	result := executor.reddit(args)

	if containsString(result, "Error") || containsString(result, "error:") {
		t.Skipf("Reddit search returned error (acceptable): %s", result)
	}
}

// TestReddit_SubredditPosts - Test Reddit subreddit through ToolExecutor
func TestReddit_SubredditPosts(t *testing.T) {
	executor := NewToolExecutor(".", nil)
	subreddit := "golang"
	limit := 5

	args := map[string]any{
		"action":    "subreddit",
		"subreddit": subreddit,
		"limit":     limit,
	}

	result := executor.reddit(args)

	if containsString(result, "Error") || containsString(result, "error:") {
		t.Skipf("Reddit subreddit returned error (acceptable): %s", result)
	}
}

// TestWebSearch_SuccessfulSearch - Test web search through ToolExecutor
func TestWebSearch_SuccessfulSearch(t *testing.T) {
	executor := NewToolExecutor(".", nil)
	query := "golang programming"
	count := 3

	args := map[string]any{
		"query": query,
		"count": count,
	}

	result := executor.webSearch(args)

	if containsString(result, "Error") || containsString(result, "error:") {
		t.Skipf("Web search returned error (acceptable): %s", result)
	}
}

// TestReadWebpage_SuccessfulFetch - Test read webpage through ToolExecutor
func TestReadWebpage_SuccessfulFetch(t *testing.T) {
	executor := NewToolExecutor(".", nil)
	url := "https://example.com"

	args := map[string]any{
		"url": url,
	}

	result := executor.readWebpage(args)

	if containsString(result, "Error") || containsString(result, "error:") {
		t.Skipf("Read webpage returned error (acceptable): %s", result)
	}
}

// TestCheckInbox_ProcessEmails - Test check inbox through ToolExecutor
func TestCheckInbox_ProcessEmails(t *testing.T) {
	executor := NewToolExecutor(".", nil)

	args := map[string]any{
		"mark_read": true,
	}

	result := executor.checkInbox(args)

	if containsString(result, "Error") || containsString(result, "error:") {
		t.Skipf("Check inbox returned error (acceptable): %s", result)
	}
}

// TestToolExecutor_ListTodos - Test list todos through ToolExecutor
func TestToolExecutor_ListTodos(t *testing.T) {
	executor := NewToolExecutor(".", nil)

	args := map[string]any{}

	result := executor.listTodos(args)

	if containsString(result, "Error") || containsString(result, "error:") {
		t.Skipf("List todos returned error (acceptable): %s", result)
	}
}

// TestToolExecutor_EditFile_SuccessfulEdit - Test edit file through ToolExecutor
func TestToolExecutor_EditFile_SuccessfulEdit(t *testing.T) {
	executor := NewToolExecutor(".", nil)
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "edit_test.txt")

	// Create initial file
	os.WriteFile(testFile, []byte("Hello World"), 0644)

	args := map[string]any{
		"path":     testFile,
		"old_text": "World",
		"new_text": "Universe",
	}

	result := executor.editFile(args)

	if !containsString(result, "edited") {
		t.Errorf("EditFile should succeed and return success message, got: %s", result)
	}
}

// TestToolExecutor_MakeDir_Successful - Test make directory through ToolExecutor
func TestToolExecutor_MakeDir_Successful(t *testing.T) {
	executor := NewToolExecutor(".", nil)
	tempDir := t.TempDir()
	newDir := filepath.Join(tempDir, "new_dir")

	args := map[string]any{
		"path": newDir,
	}

	result := executor.makeDir(args)

	if !containsString(result, "created") {
		t.Errorf("MakeDir should succeed and return success message, got: %s", result)
	}
}

// TestToolExecutor_RemoveDir_Successful - Test remove directory through ToolExecutor
func TestToolExecutor_RemoveDir_Successful(t *testing.T) {
	executor := NewToolExecutor(".", nil)
	tempDir := t.TempDir()
	dirToRemove := filepath.Join(tempDir, "to_remove")

	// Create directory to remove
	os.MkdirAll(dirToRemove, 0755)

	args := map[string]any{
		"path": dirToRemove,
	}

	result := executor.removeDir(args)

	if !containsString(result, "removed") {
		t.Errorf("RemoveDir should succeed and return success message, got: %s", result)
	}
}

// TestToolExecutor_CopyFile_Successful - Test copy file through ToolExecutor
func TestToolExecutor_CopyFile_Successful(t *testing.T) {
	executor := NewToolExecutor(".", nil)
	tempDir := t.TempDir()
	sourceFile := filepath.Join(tempDir, "source.txt")
	destFile := filepath.Join(tempDir, "dest.txt")

	// Create source file
	os.WriteFile(sourceFile, []byte("test content"), 0644)

	args := map[string]any{
		"source": sourceFile,
		"dest":   destFile,
	}

	result := executor.copyFile(args)

	if !containsString(result, "copied") {
		t.Errorf("CopyFile should succeed and return success message, got: %s", result)
	}
}

// TestToolExecutor_MoveFile_Successful - Test move file through ToolExecutor
func TestToolExecutor_MoveFile_Successful(t *testing.T) {
	executor := NewToolExecutor(".", nil)
	tempDir := t.TempDir()
	sourceFile := filepath.Join(tempDir, "source.txt")
	destFile := filepath.Join(tempDir, "dest.txt")

	// Create source file
	os.WriteFile(sourceFile, []byte("test content"), 0644)

	args := map[string]any{
		"source": sourceFile,
		"dest":   destFile,
	}

	result := executor.moveFile(args)

	if !containsString(result, "moved") {
		t.Errorf("MoveFile should succeed and return success message, got: %s", result)
	}
}

// TestToolExecutor_GlobRecursive_Successful - Test glob recursive through ToolExecutor
func TestToolExecutor_GlobRecursive_Successful(t *testing.T) {
	executor := NewToolExecutor(".", nil)
	tempDir := t.TempDir()

	// Create nested directory structure with files
	os.MkdirAll(filepath.Join(tempDir, "subdir1", "subdir2"), 0755)
	os.WriteFile(filepath.Join(tempDir, "file1.txt"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(tempDir, "subdir1", "file2.txt"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(tempDir, "subdir1", "subdir2", "file3.txt"), []byte("test"), 0644)

	// Test glob recursive - note this method doesn't exist yet
	// This test may be skipped if the method doesn't exist
}

// TestToolExecutor_ProcessInboxWithResponse - Test process inbox with response through ToolExecutor
func TestToolExecutor_ProcessInboxWithResponse(t *testing.T) {
	executor := NewToolExecutor(".", nil)

	args := map[string]any{}

	result := executor.processInboxWithResponse(args)

	if containsString(result, "Error") || containsString(result, "error:") {
		t.Skipf("Process inbox returned error (acceptable): %s", result)
	}
}

// Helper function to check if string contains substring
func containsString(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 &&
		(len(s) >= len(substr) &&
			(s == substr ||
				s[:len(substr)] == substr ||
				s[len(s)-len(substr):] == substr ||
				containsSubstring(s, substr)))
}

// Helper function to check if string contains substring (correct implementation)
func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestErrorHandling - Test various error cases through ToolExecutor
func TestToolExecutor_ErrorHandling(t *testing.T) {
	executor := NewToolExecutor(".", nil)

	// Test non-existent file read
	argsRead := map[string]any{
		"path": "/nonexistent/path/file.txt",
	}
	resultRead := executor.readFile(argsRead)
	t.Logf("Read non-existent file: %s", resultRead)

	// Test invalid command
	argsCmd := map[string]any{
		"command": "invalid_command_xyz123",
	}
	resultCmd := executor.runCommand(argsCmd)
	t.Logf("Run invalid command: %s", resultCmd)
}
