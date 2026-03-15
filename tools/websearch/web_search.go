package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// WebSearcher wraps the web search functionality with configurable settings.
type WebSearcher struct {
	DuckDuckGoURL string
	WikipediaURL  string
	Client        *http.Client
	RetryCount    int
	BaseTimeout   time.Duration
}

// SearchResult represents a search result item.
type SearchResult struct {
	Title   string `json:"title"`
	Link    string `json:"link"`
	Snippet string `json:"snippet"`
}

// InstantAnswer contains instant answer information.
type InstantAnswer struct {
	Text       string `json:"text"`
	URL        string `json:"url,omitempty"`
	Type       string `json:"type,omitempty"`
	IsFeatured bool   `json:"is_featured"`
}

// RelatedTopics represents related topics in search results.
type RelatedTopics struct {
	Title   string `json:"title,omitempty"`
	Href    string `json:"href,omitempty"`
	Text    string `json:"text,omitempty"`
	URL     string `json:"url,omitempty"`
	FirstURL string `json:"first_url,omitempty"`
}

// WebSearchResult is the main result structure.
type WebSearchResult struct {
	Query        string           `json:"query"`
	InstantAnswer []InstantAnswer  `json:"instant_answer"`
	RelatedTopics []RelatedTopics `json:"related_topics"`
	Results       []SearchResult  `json:"results,omitempty"`
	Error         string          `json:"error,omitempty"`
}

// NewWebSearcher creates a new WebSearcher with default configuration.
func NewWebSearcher() *WebSearcher {
	return &WebSearcher{
		DuckDuckGoURL: "https://html.duckduckgo.com/html/",
		WikipediaURL:  "https://en.wikipedia.org/api/rest_v1/page/summary/",
		Client: &http.Client{
			Timeout: 30 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		RetryCount:  2,
		BaseTimeout: 30 * time.Second,
	}
}

// NewWebSearcherWithConfig creates a WebSearcher with custom configuration.
func NewWebSearcherWithConfig(client *http.Client, retryCount int, timeout time.Duration) *WebSearcher {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	return &WebSearcher{
		DuckDuckGoURL: "https://html.duckduckgo.com/html/",
		WikipediaURL:  "https://en.wikipedia.org/api/rest_v1/page/summary/",
		Client:        client,
		RetryCount:    retryCount,
		BaseTimeout:   timeout,
	}
}

// Search performs a web search with proper context support and timeout handling.
func (ws *WebSearcher) Search(ctx context.Context, query string, limit int) (*WebSearchResult, error) {
	if query == "" {
		return nil, fmt.Errorf("query cannot be empty")
	}

	if limit <= 0 {
		limit = 5
	}
	if limit > 10 {
		limit = 10
	}

	searchQuery := url.QueryEscape(query)
	searchURL := fmt.Sprintf("%s?q=%s&format=json", ws.DuckDuckGoURL, searchQuery)

	return ws.searchWithRetry(ctx, searchURL, query, limit)
}

// SearchWithWikipediaFallback attempts DuckDuckGo first, falls back to Wikipedia if no results.
func (ws *WebSearcher) SearchWithWikipediaFallback(ctx context.Context, query string, limit int) (*WebSearchResult, error) {
	result, err := ws.Search(ctx, query, limit)
	if err != nil {
		return nil, err
	}

	// If DuckDuckGo returned no results, try Wikipedia
	if len(result.InstantAnswer) == 0 && len(result.RelatedTopics) == 0 {
		wikiResult := ws.tryWikipedia(ctx, query)
		if wikiResult != nil {
			result.InstantAnswer = append(result.InstantAnswer, *wikiResult)
		}
	}

	return result, nil
}

// tryWikipedia attempts to get information from Wikipedia.
func (ws *WebSearcher) tryWikipedia(ctx context.Context, query string) *InstantAnswer {
	wikiTitle := url.QueryEscape(query)
	wikiURL := fmt.Sprintf("%s%s", ws.WikipediaURL, wikiTitle)

	// Create timeout context based on base timeout
	searchCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(searchCtx, "GET", wikiURL, nil)
	if err != nil {
		return nil
	}

	req.Header.Set("User-Agent", "YOLO Agent/1.0")

	resp, err := ws.Client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	var wikiData map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&wikiData); err != nil {
		return nil
	}

	title, _ := wikiData["title"].(string)
	extract, _ := wikiData["extract"].(string)
	url, _ := wikiData["content_urls"].(map[string]interface{})["desktop"].(map[string]interface{})["url"].(string)

	if title == "" {
		return nil
	}

	return &InstantAnswer{
		Text:       extract,
		URL:        url,
		Type:       "wikipedia",
		IsFeatured: true,
	}
}

