package tools

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestFileReadTool(t *testing.T) {
	tool := &FileReadTool{}
	
	// Use README.md from repo root (relative to tools directory)
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"path": "../README.md",
	})
	
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	
	if !result.Success {
		t.Errorf("Expected success, got error: %s", result.Error)
	}
	
	if len(result.Output) == 0 {
		t.Error("Expected output to contain file contents")
	}
}

func TestFileWriteTool(t *testing.T) {
	tool := &FileWriteTool{}
	testPath := "test_tools/test_write_file.txt"
	
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"path":    testPath,
		"content": "Test content for file write",
	})
	
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	
	if !result.Success {
		t.Errorf("Expected success, got error: %s", result.Error)
	}
	
	// Cleanup
	defer func() {
		_ = os.Remove(testPath)
	}()
}

func TestMakeDirTool(t *testing.T) {
	tool := &MakeDirTool{}
	testPath := "test_tools/test_make_dir"
	
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"path": testPath,
	})
	
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	
	if !result.Success {
		t.Errorf("Expected success, got error: %s", result.Error)
	}
	
	// Cleanup
	defer func() {
		_ = executeCommand("rm -rf", []string{testPath})
	}()
}

func TestRunCommandTool(t *testing.T) {
	tool := &RunCommandTool{}
	
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"command": "echo 'Hello World'",
	})
	
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	
	if !result.Success {
		t.Errorf("Expected success, got error: %s", result.Error)
	}
	
	if result.Output != "Hello World\n" {
		t.Errorf("Expected 'Hello World\\n', got '%s'", result.Output)
	}
}

func TestWebSearchTool(t *testing.T) {
	tool := &WebSearchTool{}
	
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"query": "golang",
		"count": 3,
	})
	
	if result == nil || !result.Success {
		t.Skip("Web search skipped - may fail without network")
		return
	}
	
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
}

func TestRedditTool(t *testing.T) {
	tool := &RedditTool{}
	
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"action": "subreddit",
		"subreddit": "golang",
		"limit": 5,
	})
	
	if result == nil || !result.Success {
		t.Skip("Reddit API skipped - may fail without network")
		return
	}
	
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
}

func TestAddTodoTool(t *testing.T) {
	tool := &AddTodoTool{}
	testTitle := "Test todo item - " + time.Now().Format("15:04:05")
	
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"title": testTitle,
	})
	
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	
	if !result.Success {
		t.Errorf("Expected success, got error: %s", result.Error)
	}
	
	// Cleanup - delete the todo
	deleteTodoTool := &DeleteTodoTool{}
	_, _ = deleteTodoTool.Execute(context.Background(), map[string]interface{}{
		"title": testTitle,
	})
}

func TestListTodosTool(t *testing.T) {
	tool := &ListTodosTool{}
	
	result, err := tool.Execute(context.Background(), nil)
	
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	
	if !result.Success {
		t.Errorf("Expected success, got error: %s", result.Error)
	}
}

func TestListModelsTool(t *testing.T) {
	tool := &ListModelsTool{}
	
	result, err := tool.Execute(context.Background(), nil)
	
	// This may fail if ollama is not installed
	if !result.Success {
		t.Skip("Model listing skipped - ollama may not be available")
		return
	}
	
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
}

func TestCopyFileTool(t *testing.T) {
	tool := &CopyFileTool{}
	source := "../go.mod"
	dest := "test_tools/test_copy_file.go.mod"
	
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"source": source,
		"dest":   dest,
	})
	
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	
	if !result.Success {
		t.Errorf("Expected success, got error: %s", result.Error)
	}
	
	// Cleanup
	defer func() {
		_ = os.Remove(dest)
	}()
}

func TestMoveFileTool(t *testing.T) {
	tool := &MoveFileTool{}

	// Create a temporary source file instead of moving go.mod
	source := "test_tools/test_move_source.txt"
	dest := "test_tools/test_move_dest.txt"
	_ = os.MkdirAll("test_tools", 0755)
	_ = os.WriteFile(source, []byte("move test content"), 0644)
	defer func() {
		_ = os.Remove(source)
		_ = os.Remove(dest)
	}()

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"source": source,
		"dest":   dest,
	})

	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got error: %s", result.Error)
	}
}

func TestReadWebpageTool(t *testing.T) {
	tool := &ReadWebpageTool{}
	
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"url": "https://example.com",
	})
	
	if result == nil || !result.Success {
		t.Skip("Web reading skipped - may fail without network")
		return
	}
	
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	
	if len(result.Output) == 0 {
		t.Error("Expected output to contain webpage content")
	}
}

// Helper function for executing shell commands in tests
func executeCommand(cmd string, args []string) error {
	// Implementation depends on specific needs
	return nil
}
