package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// isError checks if result string indicates an error
func isError(result string) bool {
	return strings.HasPrefix(result, "Error:") ||
		strings.HasPrefix(result, "error:") ||
		strings.Contains(result, "cannot")
}

func TestIsBinaryData(t *testing.T) {
	tests := []struct {
		name   string
		data   []byte
		binary bool
	}{
		{"empty", []byte{}, false},
		{"plain text", []byte("Hello, world!\n"), false},
		{"text with tabs", []byte("col1\tcol2\tcol3\n"), false},
		{"null byte", []byte("hello\x00world"), true},
		{"binary header", []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}, true},
		{"mostly text with some control chars", []byte("normal text here"), false},
		{"high non-text ratio", func() []byte {
			b := make([]byte, 100)
			for i := range b {
				b[i] = 1 // non-text control char
			}
			return b
		}(), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isBinaryData(tt.data)
			if got != tt.binary {
				t.Errorf("isBinaryData() = %v, want %v", got, tt.binary)
			}
		})
	}
}

func TestSearchFiles(t *testing.T) {
	tmpDir := t.TempDir()
	executor := &ToolExecutor{baseDir: tmpDir}

	// Create test files
	os.WriteFile(filepath.Join(tmpDir, "hello.txt"), []byte("line1 foo\nline2 bar\nline3 foo bar\n"), 0o644)
	os.MkdirAll(filepath.Join(tmpDir, "sub"), 0o755)
	os.WriteFile(filepath.Join(tmpDir, "sub", "deep.go"), []byte("package main\nfunc foo() {}\n"), 0o644)

	t.Run("basic search", func(t *testing.T) {
		result := executor.searchFiles(map[string]any{"query": "foo"})
		if strings.Contains(result, "No matches") {
			t.Errorf("Expected matches, got: %s", result)
		}
		if !strings.Contains(result, "hello.txt:1") {
			t.Errorf("Expected hello.txt:1 in results, got: %s", result)
		}
	})

	t.Run("no matches", func(t *testing.T) {
		result := executor.searchFiles(map[string]any{"query": "zzzznotfound"})
		if result != "No matches found" {
			t.Errorf("Expected 'No matches found', got: %s", result)
		}
	})

	t.Run("missing query", func(t *testing.T) {
		result := executor.searchFiles(map[string]any{})
		if !strings.HasPrefix(result, "Error") {
			t.Errorf("Expected error for missing query, got: %s", result)
		}
	})

	t.Run("invalid regex", func(t *testing.T) {
		result := executor.searchFiles(map[string]any{"query": "[invalid"})
		if !strings.Contains(result, "invalid regex") {
			t.Errorf("Expected regex error, got: %s", result)
		}
	})

	t.Run("pattern filter", func(t *testing.T) {
		result := executor.searchFiles(map[string]any{"query": "foo", "pattern": "*.go"})
		if !strings.Contains(result, "deep.go") {
			t.Errorf("Expected deep.go in filtered results, got: %s", result)
		}
		if strings.Contains(result, "hello.txt") {
			t.Errorf("hello.txt should be filtered out, got: %s", result)
		}
	})
}

