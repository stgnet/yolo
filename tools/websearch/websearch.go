package websearch

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// SearchResult represents a search result from the web
type SearchResult struct {
	URL    string `json:"url"`
	Title  string `json:"title"`
	Snippet string `json:"snippet"`
}

// WebSearcher handles web search operations
type WebSearcher struct {
	client *http.Client
}

// NewWebSearcher creates a new WebSearcher instance
func NewWebSearcher() *WebSearcher {
	return &WebSearcher{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Search performs a web search and returns results
func (ws *WebSearcher) Search(ctx context.Context, query string) ([]*SearchResult, error) {
	if !ws.ValidateQuery(query) {
		return nil, fmt.Errorf("invalid query")
	}

	searchURL := fmt.Sprintf("https://www.duckduckgo.com/?q=%s", url.QueryEscape(query))
	
	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := ws.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// For now, return mock results - in production this would parse the response
	results := []*SearchResult{
		{
			URL:     fmt.Sprintf("https://example.com/search?q=%s", url.QueryEscape(query)),
			Title:   fmt.Sprintf("Results for: %s", query),
			Snippet: fmt.Sprintf("Mock search results for query: %s", query),
		},
	}

	return results, nil
}

// ValidateQuery validates if a search query is valid
func (ws *WebSearcher) ValidateQuery(query string) bool {
	if query == "" {
		return false
	}

	// Check for minimum length
	if len(query) < 2 {
		return false
	}

	// Check for invalid characters
	if regexp.MustCompile(`[<>{}|\\^`\[\]]`).MatchString(query) {
		return false
	}

	return true
}

// SearchWithCount performs a web search with a specific number of results
func (ws *WebSearcher) SearchWithCount(ctx context.Context, query string, count int) ([]*SearchResult, error) {
	if !ws.ValidateQuery(query) {
		return nil, fmt.Errorf("invalid query")
	}

	if count < 1 {
		count = 10
	} else if count > 50 {
		count = 50
	}

	results, err := ws.Search(ctx, query)
	if err != nil {
		return nil, err
	}

	// Cap results at requested count
	if len(results) > count {
		results = results[:count]
	}

	return results, nil
}

// Search performs a web search (package-level function for convenience)
func Search(ctx context.Context, query string) ([]*SearchResult, error) {
	searcher := NewWebSearcher()
	return searcher.Search(ctx, query)
}

// ValidateQuery validates if a search query is valid (package-level function)
func ValidateQuery(query string) bool {
	searcher := NewWebSearcher()
	return searcher.ValidateQuery(query)
}

// SearchWithCount performs a web search with a specific number of results (package-level function)
func SearchWithCount(ctx context.Context, query string, count int) ([]*SearchResult, error) {
	searcher := NewWebSearcher()
	return searcher.SearchWithCount(ctx, query, count)
}

// EscapeQuery escapes special characters in a search query
func EscapeQuery(query string) string {
	// Convert spaces to + for URL encoding compatibility
	query = strings.ReplaceAll(query, " ", "+")
	// Escape other special characters
	specialChars := map[string]string{
		"&":  "%26",
		"=":  "%3D",
		"+":  "%2B",
		"@":  "%40",
		":":  "%3A",
		"/":  "%2F",
		"?":  "%3F",
		"#":  "%23",
		"[":  "%5B",
		"]":  "%5D",
		" ":  "%20",
	}

	result := query
	for char, encoded := range specialChars {
		result = strings.ReplaceAll(result, char, encoded)
	}

	return result
}

// GetQueryWordCount counts the number of words in a query
func GetQueryWordCount(query string) int {
	if query == "" {
		return 0
	}
	return len(strings.Fields(query))
}
