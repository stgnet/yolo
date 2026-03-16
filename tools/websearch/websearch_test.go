package websearch

import (
	"context"
	"testing"
)

func TestSearch(t *testing.T) {
	w := NewWebSearcher()
	results, err := w.Search(context.Background(), "golang programming", 5)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results.InstantAnswer) == 0 && len(results.RelatedTopics) == 0 {
		t.Log("No results found (might be due to DuckDuckGo API changes)")
	}
}

func TestSearchWithCount(t *testing.T) {
	w := NewWebSearcher()
	results, err := w.Search(context.Background(), "go language", 3)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	totalResults := len(results.InstantAnswer) + len(results.RelatedTopics)
	if totalResults > 3 && results.Error == "" {
		t.Errorf("Expected max 3 results, got %d", totalResults)
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
		result := isValidQuery(tt.query)
		if result != tt.expected {
			t.Errorf("isValidQuery(%q) = %v, expected %v", tt.query, result, tt.expected)
		}
	}
}

func TestWebSearcher_New(t *testing.T) {
	w := NewWebSearcher()
	if w.UserAgent == "" {
		t.Error("UserAgent should not be empty")
	}
	if w.BaseTimeout == 0 {
		t.Error("BaseTimeout should be set")
	}
}
