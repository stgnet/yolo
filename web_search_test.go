// Package main implements web search improvements for YOLO
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
)

// SearchResult represents a web search result
type SearchResult struct {
	Query   string   `json:"query"`
	Results []string `json:"results"`
	Sources []string `json:"sources"`
}

// MockHTTPClient implements http.Client for testing
type MockHTTPClient struct {
	Response *http.Response
	Body     io.Reader
	Error    error
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	body := io.NopCloser(m.Body)
	resp := &http.Response{
		StatusCode: 200,
		Body:       body,
		Header:     make(http.Header),
	}
	return resp, nil
}

// DuckDuckGoSearch searches using DuckDuckGo API (limited to 5 results)
func DuckDuckGoSearch(query string, limit int) ([]string, error) {
	if limit > 5 {
		limit = 5 // DuckDuckGo max is 5
	}

	url := fmt.Sprintf("https://api.duckduckgo.com/?q=%s&format=json&limit=%d",
		http.URLEscape(query), limit)

	req, err := http.NewRequestWithContext(context.Background(), "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Extract Abstract from DuckDuckGo response
	if abstract, ok := result["Abstract"].(string); ok && abstract != "" {
		return []string{abstract}, nil
	}

	// Fallback for empty results
	if abstract == "" {
		return nil, fmt.Errorf("empty results from DuckDuckGo")
	}

	return []string{abstract}, nil
}

// JinaAISearch uses Jina AI as a more reliable search alternative
func JinaAISearch(query string) ([]string, error) {
	url := "https://r.jina.ai/" + http.URLEscape("https://duckduckgo.com/?q=" + query)

	req, err := http.NewRequestWithContext(context.Background(), "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make Jina AI request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read Jina AI response: %w", err)
	}

	if len(body) < 50 {
		return nil, fmt.Errorf("Jina AI returned empty content")
	}

	return []string{string(body)}, nil
}

// WikipediaSearch searches Wikipedia for the query
func WikipediaSearch(query string) ([]string, error) {
	url := fmt.Sprintf("https://en.wikipedia.org/api/rest_v1/page/summary/%s",
		http.URLEscape(query))

	req, err := http.NewRequestWithContext(context.Background(), "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Wikipedia request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make Wikipedia request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("Wikipedia article not found")
	}

	var wikiResult map[string]interface{}
	body, _ := io.ReadAll(resp.Body)
	json.Unmarshal(body, &wikiResult)

	if extract, ok := wikiResult["extract"].(string); ok && len(extract) > 0 {
		return []string{fmt.Sprintf("Wikipedia: %s", extract)}, nil
	}

	return nil, fmt.Errorf("no Wikipedia content found")
}

// SmartSearch tries multiple search sources with fallback strategy
func SmartSearch(query string) (*SearchResult, error) {
	results := &SearchResult{Query: query}

	// Strategy 1: Try DuckDuckGo first (fast, local)
	if ddResults, err := DuckDuckGoSearch(query, 5); err == nil && len(ddResults) > 0 {
		results.Results = append(results.Results, ddResults...)
		results.Sources = append(results.Sources, "DuckDuckGo")
	} else {
		fmt.Printf("DuckDuckGo failed for '%s': %v\n", query, err)
	}

	// Strategy 2: Try Jina AI fallback (more reliable)
	if len(results.Results) == 0 {
		if jinaResults, err := JinaAISearch(query); err == nil && len(jinaResults) > 0 {
			results.Results = append(results.Results, jinaResults...)
			results.Sources = append(results.Sources, "Jina AI")
		} else {
			fmt.Printf("Jina AI failed for '%s': %v\n", query, err)
		}
	}

	// Strategy 3: Try Wikipedia as third option
	if len(results.Results) == 0 {
		if wikiResults, err := WikipediaSearch(query); err == nil && len(wikiResults) > 0 {
			results.Results = append(results.Results, wikiResults...)
			results.Sources = append(results.Sources, "Wikipedia")
		}
	}

	// If all searches failed, return a meaningful error
	if len(results.Results) == 0 {
		return results, fmt.Errorf("all search sources returned empty results for: %s", query)
	}

	return results, nil
}

func TestDuckDuckGoSearch(t *testing.T) {
	tests := []struct {
		name    string
		query   string
		limit   int
		wantLen int
		wantErr bool
	}{
		{"valid query with DuckDuckGo", "Golang programming best practices", 5, 1, false},
		{"invalid limit (over 5)", "test", 10, 1, false}, // Should be capped at 5
		{"empty query", "", 5, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DuckDuckGoSearch(tt.query, tt.limit)
			if (err != nil) != tt.wantErr {
				t.Errorf("DuckDuckGoSearch() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(got) != tt.wantLen && !tt.wantErr {
				t.Errorf("DuckDuckGoSearch() got %d results, want %d", len(got), tt.wantLen)
			}
		})
	}
}

