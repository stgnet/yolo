package main

import (
	"strings"
	"testing"
)

func TestPlaywrightMCPExecutor(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		shouldError bool
	}{
		{
			name:        "empty selector should error",
			input:       "",
			shouldError: true,
		},
		{
			name:        "valid CSS selector",
			input:       "#main-content",
			shouldError: false,
		},
		{
			name:        "selector with class",
			input:       ".button.submit",
			shouldError: false,
		},
		{
			name:        "selector too long",
			input:       string(make([]byte, 600)),
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSelector(tt.input)
			if tt.shouldError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.shouldError && err != nil {
				t.Errorf("expected no error but got: %v", err)
			}
		})
	}
}

func TestPlaywrightMCPExecutor_ValidateURL(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		shouldError bool
	}{
		{
			name:        "empty URL should error",
			input:       "",
			shouldError: true,
		},
		{
			name:        "valid HTTPS URL",
			input:       "https://example.com",
			shouldError: false,
		},
		{
			name:        "valid HTTP URL",
			input:       "http://example.com/path?query=1",
			shouldError: false,
		},
		{
			name:        "file protocol should error",
			input:       "file:///etc/passwd",
			shouldError: true,
		},
		{
			name:        "javascript protocol should error",
			input:       "javascript:alert(1)",
			shouldError: true,
		},
		{
			name:        "URL too long",
			input:       string(make([]byte, 3000)),
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := validateURL(tt.input)
			if tt.shouldError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.shouldError && err != nil {
				t.Errorf("expected no error but got: %v", err)
			}
		})
	}
}

func TestEscapeJS(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple string",
			input:    "hello world",
			expected: "hello world",
		},
		{
			name:     "backslash escape",
			input:    "path\\to\\file",
			expected: "path\\\\to\\\\file",
		},
		{
			name:     "quote escapes",
			input:    "It's \"quoted\"",
			expected: "It\\'s \\\"quoted\\\"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := escapeJS(tt.input)
			if result != tt.expected {
				t.Errorf("expected %q but got %q", tt.expected, result)
			}
		})
	}
}

func TestPlaywrightMCPExecutor_NavigateAction(t *testing.T) {
	executor := NewToolExecutor(".", nil)

	result := executor.playwrightMCP(map[string]any{
		"action": "navigate",
		"url":    "https://example.com",
	})

	// Should return an error because Playwright/Node.js is not available in test env
	if result == "" {
		t.Error("expected non-empty result")
	}
}

func TestPlaywrightMCPExecutor_InvalidAction(t *testing.T) {
	executor := NewToolExecutor(".", nil)

	result := executor.playwrightMCP(map[string]any{
		"action": "invalid_action",
	})

	if !containsString(result, "unknown action") || !containsString(result, "Available actions") {
		t.Errorf("expected error about unknown action with available actions list, got: %s", result)
	}
}

func TestPlaywrightMCPExecutor_MissingAction(t *testing.T) {
	executor := NewToolExecutor(".", nil)

	result := executor.playwrightMCP(map[string]any{})

	if !containsString(result, "action is required") || !containsString(result, "Available actions") {
		t.Errorf("expected error about missing action with available actions list, got: %s", result)
	}
}

// Helper function for testing
func containsString(s, substr string) bool {
	return strings.Contains(s, substr)
}
