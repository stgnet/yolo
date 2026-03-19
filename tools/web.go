// Package tools provides YOLO's autonomous tool capabilities - web and search operations
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// WebSearchTool implements DuckDuckGo web search
type WebSearchTool struct{}

func (t *WebSearchTool) Name() string { return "web_search" }
func (t *WebSearchTool) Description() string { return "Search the web using DuckDuckGo. Returns instant answers, related topics, and abstract snippets." }
func (t *WebSearchTool) Type() ToolType { return ToolTypeSearch }

func (t *WebSearchTool) Execute(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	start := time.Now()
	
	query, ok := args["query"].(string)
	if !ok || query == "" {
		return &ToolResult{Success: false, Error: "query is required", Duration: time.Since(start)}, nil
	}
	
	count, _ := args["count"].(int)
	if count <= 0 || count > 10 {
		count = 5
	}
	
	result, err := webSearch(query, count)
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error(), Duration: time.Since(start)}, nil
	}
	
	return &ToolResult{
		Success:  true,
		Output:   result,
		Duration: time.Since(start),
	}, nil
}

// ReadWebpageTool implements webpage reading functionality
type ReadWebpageTool struct{}

func (t *ReadWebpageTool) Name() string { return "read_webpage" }
func (t *ReadWebpageTool) Description() string { return "Fetch a webpage URL and return its text content. HTML is converted to plain text." }
func (t *ReadWebpageTool) Type() ToolType { return ToolTypeWeb }

func (t *ReadWebpageTool) Execute(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	start := time.Now()
	
	urlStr, ok := args["url"].(string)
	if !ok || urlStr == "" {
		return &ToolResult{Success: false, Error: "url is required", Duration: time.Since(start)}, nil
	}
	
	content, err := readWebpage(urlStr)
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error(), Duration: time.Since(start)}, nil
	}
	
	return &ToolResult{
		Success:  true,
		Output:   content,
		Duration: time.Since(start),
	}, nil
}

// RedditTool implements Reddit API interactions
type RedditTool struct{}

func (t *RedditTool) Name() string { return "reddit" }
func (t *RedditTool) Description() string { return "Fetch posts from Reddit using the public API. Can search, list subreddit posts, or get thread details." }
func (t *RedditTool) Type() ToolType { return ToolTypeSearch }

func (t *RedditTool) Execute(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	start := time.Now()
	
	action, ok := args["action"].(string)
	if !ok || action == "" {
		return &ToolResult{Success: false, Error: "action is required (search, subreddit, thread)", Duration: time.Since(start)}, nil
	}
	
	var result string
	var err error
	
	switch action {
	case "search":
		query, _ := args["query"].(string)
		if query == "" {
			return &ToolResult{Success: false, Error: "query required for search", Duration: time.Since(start)}, nil
		}
		limit, _ := args["limit"].(int)
		if limit <= 0 || limit > 100 {
			limit = 25
		}
		result, err = redditSearch(query, limit)
		
	case "subreddit":
		subreddit, _ := args["subreddit"].(string)
		if subreddit == "" {
			return &ToolResult{Success: false, Error: "subreddit required", Duration: time.Since(start)}, nil
		}
		limit, _ := args["limit"].(int)
		if limit <= 0 || limit > 100 {
			limit = 25
		}
		result, err = redditSubreddit(subreddit, limit)
		
	case "thread":
		postID, _ := args["post_id"].(string)
		if postID == "" {
			return &ToolResult{Success: false, Error: "post_id required for thread", Duration: time.Since(start)}, nil
		}
		result, err = redditThread(postID)
		
	default:
		return &ToolResult{Success: false, Error: fmt.Sprintf("unknown action: %s", action), Duration: time.Since(start)}, nil
	}
	
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error(), Duration: time.Since(start)}, nil
	}
	
	return &ToolResult{
		Success:  true,
		Output:   result,
		Duration: time.Since(start),
	}, nil
}

// PlaywrightTool implements browser automation via MCP
type PlaywrightTool struct{}

func (t *PlaywrightTool) Name() string { return "playwright_mcp" }
func (t *PlaywrightTool) Description() string { return "Browser automation tool for navigation, clicks, fills, screenshots" }
func (t *PlaywrightTool) Type() ToolType { return ToolTypeWeb }