func TestJinaAISearch(t *testing.T) {
	tests := []struct {
		name    string
		query   string
		wantLen int
		wantErr bool
	}{
		{"valid query", "Golang programming best practices", 1, false},
		{"short query", "Go", 1, false},
		{"empty query", "", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := JinaAISearch(tt.query)
			if (err != nil) != tt.wantErr {
				t.Errorf("JinaAISearch() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(got) != tt.wantLen && !tt.wantErr {
				t.Errorf("JinaAISearch() got %d results, want %d", len(got), tt.wantLen)
			}
		})
	}
}

func TestWikipediaSearch(t *testing.T) {
	tests := []struct {
		name    string
		query   string
		wantLen int
		wantErr bool
	}{
		{"existing article", "Golang", 1, false},
		{"non-existent article", "zxcvbnm123456789", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := WikipediaSearch(tt.query)
			if (err != nil) != tt.wantErr {
				t.Errorf("WikipediaSearch() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(got) != tt.wantLen && !tt.wantErr {
				t.Errorf("WikipediaSearch() got %d results, want %d", len(got), tt.wantLen)
			}
		})
	}
}

func TestSmartSearch(t *testing.T) {
	tests := []struct {
		name      string
		query     string
		wantSrc   int // number of sources tried
		wantErr   bool
	}{
		{"successful search", "Golang programming best practices", 1, false},
		{"empty query", "", 0, true},
		{"short keyword", "Go", 1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SmartSearch(tt.query)
			if (err != nil) != tt.wantErr {
				t.Errorf("SmartSearch() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got == nil && !tt.wantErr {
				t.Errorf("SmartSearch() returned nil result")
			}
		})
	}
}

func TestEmptyResultsHandling(t *testing.T) {
	// Test that the system handles empty results gracefully
	result, err := SmartSearch("")
	if err == nil {
		t.Error("Expected error for empty query but got none")
	}
	if result != nil && len(result.Results) > 0 {
		t.Errorf("Expected no results but got %d", len(result.Results))
	}
}

func TestMultipleSearchSources(t *testing.T) {
	// Verify that multiple search sources are used in fallback order
	result, err := SmartSearch("Golang programming")
	if err != nil {
		t.Fatalf("SmartSearch failed: %v", err)
	}

	if len(result.Sources) == 0 {
		t.Error("Expected at least one search source to be used")
	}

	// Should start with DuckDuckGo, then fallback to Jina AI or Wikipedia
	fmt.Printf("Query: %s\nSources used: %v\nResults count: %d\n",
		result.Query, result.Sources, len(result.Results))
}

func BenchmarkDuckDuckGoSearch(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = DuckDuckGoSearch("Golang", 5)
	}
}

func BenchmarkJinaAISearch(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = JinaAISearch("Golang")
	}
}

func BenchmarkWikipediaSearch(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = WikipediaSearch("Golang")
	}
}

func BenchmarkSmartSearch(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = SmartSearch("Golang programming best practices")
	}
}

func main() {
	// Run tests if called directly with test flag
	if len(os.Args) > 1 && os.Args[1] == "-test" {
		testing.Main(
			func(pat, str string) (bool, error) { return true, nil },
			[]testing.InternalTest{
				{Name: "TestDuckDuckGoSearch", F: TestDuckDuckGoSearch},
				{Name: "TestJinaAISearch", F: TestJinaAISearch},
				{Name: "TestWikipediaSearch", F: TestWikipediaSearch},
				{Name: "TestSmartSearch", F: TestSmartSearch},
				{Name: "TestEmptyResultsHandling", F: TestEmptyResultsHandling},
				{Name: "TestMultipleSearchSources", F: TestMultipleSearchSources},
			},
			nil,
			nil,
		)
	} else {
		// Demo run
		fmt.Println("YOLO Web Search Improvements - Demo")
		fmt.Println("==================================\n")

		testQueries := []string{
			"Golang programming best practices",
			"Yolo agent architecture",
			"Go web development",
		}

		for _, query := range testQueries {
			fmt.Printf("\nSearching: %s\n", query)
			result, err := SmartSearch(query)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
			} else {
				fmt.Printf("Sources tried: %v\n", result.Sources)
				for i, r := range result.Results {
					truncated := r
					if len(truncated) > 200 {
						truncated = truncated[:200] + "..."
					}
					fmt.Printf("Result %d: %.150s\n", i+1, truncated)
				}
			}
		}

		// Test error handling with empty query
		fmt.Println("\nTesting error handling...")
		_, err := SmartSearch("")
		if err != nil {
			fmt.Printf("Empty query correctly handled: %v\n", err)
		}

		fmt.Println("\nDemo complete!")
	}
}