func TestCopyFile(t *testing.T) {
	t.Run("basic copy", func(t *testing.T) {
		tmpDir := t.TempDir()
		executor := &ToolExecutor{baseDir: tmpDir}
		os.WriteFile(filepath.Join(tmpDir, "src.txt"), []byte("content"), 0o644)

		result := executor.copyFile(map[string]any{"source": "src.txt", "dest": "dst.txt"})
		if !strings.Contains(result, "Copied") {
			t.Errorf("Expected success, got: %s", result)
		}
		// Verify both exist
		if _, err := os.Stat(filepath.Join(tmpDir, "src.txt")); err != nil {
			t.Error("Source should still exist after copy")
		}
		data, err := os.ReadFile(filepath.Join(tmpDir, "dst.txt"))
		if err != nil {
			t.Fatalf("Dest should exist: %v", err)
		}
		if string(data) != "content" {
			t.Errorf("Content mismatch: %q", data)
		}
	})

	t.Run("copy to nested dir", func(t *testing.T) {
		tmpDir := t.TempDir()
		executor := &ToolExecutor{baseDir: tmpDir}
		os.WriteFile(filepath.Join(tmpDir, "src.txt"), []byte("data"), 0o644)

		result := executor.copyFile(map[string]any{"source": "src.txt", "dest": "a/b/c/dst.txt"})
		if !strings.Contains(result, "Copied") {
			t.Errorf("Expected success, got: %s", result)
		}
	})

	t.Run("missing source", func(t *testing.T) {
		tmpDir := t.TempDir()
		executor := &ToolExecutor{baseDir: tmpDir}
		result := executor.copyFile(map[string]any{"source": "", "dest": "dst.txt"})
		if !isError(result) {
			t.Errorf("Expected error, got: %s", result)
		}
	})

	t.Run("missing dest", func(t *testing.T) {
		tmpDir := t.TempDir()
		executor := &ToolExecutor{baseDir: tmpDir}
		result := executor.copyFile(map[string]any{"source": "src.txt", "dest": ""})
		if !isError(result) {
			t.Errorf("Expected error, got: %s", result)
		}
	})

	t.Run("source not found", func(t *testing.T) {
		tmpDir := t.TempDir()
		executor := &ToolExecutor{baseDir: tmpDir}
		result := executor.copyFile(map[string]any{"source": "nope.txt", "dest": "dst.txt"})
		if !strings.Contains(result, "does not exist") {
			t.Errorf("Expected 'does not exist' error, got: %s", result)
		}
	})

	t.Run("copy directory fails", func(t *testing.T) {
		tmpDir := t.TempDir()
		executor := &ToolExecutor{baseDir: tmpDir}
		os.MkdirAll(filepath.Join(tmpDir, "mydir"), 0o755)
		result := executor.copyFile(map[string]any{"source": "mydir", "dest": "dst"})
		if !isError(result) {
			t.Errorf("Expected error when copying directory, got: %s", result)
		}
	})
}

func TestExecuteDispatcher(t *testing.T) {
	tmpDir := t.TempDir()
	executor := &ToolExecutor{baseDir: tmpDir}

	t.Run("unknown tool", func(t *testing.T) {
		result := executor.Execute("nonexistent_tool", map[string]any{})
		if !strings.Contains(result, "unknown tool") {
			t.Errorf("Expected unknown tool error, got: %s", result)
		}
	})

	t.Run("think tool", func(t *testing.T) {
		result := executor.Execute("think", map[string]any{"thought": "test"})
		if result != "Thought recorded." {
			t.Errorf("Expected 'Thought recorded.', got: %s", result)
		}
	})

	t.Run("read_file dispatches", func(t *testing.T) {
		os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("hello"), 0o644)
		result := executor.Execute("read_file", map[string]any{"path": "test.txt"})
		if !strings.Contains(result, "hello") {
			t.Errorf("Expected file content, got: %s", result)
		}
	})

	t.Run("write_file dispatches", func(t *testing.T) {
		result := executor.Execute("write_file", map[string]any{"path": "new.txt", "content": "data"})
		if !strings.Contains(result, "Wrote") {
			t.Errorf("Expected write confirmation, got: %s", result)
		}
	})

	t.Run("list_files dispatches", func(t *testing.T) {
		result := executor.Execute("list_files", map[string]any{"pattern": "*"})
		if isError(result) {
			t.Errorf("Expected success, got: %s", result)
		}
	})

	t.Run("make_dir dispatches", func(t *testing.T) {
		result := executor.Execute("make_dir", map[string]any{"path": "testdir"})
		if !strings.Contains(result, "Created directory") {
			t.Errorf("Expected success, got: %s", result)
		}
	})

	t.Run("search_files dispatches", func(t *testing.T) {
		os.WriteFile(filepath.Join(tmpDir, "search_me.txt"), []byte("this is a test search query"), 0o644)
		result := executor.Execute("search_files", map[string]any{"query": "test search"})
		if !strings.Contains(result, "search_me.txt") {
			t.Errorf("Expected search_me.txt in results, got: %s", result)
		}
	})

	t.Run("move_file dispatches", func(t *testing.T) {
		os.WriteFile(filepath.Join(tmpDir, "to_move.txt"), []byte("moving me"), 0o644)
		result := executor.Execute("move_file", map[string]any{"source": "to_move.txt", "dest": "moved.txt"})
		if !strings.Contains(result, "moved") {
			t.Errorf("Expected move confirmation, got: %s", result)
		}
		// Verify source is gone
		if _, err := os.Stat(filepath.Join(tmpDir, "to_move.txt")); err == nil {
			t.Error("Source file should be moved (not exist)")
		}
		// Verify dest exists
		if _, err := os.Stat(filepath.Join(tmpDir, "moved.txt")); err != nil {
			t.Errorf("Dest file should exist: %v", err)
		}
	})

	t.Run("remove_dir dispatches", func(t *testing.T) {
		testDir := filepath.Join(tmpDir, "to_remove")
		os.MkdirAll(testDir, 0o755)
		result := executor.Execute("remove_dir", map[string]any{"path": "to_remove"})
		if !strings.Contains(result, "Removed") {
			t.Errorf("Expected remove confirmation, got: %s", result)
		}
		if _, err := os.Stat(testDir); err == nil {
			t.Error("Directory should be removed")
		}
	})

	t.Run("edit_file dispatches", func(t *testing.T) {
		testFile := filepath.Join(tmpDir, "edit_me.txt")
		initialContent := "before text after"
		os.WriteFile(testFile, []byte(initialContent), 0o644)
		result := executor.Execute("edit_file", map[string]any{
			"path":     "edit_me.txt",
			"old_text": "text",
			"new_text": "replaced",
		})
		if !strings.Contains(result, "Edited") {
			t.Errorf("Expected edit confirmation, got: %s", result)
		}
		// Verify content was changed
		content, err := os.ReadFile(testFile)
		if err != nil {
			t.Fatalf("Failed to read edited file: %v", err)
		}
		if string(content) != "before replaced after" {
			t.Errorf("Content not edited correctly, got: %q", content)
		}
	})

	t.Run("run_command dispatches", func(t *testing.T) {
		result := executor.Execute("run_command", map[string]any{"command": "echo 'run test'"})
		if !strings.Contains(result, "run test") && !strings.Contains(result, "Error") {
			t.Errorf("Expected command output or error, got: %s", result)
		}
	})

	t.Run("empty_tool_name", func(t *testing.T) {
		result := executor.Execute("", map[string]any{})
		if !strings.Contains(result, "unknown tool") {
			t.Errorf("Expected unknown tool error, got: %s", result)
		}
	})
}