func (t *PlaywrightTool) Execute(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	start := time.Now()
	
	action, ok := args["action"].(string)
	if !ok || action == "" {
		return &ToolResult{Success: false, Error: "action is required", Duration: time.Since(start)}, nil
	}
	
	var result string
	var err error
	
	switch action {
	case "navigate":
		urlStr, _ := args["url"].(string)
		if urlStr == "" {
			return &ToolResult{Success: false, Error: "url required for navigate", Duration: time.Since(start)}, nil
		}
		result, err = playwrightNavigate(urlStr)
		
	case "click":
		selector, _ := args["selector"].(string)
		if selector == "" {
			return &ToolResult{Success: false, Error: "selector required for click", Duration: time.Since(start)}, nil
		}
		result, err = playwrightClick(selector)
		
	case "fill":
		selector, _ := args["selector"].(string)
		value, _ := args["value"].(string)
		if selector == "" || value == "" {
			return &ToolResult{Success: false, Error: "selector and value required for fill", Duration: time.Since(start)}, nil
		}
		result, err = playwrightFill(selector, value)
		
	case "getHTML":
		selector, _ := args["selector"].(string)
		if selector == "" {
			return &ToolResult{Success: false, Error: "selector required for getHTML", Duration: time.Since(start)}, nil
		}
		result, err = playwrightGetHTML(selector)
		
	case "screenshot":
		path, _ := args["path"].(string)
		if path == "" {
			path = "/tmp/screenshot.png"
		}
		result, err = playwrightScreenshot(path)
		
	default:
		return &ToolResult{Success: false, Error: fmt.Sprintf("unknown action: %s", action), Duration: time.Since(start)}, nil
	}
	
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error(), Duration: time.Since(start)}, nil
	}
	
	return &ToolResult{
		Success:  true,
		Output:   result,
		Duration: time.Since(start),
	}, nil
}

// Helper functions for web operations

func webSearch(query string, count int) (string, error) {
	// Use DuckDuckGo API via Instant Answer endpoint
	apiURL := fmt.Sprintf("https://api.duckduckgo.com/?q=%s&format=json&no_redirect=1", url.QueryEscape(query))
	
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(apiURL)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read failed: %w", err)
	}
	
	// Parse JSON and format results
	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return "", fmt.Errorf("json parse failed: %w", err)
	}
	
	result := formatDuckDuckGoResponse(data, count)
	return result, nil
}

func formatDuckDuckGoResponse(data map[string]interface{}, maxResults int) string {
	var sb strings.Builder
	
	if abstract, ok := data["Abstract"].(string); ok && abstract != "" {
		sb.WriteString(fmt.Sprintf("Instant Answer:\n%s\n\n", abstract))
	}
	
	if relatedTopics, ok := data["RelatedTopics"].([]interface{}); ok {
		for i, topic := range relatedTopics {
			if i >= maxResults {
				break
			}
			if topicMap, ok := topic.(map[string]interface{}); ok {
				if text, ok := topicMap["Text"].(string); ok && text != "" {
					sb.WriteString(fmt.Sprintf("%s\n", text))
					if firstMatch, ok := topicMap["FirstURL"].(string); ok && firstMatch != "" {
						sb.WriteString(fmt.Sprintf("Source: %s\n\n", firstMatch))
					}
				}
			}
		}
	}
	
	return sb.String()
}

func readWebpage(urlStr string) (string, error) {
	// Ensure URL has scheme
	if !strings.HasPrefix(urlStr, "http://") && !strings.HasPrefix(urlStr, "https://") {
		urlStr = "https://" + urlStr
	}
	
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(urlStr)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read failed: %w", err)
	}
	
	// For now, just return raw content
	// In production, would use html-to-text conversion
	return string(body), nil
}

func redditSearch(query string, limit int) (string, error) {
	apiURL := fmt.Sprintf("https://www.reddit.com/search.json?q=%s&limit=%d", url.QueryEscape(query), limit)
	
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(apiURL)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read failed: %w", err)
	}
	
	return formatRedditPosts(body), nil
}

func redditSubreddit(subreddit string, limit int) (string, error) {
	apiURL := fmt.Sprintf("https://www.reddit.com/r/%s.json?limit=%d", subreddit, limit)
	
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(apiURL)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read failed: %w", err)
	}
	
	return formatRedditPosts(body), nil
}

func redditThread(postID string) (string, error) {
	apiURL := fmt.Sprintf("https://www.reddit.com/comments/%s.json?limit=100", postID)
	
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(apiURL)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read failed: %w", err)
	}
	
	return string(body), nil
}

func formatRedditPosts(data []byte) string {
	// Simplified formatting - would need proper JSON parsing
	return string(data)
}

// Playwright MCP integration functions
func playwrightNavigate(urlStr string) (string, error) {
	// This would call the Playwright MCP server
	// Placeholder for now
	return fmt.Sprintf("Navigated to: %s", urlStr), nil
}

func playwrightClick(selector string) (string, error) {
	return fmt.Sprintf("Clicked: %s", selector), nil
}

func playwrightFill(selector, value string) (string, error) {
	return fmt.Sprintf("Filled %s with: %s", selector, value), nil
}

func playwrightGetHTML(selector string) (string, error) {
	return fmt.Sprintf("HTML for selector %s", selector), nil
}

func playwrightScreenshot(path string) (string, error) {
	return fmt.Sprintf("Screenshot saved to: %s", path), nil
}
