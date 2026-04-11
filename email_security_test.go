package main

import (
	"os"
	"strings"
	"testing"
)

// NOTE: sanitizeContent tests removed - this function does not exist and is not needed
// Email content should NOT be sanitized before being sent to LLM - the LLM needs full context

// TestEncodeHeader tests header injection prevention
func TestEncodeHeader(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "normal header",
			input:    "Test Subject",
			expected: "Test Subject",
		},
		{
			name:     "newline injection",
			input:    "Test\r\nBcc: attacker@evil.com",
			expected: "Test  Bcc: attacker@evil.com", // Both \r and \n replaced with spaces
		},
		{
			name:     "carriage return injection",
			input:    "Test\rHeader continuation",
			expected: "Test Header continuation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := encodeHeader(tt.input)
			if result != tt.expected {
				t.Errorf("encodeHeader(%q) = %q, want %q", tt.input, result, tt.expected)
			}

			// Ensure no newlines remain
			if strings.Contains(result, "\n") || strings.Contains(result, "\r") {
				t.Errorf("encodeHeader still contains newlines: %q", result)
			}
		})
	}
}

// TestValidateEmailFormat tests email format validation
func TestValidateEmailFormat(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "valid email",
			input:    "user@stg.net",
			expected: true,
		},
		{
			name:     "valid email with subdomain",
			input:    "user@mail.example.com",
			expected: true,
		},
		{
			name:     "invalid email (no domain)",
			input:    "notanemail",
			expected: false,
		},
		{
			name:     "valid email at test.com",
			input:    "user@test.com",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validateEmailFormat(tt.input)
			if result != tt.expected {
				t.Errorf("validateEmailFormat(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

// TestEncodeHeaderTruncation tests that very long headers are truncated
func TestEncodeHeaderTruncation(t *testing.T) {
	longHeader := strings.Repeat("A", 1000)
	result := encodeHeader(longHeader)

	if len(result) > 503 || !strings.HasSuffix(result, "..") {
		t.Errorf("Header not properly truncated: length=%d, ends_with='..': %v", len(result), strings.HasSuffix(result, ".."))
	}
}

// TestLoadAttachments tests loading email attachments from file paths
func TestLoadAttachments(t *testing.T) {
	tmpDir := t.TempDir()
	testFile1 := tmpDir + "/test1.txt"
	testFile2 := tmpDir + "/test2.txt"

	if err := os.WriteFile(testFile1, []byte("content 1"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(testFile2, []byte("content 2"), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name      string
		paths     []string
		wantCount int
		wantError bool
	}{
		{"single attachment", []string{testFile1}, 1, false},
		{"multiple attachments", []string{testFile1, testFile2}, 2, false},
		{"empty paths slice", []string{}, 0, false},
		{"non-existent file", []string{"/nonexistent/file.txt"}, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attachments, err := loadAttachments(tt.paths)

			if tt.wantError && err == nil {
				t.Errorf("loadAttachments expected error but got none")
			}

			if !tt.wantError && err != nil {
				t.Errorf("loadAttachments unexpected error: %v", err)
			}

			if !tt.wantError && len(attachments) != tt.wantCount {
				t.Errorf("loadAttachments got %d attachments, want %d", len(attachments), tt.wantCount)
			}
		})
	}
}

// TestSanitizeFilename tests filename sanitization for email headers
func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"normal filename", "test.txt", "test.txt"},
		{"with quotes", `test"file.txt`, "testfile.txt"},
		{"with newline", "test\nfile.txt", "test_file.txt"},
		{"with carriage return", "test\rfile.txt", "test_file.txt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeFilename(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeFilename(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestParseAttachmentArg tests parsing attachment paths from tool arguments
func TestParseAttachmentArg(t *testing.T) {
	tests := []struct {
		name     string
		args     map[string]any
		expected []string
	}{
		{"no attachments", map[string]any{}, nil},
		{"empty attachments array", map[string]any{"attachments": []string{}}, nil},
		{"single attachment", map[string]any{"attachments": []string{"/path/file.txt"}}, []string{"/path/file.txt"}},
		{"multiple attachments", map[string]any{"attachments": []string{"/path/file1.txt", "/path/file2.txt"}}, []string{"/path/file1.txt", "/path/file2.txt"}},
		{"mixed empty strings", map[string]any{"attachments": []string{"/path/file.txt", "", "/path/file2.txt"}}, []string{"/path/file.txt", "/path/file2.txt"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseAttachmentArg(tt.args)

			if len(result) != len(tt.expected) {
				t.Errorf("parseAttachmentArg got %d paths, want %d: got %v", len(result), len(tt.expected), result)
			}

			for i, expectedPath := range tt.expected {
				if i < len(result) && result[i] != expectedPath {
					t.Errorf("parseAttachmentArg[%d] = %q, want %q", i, result[i], expectedPath)
				}
			}
		})
	}
}
