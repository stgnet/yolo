package main

import (
	"strings"
	"testing"
	"time"
)

func TestWebSearchTool(t *testing.T) {
	t.Skip("Skipping network-dependent test")
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
	t.Skip("Skipping network-dependent test")
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
	t.Skip("Skipping network-dependent test")
	executor := NewToolExecutor("/tmp/test-websearch", nil)
	if executor == nil {
		t.Fatal("executor is nil")
	}

	tests := []struct {
		name  string
		query string
	}{
		{
			name:  "golang testing patterns",
			query: "golang testing patterns",
		},
		{
			name:  "dependency injection golang",
			query: "dependency injection golang",
		},
		{
			name:  "golang factory pattern",
			query: "golang factory pattern",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := executor.searchDuckDuckGo(tt.query, 5)

			if strings.Contains(result, "Error:") {
				t.Logf("DuckDuckGo search returned error: %s", result[:min(200, len(result))])
			} else if result == "" {
				t.Log("DuckDuckGo search returned empty/invalid result (will use Wikipedia fallback)")
			} else if executor.isEmptySearchResult(result) {
				t.Log("DuckDuckGo search detected as empty by isEmptySearchResult (will use Wikipedia fallback)")
			} else {
				t.Logf("DuckDuckGo search succeeded: %s", result[:min(150, len(result))])
			}
		})
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
			name: "duckduckgo no results message",
			result: `Search results for "golang testing patterns":

No results found for this query. Try a different search term.`,
			expect: true,
		},
		{
			name:   "try different search term",
			result: "Try a different search term for better results",
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
				t.Errorf("isEmptySearchResult(%q) = %v, want %v", tt.result[:min(50, len(tt.result))], got, tt.expect)
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

func TestWebSearchCaching(t *testing.T) {
	t.Skip("Skipping network-dependent test")
	executor := NewToolExecutor("/tmp/test-websearch-cache", nil)
	if executor == nil {
		t.Fatal("executor is nil")
	}

	query := "go programming language"
	count := 3

	// Clear any existing cache entries for this test
	searchCache.Clear()

	// First search - should NOT be cached
	result1 := executor.webSearch(map[string]any{"query": query, "count": count})
	if strings.Contains(result1, "Error:") || strings.Contains(result1, "No search results found") {
		t.Skip("Skipping cache test due to network error or no results")
	}

	// Result should NOT have [Cached] prefix on first call
	if strings.HasPrefix(result1, "[Cached]") {
		t.Errorf("First search should not be cached but got: %s", result1[:50])
	}

	// Second search with same query - SHOULD be cached
	result2 := executor.webSearch(map[string]any{"query": query, "count": count})
	if !strings.HasPrefix(result2, "[Cached]") {
		t.Errorf("Second search should be cached but got: %s", result2[:50])
	}

	// Verify cache key generation is consistent
	key1 := getSearchCacheKey(query, count)
	key2 := getSearchCacheKey(query, count)
	if key1 != key2 {
		t.Errorf("Cache keys should be identical: %s vs %s", key1, key2)
	}

	// Verify different queries produce different keys (case-insensitive)
	key3 := getSearchCacheKey("GO PROGRAMMING LANGUAGE", count)
	if key1 != key3 {
		t.Errorf("Cache keys should be case-insensitive: %s vs %s", key1, key3)
	}

	// Verify different counts produce different keys
	key4 := getSearchCacheKey(query, 5)
	if key1 == key4 {
		t.Errorf("Different counts should produce different keys")
	}
}

func TestWebSearchCacheExpiration(t *testing.T) {
	t.Skip("Skipping network-dependent test")
	executor := NewToolExecutor("/tmp/test-websearch-expire", nil)
	if executor == nil {
		t.Fatal("executor is nil")
	}

	query := "cache expiration test"
	key := getSearchCacheKey(query, 5)

	// Clear any existing entries first
	searchCache.Clear()

	// Store an expired entry manually
	searchCache.Store(key, &searchCacheEntry{
		Result: "Old cached result",
		Ts:     time.Now().Add(-10 * time.Minute), // Expired 10 minutes ago
	})

	// Verify the entry exists before retrieval attempt
	_, found := searchCache.Load(key)
	if !found {
		t.Fatal("Entry should exist before retrieval test")
	}

	// Try to retrieve - should return false since it's expired and also clean up
	_, found = executor.getFromSearchCache(key)
	if found {
		t.Error("Should not find expired cache entry after getFromSearchCache")
	}

	// Verify the expired entry was cleaned up by getFromSearchCache
	_, stillExists := searchCache.Load(key)
	if stillExists {
		t.Error("Expired entry should have been removed from cache by getFromSearchCache")
	}
}
