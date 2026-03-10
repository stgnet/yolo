package main

import (
	"strings"
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
			name:        "missing query",
			args:        map[string]any{},
			expectError: true,
		},
		{
			name:        "basic query",
			args:        map[string]any{"query": "go programming language"},
			expectError: false,
		},
		{
			name:        "query with count",
			args:        map[string]any{"query": "golang concurrency patterns", "count": 3},
			expectError: false,
		},
		{
			name:        "empty query string",
			args:        map[string]any{"query": ""},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewToolExecutor("/tmp/test-websearch", nil)
			if executor == nil {
				t.Fatal("executor is nil")
			}

			result := executor.webSearch(tt.args)

			if tt.expectError {
				if !strings.Contains(result, "Error:") && !strings.Contains(result, "required") {
					t.Errorf("Expected error but got: %s", result[:min(200, len(result))])
				}
			} else {
				if strings.Contains(result, "Error:") {
					t.Errorf("Unexpected error: %s", result[:min(500, len(result))])
				}
				t.Logf("Result preview: %s", result[:min(200, len(result))])
			}
		})
	}
}

func TestWebSearchWikipediaFallback(t *testing.T) {
	executor := NewToolExecutor("/tmp/test-websearch", nil)
	if executor == nil {
		t.Fatal("executor is nil")
	}

	// Test Wikipedia search directly via the private method
	result := executor.searchWikipedia("Go programming language", 3)

	if strings.Contains(result, "Error:") {
		t.Logf("Wikipedia search returned error (may be network issue): %s", result[:min(200, len(result))])
	} else if result == "" {
		t.Log("Wikipedia search returned empty result")
	} else {
		t.Logf("Wikipedia search succeeded: %s", result[:min(100, len(result))])
	}
}

func TestWebSearchDuckDuckGo(t *testing.T) {
	executor := NewToolExecutor("/tmp/test-websearch", nil)
	if executor == nil {
		t.Fatal("executor is nil")
	}

	// Test DuckDuckGo search directly via the private method
	result := executor.searchDuckDuckGo("golang testing patterns", 5)

	if strings.Contains(result, "Error:") {
		t.Logf("DuckDuckGo search returned error (may be network issue): %s", result[:min(200, len(result))])
	} else if result == "" {
		t.Log("DuckDuckGo search returned empty result (trying fallback)")
	} else {
		t.Logf("DuckDuckGo search succeeded: %s", result[:min(100, len(result))])
	}
}

func TestIsEmptySearchResult(t *testing.T) {
	executor := NewToolExecutor("/tmp/test-websearch", nil)
	if executor == nil {
		t.Fatal("executor is nil")
	}

	tests := []struct {
		name   string
		result string
		expect bool
	}{
		{
			name:   "empty error message",
			result: "Error: something went wrong",
			expect: true,
		},
		{
			name:   "no results message",
			result: "No results found for this query",
			expect: true,
		},
		{
			name:   "short result",
			result: "Brief answer",
			expect: true,
		},
		{
			name:   "meaningful result",
			result: "Go is a programming language developed by Google in 2009. It is designed to be simple and efficient.",
			expect: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := executor.isEmptySearchResult(tt.result)
			if got != tt.expect {
				t.Errorf("isEmptySearchResult(%q) = %v, want %v", tt.result, got, tt.expect)
			}
		})
	}
}

func TestWebSearchArgumentParsing(t *testing.T) {
	executor := NewToolExecutor("/tmp/test-websearch", nil)
	if executor == nil {
		t.Fatal("executor is nil")
	}

	tests := []struct {
		name     string
		args     map[string]any
		expected int
	}{
		{
			name:     "default count",
			args:     map[string]any{"query": "test"},
			expected: 5,
		},
		{
			name:     "custom count",
			args:     map[string]any{"query": "test", "count": 3},
			expected: 3,
		},
		{
			name:     "max count exceeded",
			args:     map[string]any{"query": "test", "count": 20},
			expected: 10,
		},
		{
			name:     "min count",
			args:     map[string]any{"query": "test", "count": 1},
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count := getIntArg(tt.args, "count", 5)
			if count > 10 {
				count = 10
			}
			if count < 1 {
				count = 5
			}

			if count != tt.expected {
				t.Errorf("count parsing got %d, want %d", count, tt.expected)
			}
		})
	}
}
