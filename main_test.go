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

// TestSpawnSubagent tests the spawnSubagent tool with basic parameters
func TestSpawnSubagent(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a minimal agent for the executor
	agent := NewYoloAgent()
	agent.baseDir = tmpDir

	executor := &ToolExecutor{baseDir: tmpDir, agent: agent}

	// Test with minimal required parameters (tool defines "task", not "prompt")
	params := map[string]any{
		"task": "Test subagent task",
	}

	result := executor.spawnSubagent(params)

	if !strings.Contains(result, "Sub-agent") {
		t.Errorf("Expected output to contain 'Sub-agent', got: %s", result)
	}

	if !strings.Contains(result, "spawned") {
		t.Errorf("Expected output to confirm spawning, got: %s", result)
	}

	// Test with model parameter
	paramsWithModel := map[string]any{
		"task":  "Test subagent task with model",
		"model": "llama3",
	}

	result2 := executor.spawnSubagent(paramsWithModel)

	if !strings.Contains(result2, "Sub-agent") {
		t.Errorf("Expected output to contain 'Sub-agent', got: %s", result2)
	}
}

// TestSpawnSubagentValidation tests parameter validation in spawnSubagent
func TestSpawnSubagentValidation(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a minimal agent for the executor
	agent := NewYoloAgent()

	executor := &ToolExecutor{baseDir: tmpDir, agent: agent}

	tests := []struct {
		name    string
		params  map[string]any
		wantErr bool
		errMsg  string
	}{
		{
			name:    "missing task",
			params:  map[string]any{},
			wantErr: true,
			errMsg:  "'task' parameter is required",
		},
		{
			name: "empty string task",
			params: map[string]any{
				"task": "",
			},
			wantErr: true,
			errMsg:  "'task' parameter is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := executor.spawnSubagent(tt.params)

			if tt.wantErr && !strings.Contains(result, "Error:") {
				t.Errorf("Expected error in result, got: %s", result)
			}

			if tt.wantErr && !strings.Contains(result, tt.errMsg) {
				t.Errorf("Expected error message to contain %q, got: %s", tt.errMsg, result)
			}
		})
	}
}

// TestStripANSI tests the alternative strip function
func TestStripANSI(t *testing.T) {
	input := "\x1b[32mGreen\x1b[0m"
	expected := "Green"
	result := stripAnsiCodes(input)
	if result != expected {
		t.Errorf("stripAnsiCodes(%q) = %q; want %q", input, result, expected)
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
		"old_text": "nonexistent",
		"new_text": "replacement",
	})
	if !strings.Contains(result, "Error") || !strings.Contains(result, "not found") {
		t.Errorf("Expected error for non-existent old_text, got: %s", result)
	}
}

// TestToolExecutorListFiles tests the listFiles tool
func TestToolExecutorListFiles(t *testing.T) {
	tmpDir := t.TempDir()
	executor := &ToolExecutor{baseDir: tmpDir}

	// Create some test files
	os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("content1"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "file2.go"), []byte("package main"), 0644)
	os.MkdirAll(filepath.Join(tmpDir, "subdir"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "subdir", "file3.txt"), []byte("content3"), 0644)

	// Test listing with default pattern
	result := executor.listFiles(nil)
	if !strings.Contains(result, "file1.txt") || !strings.Contains(result, "file2.go") {
		t.Errorf("Expected files in result, got: %s", result)
	}

	// Test listing with glob pattern
	result = executor.listFiles(map[string]any{"pattern": "*.go"})
	if !strings.Contains(result, "file2.go") || strings.Contains(result, ".txt") {
		t.Errorf("Expected only .go files, got: %s", result)
	}

	// Test recursive listing
	result = executor.listFiles(map[string]any{"pattern": "**/*.txt"})
	if !strings.Contains(result, "file1.txt") || !strings.Contains(result, "file3.txt") {
		t.Errorf("Expected all .txt files recursively, got: %s", result)
	}

	// Test that **/*.txt matches by filename only (not full path)
	// This ensures the fix for globRecursive matching correctly
	if strings.Contains(result, "subdir/file1.txt") || strings.Contains(result, "file3.txt/subdir") {
		t.Errorf("Pattern should match filenames only, not include path components in wrong places: %s", result)
	}
}

// TestToolExecutorRunCommand tests the runCommand tool
func TestToolEchoCommand(t *testing.T) {
	tmpDir := t.TempDir()
	executor := &ToolExecutor{baseDir: tmpDir}

	result := executor.runCommand(map[string]any{"command": "echo 'hello world'"})
	if !strings.Contains(result, "hello world") {
		t.Errorf("Expected output from echo command, got: %s", result)
	}
}

