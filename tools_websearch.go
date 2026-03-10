// Web Search Tool Implementation
// Uses multiple strategies: Wikipedia API, DuckDuckGo, and shell commands for flexibility

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"strings"
)

func (t *ToolExecutor) webSearch(args map[string]any) string {
	query := getStringArg(args, "query", "")
	if query == "" {
		return "Error: query parameter is required"
	}

	count := getIntArg(args, "count", 5)
	if count > 10 {
		count = 10
	}
	if count < 1 {
		count = 1
	}

	result := &strings.Builder{}

	// Strategy 1: Try Wikipedia API for authoritative definitions
	wikiResults := searchWikipedia(query, 3)
	if wikiResults != "" {
		result.WriteString(fmt.Sprintf("\n📚 **From Wikipedia:**\n%s\n", wikiResults))
	}

	// Strategy 2: Try Bing search via curl (more reliable than DDG Instant API)
	bingResults := searchBing(query, count)
	if bingResults != "" {
		result.WriteString(fmt.Sprintf("\n🌐 **Web Search Results:**\n%s\n", bingResults))
	}

	output := result.String()
	if strings.TrimSpace(output) == "" {
		output = fmt.Sprintf(
			"\n⚠️ No results found for '%s'. \n\n"+
				"Consider:\n"+
				"  - Using more specific keywords\n"+
				"  - Try Reddit tool: `reddit action=search query=\"%s\"`\n"+
				"  - Manual search: open your browser",
			query, query)
	}

	return output
}

func searchWikipedia(query string, maxResults int) string {
	urlStr := fmt.Sprintf("https://en.wikipedia.org/w/api.php?action=query&list=search&srsearch=%s&format=json&origin=*&srlimit=%d",
		url.QueryEscape(query), maxResults)

	resp, err := http.Get(urlStr)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}

	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return ""
	}

	if queryObj, ok := data["query"].(map[string]interface{}); ok {
		if searchResults, ok := queryObj["search"].([]interface{}); ok {
			var sb strings.Builder
			for i, item := range searchResults {
				if i >= maxResults {
					break
				}
				if m, ok := item.(map[string]interface{}); ok {
					if title, ok := m["title"].(string); ok && title != "" {
						sb.WriteString(fmt.Sprintf("  • %s", title))
						if snippet, ok := m["snippet"].(string); ok && snippet != "" {
							// Clean up HTML entities and ellipsis
							snippet = strings.ReplaceAll(snippet, "…", "...")
							snippet = strings.TrimSpace(snippet)
							if len(snippet) > 150 {
								snippet = snippet[:150] + "..."
							}
							sb.WriteString(fmt.Sprintf("\n    %s", snippet))
						}
						sb.WriteString("\n")
					}
				}
			}
			if sb.Len() > 0 {
				return sb.String()
			}
		}
	}
	return ""
}

func searchBing(query string, maxResults int) string {
	// Use curl to fetch Bing search results and parse them
	escapedQuery := url.QueryEscape(query)
	cmd := exec.Command("curl", "-s", "--user-agent", "Mozilla/5.0",
		fmt.Sprintf("https://www.bing.com/search?q=%s&count=%d", escapedQuery, maxResults*2))

	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	html := string(output)
	
	// Simple extraction of search result titles and snippets
	var results strings.Builder
	
	// Look for h2 tags with result titles (Bing's format)
	lines := strings.Split(html, "\n")
	for i, line := range lines {
		if strings.Contains(line, "<h2") && strings.Contains(line, "title=") {
			// Try to get title from next few lines
			titleStart := strings.Index(line, ">")
			titleEnd := strings.Index(line, "</h2>")
			if titleStart != -1 && titleEnd > titleStart {
				title := line[titleStart+1 : titleEnd]
				title = cleanHTML(title)
				if len(title) > 5 && !strings.Contains(title, "Microsoft ") {
					results.WriteString(fmt.Sprintf("  • %s\n", title))
					
					// Try to get URL from nearby
					for j := i - 2; j < i+5 && j < len(lines); j++ {
						if idx := strings.Index(lines[j], "href=\""); idx != -1 {
							urlStart := idx + 6
							urlEnd := strings.Index(lines[j][urlStart:], "\"")
							if urlEnd != -1 {
								linkURL := lines[j][urlStart : urlStart+urlEnd]
								if strings.Contains(linkURL, "bing.com/search?q=") == false &&
									strings.Contains(title, linkURL) == false {
									results.WriteString(fmt.Sprintf("    %s\n", linkURL))
									break
								}
							}
						}
					}
				}
			}
		}
	}

	if results.Len() > 50 {
		return results.String()
	}
	return ""
}

func cleanHTML(html string) string {
	html = strings.ReplaceAll(html, "&amp;", "&")
	html = strings.ReplaceAll(html, "&quot;", "\"")
	html = strings.ReplaceAll(html, "&lt;", "<")
	html = strings.ReplaceAll(html, "&gt;", ">")
	html = strings.ReplaceAll(html, "&nbsp;", " ")
	
	// Remove any remaining HTML tags
	var result strings.Builder
	inTag := false
	for _, c := range html {
		if c == '<' {
			inTag = true
		} else if c == '>' {
			inTag = false
		} else if !inTag {
			result.WriteRune(c)
		}
	}
	return strings.TrimSpace(result.String())
}
