package yolo

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// TestWrapWithDetails tests the error wrapping function
func TestWrapWithDetails(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		message  string
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			message:  "test message",
			expected: false,
		},
		{
			name:     "wrapped error preserves original",
			err:      fmt.Errorf("original error"),
			message:  "additional context",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := WrapWithDetails(tt.err, tt.message)
			if result == nil && tt.expected {
				t.Errorf("Expected non-nil error, got nil")
			}
			if result != nil && !tt.expected {
				t.Errorf("Expected nil error, got: %v", result)
			}
		})
	}
}

// TestFormatErrorForDisplay tests error formatting
func TestFormatErrorForDisplay(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "error with message",
			err:      fmt.Errorf("test error"),
			expected: "error: test error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatErrorForDisplay(tt.err)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// TestParseDuration tests duration parsing
func TestParseDuration(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int64
	}{
		{
			name:     "1 second",
			input:    "1s",
			expected: 1000,
		},
		{
			name:     "30 seconds",
			input:    "30s",
			expected: 30000,
		},
		{
			name:     "5 minutes",
			input:    "5m",
			expected: 300000,
		},
		{
			name:     "1 hour",
			input:    "1h",
			expected: 3600000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseDuration(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %d ms, got %d ms", tt.expected, result)
			}
		})
	}
}

// TestParseTime tests time parsing
func TestParseTime(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "valid timestamp",
			input:    "2024-01-01T00:00:00Z",
			expected: "2024-01-01 00:00:00 +0000 UTC",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseTime(tt.input)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// TestReadLines tests reading lines from file
func TestReadLines(t *testing.T) {
	// Create temporary file
	tmpFile := filepath.Join(os.TempDir(), "test_read_lines.txt")
	defer os.Remove(tmpFile)

	content := "line1\nline2\nline3"
	err := os.WriteFile(tmpFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	lines, err := ReadLines(tmpFile)
	if err != nil {
		t.Fatalf("ReadLines failed: %v", err)
	}

	if len(lines) != 3 {
		t.Errorf("Expected 3 lines, got %d", len(lines))
	}

	if lines[0] != "line1" || lines[1] != "line2" || lines[2] != "line3" {
		t.Errorf("Lines don't match: %v", lines)
	}
}

// TestWriteLines tests writing lines to file
func TestWriteLines(t *testing.T) {
	tmpFile := filepath.Join(os.TempDir(), "test_write_lines.txt")
	defer os.Remove(tmpFile)

	lines := []string{"line1", "line2", "line3"}
	err := WriteLines(tmpFile, lines)
	if err != nil {
		t.Fatalf("WriteLines failed: %v", err)
	}

	content, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("Failed to read temp file: %v", err)
	}

	expected := "line1\nline2\nline3"
	if string(content) != expected {
		t.Errorf("Expected '%s', got '%s'", expected, string(content))
	}
}

// TestAppendToFile tests appending to file
func TestAppendToFile(t *testing.T) {
	tmpFile := filepath.Join(os.TempDir(), "test_append.txt")
	defer os.Remove(tmpFile)

	// First write
	err := AppendToFile(tmpFile, "line1\n")
	if err != nil {
		t.Fatalf("First AppendToFile failed: %v", err)
	}

	// Second append
	err = AppendToFile(tmpFile, "line2\n")
	if err != nil {
		t.Fatalf("Second AppendToFile failed: %v", err)
	}

	content, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("Failed to read temp file: %v", err)
	}

	expected := "line1\nline2\n"
	if string(content) != expected {
		t.Errorf("Expected '%s', got '%s'", expected, string(content))
	}
}

// TestTruncate tests truncating files
func TestTruncate(t *testing.T) {
	tmpFile := filepath.Join(os.TempDir(), "test_truncate.txt")
	defer os.Remove(tmpFile)

	// Create file with content
	err := os.WriteFile(tmpFile, []byte("line1\nline2\nline3\n"), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	// Truncate to 2 lines
	err = Truncate(tmpFile, 2)
	if err != nil {
		t.Fatalf("Truncate failed: %v", err)
	}

	content, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("Failed to read temp file: %v", err)
	}

	expected := "line1\nline2\n"
	if string(content) != expected {
		t.Errorf("Expected '%s', got '%s'", expected, string(content))
	}
}