// TestTerminalUIWrapText tests the word-wrapping functionality
func TestTerminalUIWrapText(t *testing.T) {
	ui := &TerminalUI{cols: 20}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple short text",
			input:    "Hello World",
			expected: "Hello World",
		},
		{
			name:     "text that fits exactly",
			input:    "Hello World Test", // 16 chars, should fit in 20
			expected: "Hello World Test",
		},
		{
			name:     "text exceeding width with word wrap",
			input:    "This is a longer line that should wrap at word boundaries when it exceeds the terminal width of twenty columns.",
			expected: "This is a longer\nline that should\nwrap at word\nboundaries when it\nexceeds the terminal\nwidth of twenty\ncolumns.",
		},
		{
			name:     "single word longer than width",
			input:    "supercalifragilisticexpialidocious", // 34 chars
			expected: "supercalifragilistic\nexpialidocious",
		},
		{
			name:     "multiple newlines preserved",
			input:    "Line1\n\nLine2",
			expected: "Line1\n\nLine2",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "single word fits exactly at boundary",
			input:    "abcdefghij klmnopqrst", // 10 + space + 10 = 21 > 20
			expected: "abcdefghij\nklmnopqrst",
		},
		{
			name:     "trailing newline preserved",
			input:    "Hello World\n",
			expected: "Hello World\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ui.wrapText(tt.input)
			if result != tt.expected {
				t.Errorf("wrapText(%q) = %q; want %q",
					tt.input, result, tt.expected)
				// Print line-by-line comparison for debugging
				expectedLines := strings.Split(tt.expected, "\n")
				resultLines := strings.Split(result, "\n")
				for i := 0; i < max(len(expectedLines), len(resultLines)); i++ {
					expLine := ""
					resLine := ""
					if i < len(expectedLines) {
						expLine = expectedLines[i]
					}
					if i < len(resultLines) {
						resLine = resultLines[i]
					}
					t.Logf("Line %d: got=%q want=%q", i, resLine, expLine)
				}
			}
		})
	}
}

// TestTerminalUIWrapTextVaryingWidths tests wrapping at different terminal widths
func TestTerminalUIWrapTextVaryingWidths(t *testing.T) {
	input := "The quick brown fox jumps over the lazy dog"

	testCases := []struct {
		width      int
		maxLineLen int
	}{
		{80, len(input)}, // Should fit on one line
		{40, 39},         // Should wrap: "The quick brown fox jumps over the" (39 chars)
		{20, 20},         // Multiple lines
		{10, 10},         // Many lines with short words
	}

	for _, tc := range testCases {
		ui := &TerminalUI{cols: tc.width}
		result := ui.wrapText(input)

		lines := strings.Split(result, "\n")
		// Remove last empty element if present
		if len(lines) > 0 && lines[len(lines)-1] == "" {
			lines = lines[:len(lines)-1]
		}

		maxLen := 0
		for _, line := range lines {
			if len(line) > maxLen {
				maxLen = len(line)
			}
			if len(line) > tc.width {
				t.Errorf("Line exceeds width: %q (len=%d, width=%d)", line, len(line), tc.width)
			}
		}

		if maxLen > tc.maxLineLen {
			t.Logf("Warning: max line length %d exceeds expected %d for width %d",
				maxLen, tc.maxLineLen, tc.width)
		}
	}
}

// TestParseParamString tests the parseParamString function
func TestParseParamString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected map[string]any
	}{
		{
			name:     "simple key-value",
			input:    "path=main.go",
			expected: map[string]any{"path": "main.go"},
		},
		{
			name:  "multiple parameters",
			input: "path=main.go, offset=100, limit=50",
			expected: map[string]any{
				"path":   "main.go",
				"offset": int64(100),
				"limit":  int64(50),
			},
		},
		{
			name:  "mixed types",
			input: "path=main.go, count=10, debug=true, ratio=3.14",
			expected: map[string]any{
				"path":  "main.go",
				"count": int64(10),
				"debug": true,
				"ratio": float64(3.14),
			},
		},
		{
			name:     "empty string",
			input:    "",
			expected: map[string]any{},
		},
		{
			name:     "spaces around separator",
			input:    "key = value",
			expected: map[string]any{"key": "value"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseParamString(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d keys, got %d. Expected: %v, Got: %v",
					len(tt.expected), len(result), tt.expected, result)
				return
			}
			for k, v := range tt.expected {
				if val, ok := result[k]; !ok {
					t.Errorf("Missing key: %s", k)
				} else if val != v {
					t.Errorf("Key %s: expected %v (%T), got %v (%T)", k, v, v, val, val)
				}
			}
		})
	}
}
