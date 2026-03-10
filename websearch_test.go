package main

import (
	"testing"
)

func TestWebSearchTool(t *testing.T) {
	tests := []struct {
		name         string
		args         map[string]any
		expectError  bool
		expectOutput string
	}{
		{
			name: "missing query",
			args: map[string]any{},
		},
		{
			name: "basic query",
			args: map[string]any{
				"query": "go programming language",
			},
			expectError: false,
		},
		{
			name: "query with count",
			args: map[string]any{
				"query": "golang concurrency patterns",
				"count": 3,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewToolExecutor("/tmp/test-websearch", nil)
			
			if executor == nil {
				t.Fatal("executor is nil")
			}
			
			result := executor.webSearch(tt.args)
			t.Logf("Result: %s", result[:min(200, len(result))])
		})
	}
}

func TestWebSearchWikipedia(t *testing.T) {
	result := searchWikipedia("Go programming language", 3)
	if result == "" {
		t.Log("Wikipedia search returned empty")
	} else {
		t.Logf("Found: %s", result[:min(100, len(result))])
	}
}

func TestWebSearchBing(t *testing.T) {
	result := searchBing("golang testing patterns", 5)
	if result == "" {
		t.Log("Bing search returned empty")
	} else {
		t.Logf("Found: %s", result[:min(100, len(result))])
	}
}

func TestCleanHTML(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "Hello &amp; World",
			expected: "Hello & World",
		},
		{
			input:    "<div>Content</div>",
			expected: "Content",
		},
		{
			input:    "&lt;tag&gt;",
			expected: "<tag>",
		},
	}

	for _, tt := range tests {
		result := cleanHTML(tt.input)
		if result != tt.expected {
			t.Errorf("cleanHTML(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}
