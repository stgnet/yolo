package main

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
)

// ─── Web Search Tool Implementation ─────────────────────────────────

// webSearchResult represents a search result from DuckDuckGo Instant Answer
type webSearchResult struct {
	Abstract       string `json:"abstract"`
	AbstractSource string `json:"abstract_source"`
	AbstractURL    string `json:"abstract_url"`
	Url            string `json:"url"`
	Image          string `json:"image"`
	RelatedTopics  []struct {
		Title     string `json:"text,omitempty"`
		TopicName string `json:"topic_name"`
		Content   struct {
			Text string `json:"text"`
		} `json:"text_content,omitempty"`
		FirstValue string `json:"first_value"`
	} `json:"related_topics,omitempty"`
}

// searchCacheEntry represents a cached web search result
type searchCacheEntry struct {
	Result string    `json:"result"`
	Ts     time.Time `json:"ts"`
}

// searchCache is a thread-safe in-memory cache for web search results
var searchCache = &sync.Map{}
var searchCacheTTL = 5 * time.Minute

func getSearchCacheKey(query string, count int) string {
	hash := md5.Sum([]byte(fmt.Sprintf("%s:%d", strings.ToLower(query), count)))
	return fmt.Sprintf("%x", hash[:8])
}

func (t *ToolExecutor) getFromSearchCache(key string) (string, bool) {
	if entry, ok := searchCache.Load(key); ok {
		if e, ok := entry.(*searchCacheEntry); ok {
			if time.Since(e.Ts) < searchCacheTTL {
				return e.Result, true
			}
		}
		searchCache.Delete(key)
	}
	return "", false
}

func (t *ToolExecutor) addToSearchCache(key, result string) {
	if result != "" && !t.isEmptySearchResult(result) {
		searchCache.Store(key, &searchCacheEntry{
			Result: result,
			Ts:     time.Now(),
		})
	}
}

func (t *ToolExecutor) webSearch(args map[string]any) string {
	query := getStringArg(args, "query", "")
	if query == "" {
		return errorMessage("'query' parameter is required")
	}

	count := getIntArg(args, "count", 5)
	if count > 10 {
		count = 10
	}
	if count < 1 {
		count = 5
	}

	cacheKey := getSearchCacheKey(query, count)

	if cachedResult, ok := t.getFromSearchCache(cacheKey); ok {
		return fmt.Sprintf("[Cached] %s", cachedResult)
	}

	ddgResult := t.searchDuckDuckGo(query, count)

	if !t.isEmptySearchResult(ddgResult) {
		t.addToSearchCache(cacheKey, ddgResult)
		return ddgResult
	}

	wikiResult := t.searchWikipedia(query, count)

	if t.isEmptySearchResult(wikiResult) {
		return fmt.Sprintf("No search results found for \"%s\". DuckDuckGo and Wikipedia returned no relevant information.\n\nTry:\n- Using more specific keywords\n- Searching for a different topic\n- Checking spelling of terms", query)
	}

	if ddgResult != "" && !t.isEmptySearchResult(ddgResult) {
		combined := ddgResult + "\n---\n\n" + wikiResult
		t.addToSearchCache(cacheKey, combined)
		return combined
	}

	t.addToSearchCache(cacheKey, wikiResult)
	return wikiResult
}

func (t *ToolExecutor) isEmptySearchResult(result string) bool {
	emptyPatterns := []string{
		"No results found",
		"Error:",
		"returned no relevant information",
		"Try a different search term",
	}

	for _, pattern := range emptyPatterns {
		if strings.Contains(result, pattern) {
			return true
		}
	}

	if len(result) < 100 {
		return true
	}

	return false
}

func (t *ToolExecutor) searchDuckDuckGo(query string, count int) string {
	result := t.searchDuckDuckGoWithRetry(query, count, 3)
	return result
}

func (t *ToolExecutor) searchDuckDuckGoWithRetry(query string, count int, maxRetries int) string {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		url := fmt.Sprintf("https://api.duckduckgo.com/?q=%s&format=json&no_html=1",
			url.QueryEscape(query))

		client := &http.Client{Timeout: 15 * time.Second}
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			lastErr = fmt.Errorf("could not create DuckDuckGo request: %w", err)
			continue
		}

		req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; YOLO-Search-Bot/1.0)")

		resp, err := client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("could not fetch from DuckDuckGo: %w", err)
			if attempt < maxRetries {
				delay := time.Duration(attempt+1) * 2 * time.Second
				time.Sleep(delay)
			}
			continue
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = fmt.Errorf("could not read DuckDuckGo response: %w", err)
			if attempt < maxRetries {
				delay := time.Duration(attempt+1) * 2 * time.Second
				time.Sleep(delay)
			}
			continue
		}

		var iaResult map[string]any
		if err := json.Unmarshal(body, &iaResult); err == nil {
			result := t.parseDuckDuckGoJSON(query, count, body)
			if !t.isEmptySearchResult(result) {
				return result
			}
		}

		return ""
	}

	if lastErr != nil {
		return errorMessage("DuckDuckGo search failed after %d retries: %v", maxRetries+1, lastErr)
	}

	return ""
}

