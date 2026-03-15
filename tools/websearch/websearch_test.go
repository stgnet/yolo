package websearch

import (
	"context"
	"strings"
	"testing"
)

func TestSearch(t *testing.T) {
	w := NewWebSearcher("DuckDuckGo")
	results, err := w.Search(context.Background(), "golang programming")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) == 0 {
		t.Error("Expected results, got none")
	}
	for _, url := range results {
		if !strings.HasPrefix(url, "http") {
			t.Errorf("Invalid URL format: %s", url)
		}
	}
}

func TestSearchWithCount(t *testing.T) {
	w := NewWebSearcher("DuckDuckGo")
	results, err := w.SearchWithCount(context.Background(), "go language", 3)
	if err != nil {
		t.Fatalf("SearchWithCount failed: %v", err)
	}
	if len(results) > 3 {
		t.Errorf("Expected max 3 results, got %d", len(results))
	}
}

func TestValidateQuery(t *testing.T) {
	tests := []struct {
		query    string
		expected bool
	}{
		{"golang", true},
		{"go programming", true},
		{"", false},
		{"   ", false},
	}

	for _, tt := range tests {
		result := ValidateQuery(tt.query)
		if result != tt.expected {
			t.Errorf("ValidateQuery(%q) = %v, expected %v", tt.query, result, tt.expected)
		}
	}
}

func TestWebSearcher_New(t *testing.T) {
	w := NewWebSearcher("DuckDuckGo")
	if w.SearchEngine != "DuckDuckGo" {
		t.Errorf("Expected search engine DuckDuckGo, got %s", w.SearchEngine)
	}
	if w.UserAgent == "" {
		t.Error("UserAgent should not be empty")
	}
}
