// Package web_search provides web search functionality for YOLO
package web_search

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// SearchResult represents a web search result
type SearchResult struct {
	Query   string   `json:"query"`
	Results []string `json:"results"`
	Sources []string `json:"sources"`
}

// MockHTTPClient implements http.Client for testing with configurable behavior
type MockHTTPClient struct {
	Response      *http.Response
	Body          io.Reader
	Error         error
	StatusCode    int
	Header        http.Header
	Delay         time.Duration
	RateLimit     bool
	RetryAfter    int
}

// Do implements the RoundTripper interface with custom behaviors for testing
func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	if m.Delay > 0 {
		time.Sleep(m.Delay)
	}
	if m.Error != nil {
		return nil, m.Error
	}

	body := io.NopCloser(m.Body)
	resp := &http.Response{
		StatusCode: m.StatusCode,
		Body:       body,
		Header:     m.Header,
		Request:    req,
	}
	return resp, nil
}

// newMockResponse creates a mock HTTP response for testing
func newMockResponse(body string, statusCode int, header http.Header) *MockHTTPClient {
	mock := &MockHTTPClient{
		Body:       strings.NewReader(body),
		StatusCode: statusCode,
		Header:     make(http.Header),
	}
	if header != nil {
		for k, v := range header {
			mock.Header[k] = v
		}
	}
	return mock
}

// DuckDuckGoSearch searches using DuckDuckGo API (limited to 5 results)
func DuckDuckGoSearch(query string, limit int) ([]string, error) {
	if limit > 5 {
		limit = 5 // DuckDuckGo max is 5
	}

	url := fmt.Sprintf("https://api.duckduckgo.com/?q=%s&format=json&limit=%d",
		url.QueryEscape(query), limit)

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
	var abstract string
	if val, ok := result["Abstract"].(string); ok {
		abstract = val
	}

	if abstract != "" {
		return []string{abstract}, nil
	}

	return nil, fmt.Errorf("empty results from DuckDuckGo")
}

// JinaAISearch uses Jina AI as a more reliable search alternative
func JinaAISearch(query string) ([]string, error) {
	apiURL := "https://r.jina.ai/" + url.QueryEscape("https://duckduckgo.com/?q="+query)

	req, err := http.NewRequestWithContext(context.Background(), "GET", apiURL, nil)
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
	apiURL := fmt.Sprintf("https://en.wikipedia.org/api/rest_v1/page/summary/%s",
		url.QueryEscape(query))

	req, err := http.NewRequestWithContext(context.Background(), "GET", apiURL, nil)
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
