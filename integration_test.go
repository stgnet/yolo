package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestYOLOAutonomousWorkflow tests the complete autonomous workflow
func TestYOLOAutonomousWorkflow(t *testing.T) {
	tempDir := t.TempDir()

	executor := NewToolExecutor(tempDir, nil)
	if executor == nil {
		t.Fatal("Failed to create ToolExecutor")
	}

	tests := []struct {
		name           string
		task           string
		expectedFiles  []string
		expectedExists bool
	}{
		{
			name:           "create new file",
			task:           "Create a test file at 'test_output/hello.txt' with content 'Hello World'",
			expectedFiles:  []string{"test_output/hello.txt"},
			expectedExists: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Execute task using available tools
			parts := strings.Split(tt.task, " at ")
			if len(parts) >= 2 {
				filePath := strings.Trim(parts[1], "'")
				contentParts := strings.Split(tt.task, " with content '")
				if len(contentParts) >= 2 {
					content := strings.TrimSuffix(contentParts[1], "'")

					// Use write_file tool
					result := executor.writeFile(map[string]any{
						"path":    filePath,
						"content": content,
					})

					if strings.Contains(result, "Error:") {
						t.Errorf("write_file failed: %s", result)
					}

					// Verify file exists
					fullPath := filepath.Join(tempDir, filePath)
					if _, err := os.Stat(fullPath); os.IsNotExist(err) {
						t.Errorf("File was not created at %s", fullPath)
					} else {
						t.Logf("Successfully created file at %s", fullPath)
					}
				}
			}
		})
	}
}

// TestToolChaining tests chaining multiple tools together
func TestToolChaining(t *testing.T) {
	tempDir := t.TempDir()
	executor := NewToolExecutor(tempDir, nil)
	if executor == nil {
		t.Fatal("Failed to create ToolExecutor")
	}

	// Step 1: Create a directory
	mkdirResult := executor.makeDir(map[string]any{"path": "test_chain/subdir"})
	if strings.Contains(mkdirResult, "Error:") {
		t.Errorf("makeDir failed: %s", mkdirResult)
	}

	// Step 2: Write a file in that directory
	writeResult := executor.writeFile(map[string]any{
		"path":    "test_chain/subdir/data.txt",
		"content": "Test content",
	})
	if strings.Contains(writeResult, "Error:") {
		t.Errorf("writeFile failed: %s", writeResult)
	}

	// Step 3: Read the file back
	readResult := executor.readFile(map[string]any{
		"path": "test_chain/subdir/data.txt",
	})
	if strings.Contains(readResult, "Error:") || !strings.Contains(readResult, "Test content") {
		t.Errorf("readFile failed or returned wrong content: %s", readResult)
	}

	// Step 4: Search for the content
	searchResult := executor.searchFiles(map[string]any{
		"query":   "Test content",
		"pattern": "test_chain/*",
	})
	if strings.Contains(searchResult, "Error:") {
		t.Errorf("searchFiles failed: %s", searchResult)
	}

	// Step 5: List files
	listResult := executor.listFiles(map[string]any{
		"pattern": "test_chain/*",
	})
	if strings.Contains(listResult, "Error:") {
		t.Errorf("listFiles failed: %s", listResult)
	}

	t.Log("All tool chaining steps completed successfully")
}

// TestWebSearchIntegration tests the web_search tool end-to-end
func TestWebSearchIntegration(t *testing.T) {
	t.Skip("Skipping network-dependent integration test")

	executor := NewToolExecutor("/tmp/test", nil)
	if executor == nil {
		t.Fatal("Failed to create ToolExecutor")
	}

	// Test web search with a known query
	result := executor.webSearch(map[string]any{
		"query": "Go programming language",
		"count": 3,
	})

	if strings.Contains(result, "Error:") {
		t.Errorf("webSearch failed: %s", result)
	}

	if !strings.Contains(result, "Go") && !strings.Contains(result, "go") {
		t.Errorf("webSearch result doesn't contain expected content:\n%s", result[:min(200, len(result))])
	}

	t.Logf("Web search successful, got %d chars", len(result))
}

// TestRedditIntegration tests the Reddit API integration
func TestRedditIntegration(t *testing.T) {
	t.Skip("Skipping network-dependent integration test")

	executor := NewToolExecutor("/tmp/test", nil)
	if executor == nil {
		t.Fatal("Failed to create ToolExecutor")
	}

	// Test Reddit search
	result := executor.reddit(map[string]any{
		"action": "search",
		"query":  "golang programming",
		"limit":  2,
	})

	if strings.Contains(result, "Error:") {
		t.Errorf("reddit search failed: %s", result)
	}

	if result == "" {
		t.Error("Reddit search returned empty result")
	}

	t.Logf("Reddit search successful, got %d chars", len(result))
}

// TestSubagentSpawn tests subagent creation and execution
func TestSubagentSpawn(t *testing.T) {
	executor := NewToolExecutor("/tmp/test", nil)
	if executor == nil {
		t.Fatal("Failed to create ToolExecutor")
	}

	// Note: spawnSubagent requires an agent context which isn't available in isolated tests
	// This test validates that listSubagents works correctly
	listResult := executor.listSubagents(map[string]any{})
	if strings.Contains(listResult, "Error:") {
		t.Errorf("listSubagents failed: %s", listResult)
	}

	t.Log("Subagent list successful")
}

func TestErrorHandling(t *testing.T) {
	tempDir := t.TempDir()
	executor := NewToolExecutor(tempDir, nil)
	if executor == nil {
		t.Fatal("Failed to create ToolExecutor")
	}

	tests := []struct {
		name                 string
		action               func() string
		expectErrorSubstring string
	}{
		{
			name: "write to invalid path",
			action: func() string {
				return executor.writeFile(map[string]any{
					"path":    "/invalid/path/file.txt",
					"content": "test",
				})
			},
			expectErrorSubstring: "Error:",
		},
		{
			name: "read non-existent file",
			action: func() string {
				return executor.readFile(map[string]any{
					"path": "non_existent_file.txt",
				})
			},
			expectErrorSubstring: "no such file or directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.action()
			if !strings.Contains(result, tt.expectErrorSubstring) {
				t.Errorf("Expected result to contain '%s' but got: %s", tt.expectErrorSubstring, result[:min(200, len(result))])
			}
		})
	}
}