// TestSanitizePath tests path sanitization
func TestSanitizePath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "path with ..",
			input:    "/safe/../unsafe/path",
			expected: "/unsafe/path",
		},
		{
			name:     "already safe path",
			input:    "/safe/path",
			expected: "/safe/path",
		},
		{
			name:     "empty path",
			input:    "",
			expected: ".",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizePath(tt.input)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// TestCheckFileAccess tests file access checking
func TestCheckFileAccess(t *testing.T) {
	tmpFile := filepath.Join(os.TempDir(), "test_access.txt")
	defer os.Remove(tmpFile)

	// Create file with 0644 permissions
	err := os.WriteFile(tmpFile, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	// Should be readable
	readable, _ := CheckFileAccess(tmpFile, "r")
	if !readable {
		t.Errorf("Expected file to be readable")
	}

	// Should not be executable
	executable, _ := CheckFileAccess(tmpFile, "x")
	if executable {
		t.Errorf("Expected file to not be executable")
	}
}

// TestIsDirectoryWritable tests directory write permission check
func TestIsDirectoryWritable(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "test_writable")
	defer os.RemoveAll(tmpDir)

	err := os.MkdirAll(tmpDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	writable := IsDirectoryWritable(tmpDir)
	if !writable {
		t.Errorf("Expected directory to be writable")
	}
}

// TestToJSON tests JSON serialization
func TestToJSON(t *testing.T) {
	testMap := map[string]interface{}{
		"name": "test",
		"value": 123,
		"active": true,
	}

	result, err := ToJSON(testMap)
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	if result == "" {
		t.Errorf("Expected non-empty JSON string")
	}
}

// TestFromJSON tests JSON deserialization
func TestFromJSON(t *testing.T) {
	jsonStr := `{"name":"test","value":123}`

	var result map[string]interface{}
	err := FromJSON(jsonStr, &result)
	if err != nil {
		t.Fatalf("FromJSON failed: %v", err)
	}

	if result["name"] != "test" || result["value"].(float64) != 123 {
		t.Errorf("JSON deserialization didn't work correctly")
	}
}

// TestGetConfigDir tests config directory retrieval
func TestGetConfigDir(t *testing.T) {
	result := GetConfigDir()
	if result == "" {
		t.Errorf("Expected non-empty config directory path")
	}
}

// TestGetCacheDir tests cache directory retrieval
func TestGetCacheDir(t *testing.T) {
	result := GetCacheDir()
	if result == "" {
		t.Errorf("Expected non-empty cache directory path")
	}
}

// TestReadFirstLine tests reading first line from file
func TestReadFirstLine(t *testing.T) {
	tmpFile := filepath.Join(os.TempDir(), "test_first_line.txt")
	defer os.Remove(tmpFile)

	content := "first line\nsecond line\nthird line"
	err := os.WriteFile(tmpFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	firstLine, err := ReadFirstLine(tmpFile)
	if err != nil {
		t.Fatalf("ReadFirstLine failed: %v", err)
	}

	if firstLine != "first line" {
		t.Errorf("Expected 'first line', got '%s'", firstLine)
	}
}

// TestGetEnvOrDefault tests environment variable defaults
func TestGetEnvOrDefault(t *testing.T) {
	result := GetEnvOrDefault("NONEXISTENT_VAR", "default_value")
	if result != "default_value" {
		t.Errorf("Expected 'default_value', got '%s'", result)
	}

	os.Setenv("TEST_VAR", "test_value")
	result = GetEnvOrDefault("TEST_VAR", "default_value")
	if result != "test_value" {
		t.Errorf("Expected 'test_value', got '%s'", result)
	}
}