func (t *ToolExecutor) searchWikipedia(query string, count int) string {
	result := t.searchWikipediaWithRetry(query, count, 3)
	return result
}

func (t *ToolExecutor) searchWikipediaWithRetry(query string, count int, maxRetries int) string {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		urlStr := fmt.Sprintf("https://en.wikipedia.org/w/api.php?action=query&list=search&srsearch=%s&format=json&origin=*&srlimit=%d",
			url.QueryEscape(query), count)

		client := &http.Client{Timeout: 15 * time.Second}
		req, err := http.NewRequest("GET", urlStr, nil)
		if err != nil {
			lastErr = fmt.Errorf("could not create Wikipedia request: %w", err)
			continue
		}

		req.Header.Set("User-Agent", "YOLO-Search-Bot/1.0 (yolo@b-haven.org)")

		resp, err := client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("could not fetch from Wikipedia: %w", err)
			if attempt < maxRetries {
				delay := time.Duration(attempt+1) * 2 * time.Second
				time.Sleep(delay)
			}
			continue
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = fmt.Errorf("could not read Wikipedia response: %w", err)
			if attempt < maxRetries {
				delay := time.Duration(attempt+1) * 2 * time.Second
				time.Sleep(delay)
			}
			continue
		}

		var result struct {
			Query struct {
				Search []struct {
					Title    string `json:"title"`
					PageID   int    `json:"pageid"`
					Snippet  string `json:"snippet"`
					Fragment string `json:"fragment"`
				} `json:"search"`
			} `json:"query"`
		}

		if err := json.Unmarshal(body, &result); err != nil {
			lastErr = fmt.Errorf("could not parse Wikipedia JSON: %w", err)
			continue
		}

		if len(result.Query.Search) == 0 {
			return ""
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Wikipedia results for \"%s\":\n\n", query))

		for i, article := range result.Query.Search {
			sb.WriteString(fmt.Sprintf("%d. **[%s](https://en.wikipedia.org/wiki/%s)**\n",
				i+1,
				article.Title,
				strings.ReplaceAll(article.Title, " ", "_")))

			snippet := article.Snippet
			if article.Fragment != "" {
				snippet = article.Fragment
			}

			snippet = strings.ReplaceAll(snippet, "&amp;", "&")
			snippet = strings.ReplaceAll(snippet, "&lt;", "<")
			snippet = strings.ReplaceAll(snippet, "&gt;", ">")
			snippet = regexp.MustCompile(`<[^>]+>`).ReplaceAllString(snippet, "")

			if len(snippet) > 300 {
				snippet = snippet[:300] + "..."
			}

			sb.WriteString(fmt.Sprintf("   %s\n\n", strings.TrimSpace(snippet)))
		}

		return sb.String()
	}

	if lastErr != nil {
		return errorMessage("Wikipedia search failed after %d retries: %v", maxRetries+1, lastErr)
	}

	return ""
}