func TestGetStringArg(t *testing.T) {
	args := map[string]any{
		"str":   "hello",
		"num":   42,
		"float": 3.14,
	}

	if v := getStringArg(args, "str", ""); v != "hello" {
		t.Errorf("Expected 'hello', got %q", v)
	}
	if v := getStringArg(args, "num", ""); v != "42" {
		t.Errorf("Expected '42', got %q", v)
	}
	if v := getStringArg(args, "missing", "default"); v != "default" {
		t.Errorf("Expected 'default', got %q", v)
	}
}

func TestGetIntArg(t *testing.T) {
	args := map[string]any{
		"float": 42.0,
		"int":   10,
		"str":   "5",
		"bad":   "notanumber",
	}

	if v := getIntArg(args, "float", 0); v != 42 {
		t.Errorf("Expected 42, got %d", v)
	}
	if v := getIntArg(args, "int", 0); v != 10 {
		t.Errorf("Expected 10, got %d", v)
	}
	if v := getIntArg(args, "str", 0); v != 5 {
		t.Errorf("Expected 5, got %d", v)
	}
	if v := getIntArg(args, "missing", 99); v != 99 {
		t.Errorf("Expected 99, got %d", v)
	}
	if v := getIntArg(args, "bad", 7); v != 7 {
		t.Errorf("Expected fallback 7, got %d", v)
	}
}

func TestReadFileBinary(t *testing.T) {
	tmpDir := t.TempDir()
	executor := &ToolExecutor{baseDir: tmpDir}

	// Write a binary file
	os.WriteFile(filepath.Join(tmpDir, "binary.dat"), []byte{0x89, 0x50, 0x4E, 0x47, 0x00, 0x01, 0x02}, 0o644)
	result := executor.readFile(map[string]any{"path": "binary.dat"})
	if !strings.Contains(result, "binary file") {
		t.Errorf("Expected binary file error, got: %s", result)
	}
}

func TestReadFileOffsetLimit(t *testing.T) {
	tmpDir := t.TempDir()
	executor := &ToolExecutor{baseDir: tmpDir}

	content := "line1\nline2\nline3\nline4\nline5\n"
	os.WriteFile(filepath.Join(tmpDir, "lines.txt"), []byte(content), 0o644)

	result := executor.readFile(map[string]any{"path": "lines.txt", "offset": 2.0, "limit": 2.0})
	if !strings.Contains(result, "line2") {
		t.Errorf("Expected line2 in output, got: %s", result)
	}
	if !strings.Contains(result, "line3") {
		t.Errorf("Expected line3 in output, got: %s", result)
	}
	// Should show continuation hint
	if !strings.Contains(result, "offset=4") {
		t.Errorf("Expected offset hint, got: %s", result)
	}
}
