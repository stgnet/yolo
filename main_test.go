package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestStripAnsiCodes tests that ANSI escape codes are properly removed
func TestStripAnsiCodes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "plain text",
			input:    "Hello World",
			expected: "Hello World",
		},
		{
			name:     "simple color code",
			input:    "\x1b[32mGreen\x1b[0m",
			expected: "Green",
		},
		{
			name:     "bold and color",
			input:    "\x1b[1;36mBlue Bold\x1b[0m",
			expected: "Blue Bold",
		},
		{
			name:     "multiple codes",
			input:    "\x1b[32mGreen\x1b[0m and \x1b[33mYellow\x1b[0m",
			expected: "Green and Yellow",
		},
		{
			name:     "newline with colors",
			input:    "\x1b[32mLine1\x1b[0m\n\x1b[33mLine2\x1b[0m",
			expected: "Line1\nLine2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stripAnsiCodes(tt.input)
			if result != tt.expected {
				t.Errorf("stripAnsiCodes(%q) = %q; want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestStripANSI tests the alternative strip function
func TestStripANSI(t *testing.T) {
	input := "\x1b[32mGreen\x1b[0m"
	expected := "Green"
	result := stripANSI(input)
	if result != expected {
		t.Errorf("stripANSI(%q) = %q; want %q", input, result, expected)
	}
}

// TestGetEnvDefault tests environment variable handling
func TestGetEnvDefault(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		fallback string
		expected string
	}{
		{
			name:     "unset key with fallback",
			key:      "UNSET_VAR_TEST_12345",
			fallback: "default",
			expected: "default",
		},
		{
			name:     "set key overrides fallback",
			key:      "YOLO_TEST_SET_KEY",
			fallback: "fallback_value",
			expected: "actual_value",
		},
		{
			name:     "empty env var uses fallback",
			key:      "YOLO_TEST_EMPTY_KEY",
			fallback: "fallback_for_empty",
			expected: "fallback_for_empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup: set env vars for tests that need them
			if tt.key == "YOLO_TEST_SET_KEY" {
				os.Setenv(tt.key, "actual_value")
				defer os.Unsetenv(tt.key)
			} else if tt.key == "YOLO_TEST_EMPTY_KEY" {
				os.Setenv(tt.key, "")
				defer os.Unsetenv(tt.key)
			}

			result := getEnvDefault(tt.key, tt.fallback)
			if result != tt.expected {
				t.Errorf("getEnvDefault(%q, %q) = %q; want %q", tt.key, tt.fallback, result, tt.expected)
			}
		})
	}
}

// TestToolExecutorReadFile tests the readFile tool
func TestToolExecutorReadFile(t *testing.T) {
	tmpDir := t.TempDir()
	executor := &ToolExecutor{baseDir: tmpDir}

	// Create a test file
	testFile := filepath.Join(tmpDir, "test.txt")
	content := "Hello World\nLine 2"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	result := executor.readFile(map[string]any{"path": "test.txt"})
	if !strings.Contains(result, "Hello World") {
		t.Errorf("Expected file content in result, got: %s", result)
	}

	// Test non-existent file
	result = executor.readFile(map[string]any{"path": "nonexistent.txt"})
	if !strings.Contains(result, "Error") {
		t.Errorf("Expected error for non-existent file, got: %s", result)
	}
}

// TestToolExecutorWriteFile tests the writeFile tool
func TestToolExecutorWriteFile(t *testing.T) {
	tmpDir := t.TempDir()
	executor := &ToolExecutor{baseDir: tmpDir}

	result := executor.writeFile(map[string]any{
		"path":    "test.txt",
		"content": "Test content here",
	})

	if !strings.Contains(result, "Wrote") {
		t.Errorf("Expected success message, got: %s", result)
	}

	// Verify file was created
	data, err := os.ReadFile(filepath.Join(tmpDir, "test.txt"))
	if err != nil {
		t.Fatal(err)
	}
	expected := "Test content here"
	if string(data) != expected {
		t.Errorf("Expected %q, got %q", expected, string(data))
	}

	// Test missing path argument
	result = executor.writeFile(map[string]any{"content": "test"})
	if !strings.Contains(result, "Error") || !strings.Contains(result, "path is required") {
		t.Errorf("Expected error for missing path, got: %s", result)
	}
}

// TestToolExecutorEditFile tests the editFile tool
func TestToolExecutorEditFile(t *testing.T) {
	tmpDir := t.TempDir()
	executor := &ToolExecutor{baseDir: tmpDir}

	// Create a test file
	testFile := filepath.Join(tmpDir, "test.txt")
	initialContent := "Hello World\nThis is a test"
	if err := os.WriteFile(testFile, []byte(initialContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Edit the file - basic replacement
	result := executor.editFile(map[string]any{
		"path":     "test.txt",
		"old_text": "World",
		"new_text": "Universe",
	})

	if !strings.Contains(result, "Edited") {
		t.Errorf("Expected success message, got: %s", result)
	}

	// Verify file was edited
	data, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}
	expectedAfterEdit := "Hello Universe\nThis is a test"
	if string(data) != expectedAfterEdit {
		t.Errorf("Expected %q, got %q", expectedAfterEdit, string(data))
	}

	// Test non-existent old_text
	result = executor.editFile(map[string]any{
		"path":     "test.txt",
		"old_text": "NonExistentText12345",
		"new_text": "Replacement",
	})
	if !strings.Contains(result, "Error") || !strings.Contains(result, "not found") {
		t.Errorf("Expected error for non-existent old_text, got: %s", result)
	}
}

// TestToolExecutorListFiles tests the listFiles tool
func TestToolExecutorListFiles(t *testing.T) {
	tmpDir := t.TempDir()
	executor := &ToolExecutor{baseDir: tmpDir}

	// Create test files and directories
	os.MkdirAll(filepath.Join(tmpDir, "subdir"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("content1"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "file2.go"), []byte("content2"), 0644)
	os.WriteFile(filepath.Join(tmpDir, ".hidden"), []byte("hidden"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "subdir", "nested.txt"), []byte("nested"), 0644)

	// Test listing all files
	result := executor.listFiles(map[string]any{"pattern": "*"})

	// Should find file1.txt and file2.go
	if !strings.Contains(result, "file1.txt") {
		t.Errorf("Expected file1.txt in result, got: %s", result)
	}
	if !strings.Contains(result, "file2.go") {
		t.Errorf("Expected file2.go in result, got: %s", result)
	}

	// Test listing specific pattern
	result = executor.listFiles(map[string]any{"pattern": "*.go"})
	if !strings.Contains(result, "file2.go") {
		t.Errorf("Expected file2.go with *.go pattern, got: %s", result)
	}
	if strings.Contains(result, "file1.txt") {
		t.Errorf("Did not expect file1.txt with *.go pattern, got: %s", result)
	}
}

// TestOllamaClient tests client initialization
func TestOllamaClient(t *testing.T) {
	// Test default URL
	client := NewOllamaClient("", "")
	if client.baseURL != DefaultOllamaURL {
		t.Errorf("Expected default URL %q, got %q", DefaultOllamaURL, client.baseURL)
	}

	// Test with custom URL and key
	customURL := "http://custom:11434"
	client = NewOllamaClient(customURL, "test-key-abc")
	if client.baseURL != customURL {
		t.Errorf("Expected custom URL %q, got %q", customURL, client.baseURL)
	}
	if client.apiKey != "test-key-abc" {
		t.Errorf("Expected API key 'test-key-abc', got %q", client.apiKey)
	}

	// Test with empty strings (should use defaults)
	client = NewOllamaClient("", "")
	if client.baseURL != DefaultOllamaURL {
		t.Errorf("Expected default URL, got %q", client.baseURL)
	}
}

// TestYoloAgent tests agent initialization
func TestYoloAgent(t *testing.T) {
	// Test default configuration
	agent := NewYoloAgent()

	if agent.thinkDelay != ThinkLoopDelay {
		t.Errorf("Expected think delay %d, got %d", ThinkLoopDelay, agent.thinkDelay)
	}

	if agent.userPrompt != UserPromptDefault {
		t.Errorf("Expected default user prompt, got: %s", agent.userPrompt)
	}

	if agent.maxTokens <= 0 {
		t.Error("Expected maxTokens to be positive")
	}

	if len(agent.tools) == 0 {
		t.Error("Expected at least one tool to be registered")
	}
}