func (t *ToolExecutor) parseDuckDuckGoJSON(query string, count int, data []byte) string {
	var result struct {
		Query          string `json:"query"`
		Results        int    `json:"results"`
		Answer         string `json:"answer"`
		Abstract       string `json:"abstract"`
		AbstractSource string `json:"abstract_source"`
		AbstractURL    string `json:"abstract_url"`
		Image          string `json:"image"`
		RelatedTopics  []struct {
			Title     string          `json:"text,omitempty"`
			TopicName string          `json:"topic_name"`
			Result    json.RawMessage `json:"result,omitempty"`
			Results   []struct {
				Text string `json:"text"`
				Url  string `json:"url"`
			} `json:"results,omitempty"`
		} `json:"related_topics,omitempty"`
	}

	if err := json.Unmarshal(data, &result); err != nil {
		return errorMessage("parsing JSON: %v", err)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Search results for \"%s\":\n\n", query))

	if result.Answer != "" {
		sb.WriteString(fmt.Sprintf("**Answer:** %s\n\n", result.Answer))
	}

	if result.Abstract != "" {
		sb.WriteString(fmt.Sprintf("**Summary:** %s\n", result.Abstract))
		if result.AbstractSource != "" {
			sb.WriteString(fmt.Sprintf("Source: [from%s](%s)\n\n",
				result.AbstractSource, result.AbstractURL))
		} else {
			sb.WriteString("\n")
		}
	}

	if result.Image != "" {
		sb.WriteString(fmt.Sprintf("[![](%s)](%s)\n\n", result.Image, result.Image))
	}

	resultsCount := 0
	for _, topic := range result.RelatedTopics {
		var topicResults []struct {
			Text string `json:"text"`
			Url  string `json:"url"`
		}

		if len(topic.Result) > 0 {
			var singleResult struct {
				Text string `json:"text"`
				Url  string `json:"url"`
			}
			if err := json.Unmarshal(topic.Result, &singleResult); err == nil && singleResult.Text != "" {
				topicResults = append(topicResults, singleResult)
			}
		}

		topicResults = append(topicResults, topic.Results...)

		if len(topicResults) > 0 {
			if topic.TopicName != "" || topic.Title != "" {
				title := topic.TopicName
				if title == "" {
					title = topic.Title
				}
				sb.WriteString(fmt.Sprintf("\n### %s\n", title))
			}

			for _, r := range topicResults {
				if resultsCount >= count {
					break
				}
				resultsCount++

				if r.Text != "" {
					sb.WriteString(fmt.Sprintf("%d. **%s**\n", resultsCount, r.Text))
				}
				if r.Url != "" {
					sb.WriteString(fmt.Sprintf("   [%s](%s)\n\n", r.Url, r.Url))
				}
			}
		}
	}

	if resultsCount == 0 && result.Abstract == "" && result.Answer == "" {
		sb.WriteString("No results found for this query. Try a different search term.\n")
	}

	return sb.String()
}

func (t *ToolExecutor) parseDuckDuckGoHTML(query string, count int, data []byte) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Search results for \"%s\":\n\n", query))

	lines := strings.Split(string(data), "\n")

	type SearchResult struct {
		Title   string
		URL     string
		Snippet string
	}

	var results []SearchResult

	for i, line := range lines {
		if strings.Contains(line, `class="`) && (strings.Contains(line, "result") || strings.Contains(line, "link")) {
			titleMatch := regexp.MustCompile(`>([^<]+?)<`).FindStringSubmatch(strings.TrimSpace(line))
			if len(titleMatch) > 1 {
				title := titleMatch[1]
				cleanTitle := strings.TrimPrefix(title, "[")
				cleanTitle = strings.TrimSuffix(cleanTitle, "]")

				var url string
				startIdx := i - 2
				if startIdx < 0 {
					startIdx = 0
				}
				for j := startIdx; j <= i+2 && j < len(lines); j++ {
					if strings.Contains(lines[j], `href="http`) {
						urlMatch := regexp.MustCompile(`href="(https?://[^"]+)"`).FindStringSubmatch(lines[j])
						if len(urlMatch) > 1 {
							url = urlMatch[1]
							break
						}
					}
				}

				var snippet string
				for j := i + 1; j < i+5 && j < len(lines); j++ {
					if strings.Contains(lines[j], "<div") || strings.Contains(lines[j], "<span") {
						snippetLines := regexp.MustCompile(`<[^>]+>([^<]+)</[^>]+>`).FindAllStringSubmatch(lines[j], -1)
						for _, sm := range snippetLines {
							if len(sm) > 1 && sm[1] != title && len(sm[1]) > 20 {
								snippet = sm[1]
								break
							}
						}
					}
					if snippet != "" {
						break
					}
				}

				results = append(results, SearchResult{
					Title:   cleanTitle,
					URL:     url,
					Snippet: strings.TrimSpace(snippet),
				})
			}
		}

		if len(results) >= count {
			break
		}
	}

	if len(results) == 0 {
		sb.WriteString("No results found. DuckDuckGo HTML parsing failed.\n")
		return sb.String()
	}

	for i, r := range results {
		sb.WriteString(fmt.Sprintf("%d. **%s**\n", i+1, r.Title))
		if r.URL != "" {
			sb.WriteString(fmt.Sprintf("   [%s](%s)\n", r.URL, r.URL))
		}
		if r.Snippet != "" {
			sb.WriteString(fmt.Sprintf("   %s\n", r.Snippet))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}