// searchWithRetry performs the actual search with retry logic.
func (ws *WebSearcher) searchWithRetry(ctx context.Context, searchURL, query string, limit int) (*WebSearchResult, error) {
	var lastErr error
	
	for attempt := 0; attempt <= ws.RetryCount; attempt++ {
		result, err := ws.performSearch(ctx, searchURL, query, limit)
		if err == nil {
			return result, nil
		}

		lastErr = err
		
		// Check if it's a transient error that we can retry
		if isTransientError(err) && attempt < ws.RetryCount {
			retryTime := time.Duration(attempt+1) * time.Second
			select {
			case <-time.After(retryTime):
				continue
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
		
		return result, err
	}

	return &WebSearchResult{
		Query: query,
		Error: lastErr.Error(),
	}, nil
}

// performSearch makes the actual HTTP request to DuckDuckGo.
func (ws *WebSearcher) performSearch(ctx context.Context, searchURL, query string, limit int) (*WebSearchResult, error) {
	// Create a timeout context based on base timeout
	searchCtx, cancel := context.WithTimeout(ctx, ws.BaseTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(searchCtx, "GET", searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "YOLO Agent/1.0")

	resp, err := ws.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return &WebSearchResult{
			Query: query,
			Error: fmt.Sprintf("DuckDuckGo returned status code %d", resp.StatusCode),
		}, nil
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024)) // Max 1MB
	if err != nil {
		return &WebSearchResult{
			Query: query,
			Error: fmt.Sprintf("failed to read response body: %w", err),
		}, nil
	}

	var ddgResponse struct {
		Abstract     string `json:"abstract"`
		AbstractText string `json:"abstract_text"`
		Title        string `json:"title"`
		URL          string `json:"url"`
		ResultIndex  int    `json:"result_index"`
	}

	if err := json.Unmarshal(body, &ddgResponse); err == nil && ddgResponse.Abstract != "" {
		return &WebSearchResult{
			Query: query,
			InstantAnswer: []InstantAnswer{{
				Text:       ddgResponse.Abstract + " " + ddgResponse.AbstractText,
				URL:        ddgResponse.URL,
				Type:       "duckduckgo",
				IsFeatured: true,
			}},
		}, nil
	}

	// Parse results from JSON format
	var searchResults WebSearchResult
	
	if strings.HasPrefix(string(body), "<!DOCTYPE") {
		return ws.parseHTMLResults(ctx, string(body), query)
	}

	// Try parsing as JSON (DuckDuckGo JSON endpoint format)
	var jsonBody struct {
		RelatedTopics []RelatedTopics `json:"RelatedTopics"`
	}
	
	if err := json.Unmarshal(body, &jsonBody); err == nil && len(jsonBody.RelatedTopics) > 0 {
		searchResults.Query = query
		searchResults.RelatedTopics = jsonBody.RelatedTopics
		return &searchResults, nil
	}

	return &searchResults, nil
}

// parseHTMLResults parses HTML results from DuckDuckGo.
func (ws *WebSearcher) parseHTMLResults(ctx context.Context, htmlContent, query string) (*WebSearchResult, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return &WebSearchResult{
			Query: query,
			Error: fmt.Sprintf("failed to parse HTML: %w", err),
		}, nil
	}

	var results []SearchResult
	
	doc.Find("a.result__a").Each(func(i int, s *goquery.Selection) {
		if i >= 5 {
			return // Limit to 5 results
		}
		
		title := strings.TrimSpace(s.Text())
		href, _ := s.Attr("href")
		
		if title != "" && href != "" {
			results = append(results, SearchResult{
				Title: title,
				Link:  href,
			})
		}
	})

	return &WebSearchResult{
		Query:   query,
		Results: results,
	}, nil
}

// SearchWithCount performs a web search with custom count parameter.
func (ws *WebSearcher) SearchWithCount(ctx context.Context, query string, limit int) (*WebSearchResult, error) {
	return ws.Search(ctx, query, limit)
}

// IsTransientError checks if an error is transient and worth retrying.
func isTransientError(err error) bool {
	errMsg := err.Error()
	transientPatterns := []string{
		"timeout", "connection refused", "network unavailable", 
		"temporary failure", "rate limit", "502", "503", "504",
	}
	
	for _, pattern := range transientPatterns {
		if strings.Contains(strings.ToLower(errMsg), pattern) {
			return true
		}
	}
	
	return false
}

// SetTimeout sets the base timeout for web searches.
func (ws *WebSearcher) SetTimeout(duration time.Duration) {
	ws.BaseTimeout = duration
	ws.Client.Timeout = duration
}

// SetRetryCount sets the number of retry attempts.
func (ws *WebSearcher) SetRetryCount(count int) {
	if count < 0 {
		count = 0
	}
	ws.RetryCount = count
}

// SetUserAgent sets a custom user agent for requests.
func (ws *WebSearcher) SetUserAgent(ua string) {
	originalClient := ws.Client
	var newClient *http.Client
	
	if originalClient != nil {
		newClient = &http.Client{
			Timeout: originalClient.Timeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
	} else {
		newClient = &http.Client{Timeout: ws.BaseTimeout}
	}
	
	ws.Client = newClient
	
	// Custom transport to set user agent
	trans := &http.Transport{
		UserAgent: ua,
	}
	newClient.Transport = trans
}

// ValidateContext checks if the context is valid.
func ValidateContext(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

// GetEnvironmentVariables returns environment variables for external services.
func GetEnvironmentVariables() map[string]string {
	return map[string]string{
		"YoloAgent":     "YOLO Agent/1.0",
		"Version":       "1.0.0",
		"User-AgentEnv": os.Getenv("USER_AGENT"),
	}
}

// Global searcher instance for backwards compatibility
var globalSearcher *WebSearcher

// init initializes the global searcher.
func init() {
	globalSearcher = NewWebSearcher()
}
