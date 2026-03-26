package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// ─── Web & Browser Improvements ─────────────────────────────────────
//
// Enhanced web tools:
//   - SearXNG search: self-hosted meta-search engine support
//   - Screenshot tool: capture and save page screenshots
//   - Readability extraction: article-focused content extraction
//   - Search engine selection: choose between DuckDuckGo and SearXNG

// ─── SearXNG Search ─────────────────────────────────────────────────
//
// SearXNG is a self-hosted meta-search engine that aggregates results
// from multiple search engines. Configure via SEARXNG_URL env var.

// searxngSearch queries a SearXNG instance and returns formatted results.
func (t *ToolExecutor) searxngSearch(query string, count int) string {
	baseURL := os.Getenv("SEARXNG_URL")
	if baseURL == "" {
		return ""
	}

	searchURL := fmt.Sprintf("%s/search?q=%s&format=json&categories=general&language=en",
		strings.TrimRight(baseURL, "/"), url.QueryEscape(query))

	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		return errorMessage("SearXNG request error: %v", err)
	}
	req.Header.Set("User-Agent", t.getUserAgent())
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return errorMessage("SearXNG fetch error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return errorMessage("SearXNG returned HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return errorMessage("SearXNG read error: %v", err)
	}

	var result struct {
		Results []struct {
			Title   string  `json:"title"`
			URL     string  `json:"url"`
			Content string  `json:"content"`
			Engine  string  `json:"engine"`
			Score   float64 `json:"score"`
		} `json:"results"`
		NumberOfResults int `json:"number_of_results"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return errorMessage("SearXNG parse error: %v", err)
	}

	if len(result.Results) == 0 {
		return ""
	}

	// Limit results
	results := result.Results
	if len(results) > count {
		results = results[:count]
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("SearXNG results for \"%s\":\n\n", query))
	for i, r := range results {
		sb.WriteString(fmt.Sprintf("%d. **%s**\n", i+1, r.Title))
		sb.WriteString(fmt.Sprintf("   %s\n", r.URL))
		if r.Content != "" {
			content := r.Content
			if len(content) > 300 {
				content = content[:300] + "..."
			}
			sb.WriteString(fmt.Sprintf("   %s\n", content))
		}
		if r.Engine != "" {
			sb.WriteString(fmt.Sprintf("   (via %s)\n", r.Engine))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// ─── Enhanced Web Search ────────────────────────────────────────────

// webSearchEnhanced tries SearXNG first (if configured), then falls back
// to the existing DuckDuckGo + Wikipedia pipeline.
func (t *ToolExecutor) webSearchEnhanced(args map[string]any) string {
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

	engine := getStringArg(args, "engine", "")

	cacheKey := getSearchCacheKey(query, count)
	if cachedResult, ok := t.getFromSearchCache(cacheKey); ok {
		return fmt.Sprintf("[Cached] %s", cachedResult)
	}

	// If engine is explicitly specified, use only that engine
	if engine == "searxng" {
		result := t.searxngSearch(query, count)
		if result != "" && !t.isEmptySearchResult(result) {
			t.addToSearchCache(cacheKey, result)
			return result
		}
		return errorMessage("SearXNG returned no results (is SEARXNG_URL set?)")
	}

	if engine == "duckduckgo" || engine == "ddg" {
		return t.webSearch(args)
	}

	// Auto mode: try SearXNG first if configured, then DuckDuckGo
	if os.Getenv("SEARXNG_URL") != "" {
		result := t.searxngSearch(query, count)
		if result != "" && !t.isEmptySearchResult(result) {
			t.addToSearchCache(cacheKey, result)
			return result
		}
	}

	// Fall back to existing search
	return t.webSearch(args)
}

// ─── Screenshot Tool ────────────────────────────────────────────────

// screenshot captures a screenshot of a URL using a headless browser.
func (t *ToolExecutor) screenshot(args map[string]any) string {
	rawURL := getStringArg(args, "url", "")
	if rawURL == "" {
		return errorMessage("'url' parameter is required")
	}

	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		rawURL = "https://" + rawURL
	}

	defaultScreenshot := filepath.Join(t.getTempDir(), "screenshot.png")
	outputPath := getStringArg(args, "path", defaultScreenshot)
	fullPage := getBoolArg(args, "full_page", false)
	width := getIntArg(args, "width", 1280)
	height := getIntArg(args, "height", 720)

	// Resolve output path via safePath if it's relative and not in temp dir
	tempDir := t.getTempDir()
	if outputPath != defaultScreenshot && !strings.HasPrefix(outputPath, tempDir) {
		if resolved, err := t.safePath(outputPath); err == nil {
			outputPath = resolved
		}
	}

	// Try multiple screenshot methods
	if err := screenshotWithPlaywright(rawURL, outputPath, fullPage, width, height); err == nil {
		info, _ := os.Stat(outputPath)
		return fmt.Sprintf("Screenshot saved to %s (%s)\nURL: %s\nSize: %dx%d",
			outputPath, formatSize(info.Size()), rawURL, width, height)
	}

	if err := screenshotWithChromium(rawURL, outputPath, fullPage, width, height); err == nil {
		info, _ := os.Stat(outputPath)
		return fmt.Sprintf("Screenshot saved to %s (%s)\nURL: %s\nSize: %dx%d",
			outputPath, formatSize(info.Size()), rawURL, width, height)
	}

	return errorMessage("no screenshot tool available (install playwright or chromium/google-chrome)")
}

// screenshotWithPlaywright takes a screenshot using Playwright via Node.js.
func screenshotWithPlaywright(url, outputPath string, fullPage bool, width, height int) error {
	if _, err := exec.LookPath("node"); err != nil {
		return fmt.Errorf("node not found")
	}

	fullPageJS := "false"
	if fullPage {
		fullPageJS = "true"
	}

	script := fmt.Sprintf(`
const pw = require('playwright');
(async () => {
  const browser = await pw.chromium.launch({headless: true});
  const page = await browser.newPage({viewport: {width: %d, height: %d}});
  await page.goto('%s', {waitUntil: 'networkidle', timeout: 30000});
  await page.screenshot({path: '%s', fullPage: %s});
  await browser.close();
})().catch(e => { console.error(e.message); process.exit(1); });
`, width, height, escapeJS(url), escapeJS(outputPath), fullPageJS)

	cmd := exec.Command("node", "-e", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("playwright: %v: %s", err, string(output))
	}
	return nil
}

// screenshotWithChromium takes a screenshot using headless Chromium.
func screenshotWithChromium(url, outputPath string, fullPage bool, width, height int) error {
	chromium := ""
	for _, name := range []string{"chromium", "chromium-browser", "google-chrome", "google-chrome-stable"} {
		if path, err := exec.LookPath(name); err == nil {
			chromium = path
			break
		}
	}
	if chromium == "" {
		return fmt.Errorf("no chromium found")
	}

	args := []string{
		"--headless",
		"--disable-gpu",
		"--no-sandbox",
		fmt.Sprintf("--window-size=%d,%d", width, height),
		fmt.Sprintf("--screenshot=%s", outputPath),
		url,
	}

	cmd := exec.Command(chromium, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("chromium: %v: %s", err, string(output))
	}
	return nil
}

// ─── Readability Extraction ─────────────────────────────────────────
//
// extractArticleContent implements a readability-style algorithm that
// identifies the main content area of a web page by scoring text-heavy
// elements and removing boilerplate (nav, sidebar, footer, ads).

// readWebpageReadable fetches a URL and extracts the main article content.
func (t *ToolExecutor) readWebpageReadable(args map[string]any) string {
	rawURL := getStringArg(args, "url", "")
	if rawURL == "" {
		return errorMessage("'url' parameter is required")
	}
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		rawURL = "https://" + rawURL
	}

	mode := getStringArg(args, "mode", "auto")

	// Check cache
	cacheKey := fmt.Sprintf("readable:%s:%s", mode, rawURL)
	if cached, ok := searchCache.Load(cacheKey); ok {
		if e, ok := cached.(*searchCacheEntry); ok {
			if time.Since(e.Ts) < searchCacheTTL {
				return fmt.Sprintf("[Cached] %s", e.Result)
			}
		}
		searchCache.Delete(cacheKey)
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}

	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return errorMessage("invalid URL: %v", err)
	}
	req.Header.Set("User-Agent", t.getUserAgent())
	req.Header.Set("Accept", "text/html,application/xhtml+xml,text/plain,*/*")

	resp, err := client.Do(req)
	if err != nil {
		return errorMessage("fetch failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return errorMessage("HTTP %d fetching '%s'", resp.StatusCode, rawURL)
	}

	const maxBody = 1 << 20
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBody))
	if err != nil {
		return errorMessage("read failed: %v", err)
	}

	html := string(body)
	var text string

	switch mode {
	case "article":
		text = extractArticleContent(html)
	case "full":
		text = htmlToText(html)
	default: // "auto"
		text = extractArticleContent(html)
		if len(text) < 200 {
			// Fall back to full extraction if article mode found too little
			text = htmlToText(html)
		}
	}

	text = strings.TrimSpace(text)
	if text == "" {
		return fmt.Sprintf("Fetched '%s' but no content could be extracted.", rawURL)
	}

	const maxChars = 50000
	if len(text) > maxChars {
		text = text[:maxChars] + "\n\n[Content truncated at 50,000 characters]"
	}

	// Extract title
	title := extractTitle(html)
	var result string
	if title != "" {
		result = fmt.Sprintf("# %s\nSource: %s\n\n%s", title, rawURL, text)
	} else {
		result = fmt.Sprintf("Content from %s:\n\n%s", rawURL, text)
	}

	searchCache.Store(cacheKey, &searchCacheEntry{Result: result, Ts: time.Now()})
	return result
}

// extractTitle pulls the <title> from HTML.
func extractTitle(html string) string {
	re := regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)
	m := re.FindStringSubmatch(html)
	if len(m) > 1 {
		title := strings.TrimSpace(m[1])
		title = regexp.MustCompile(`<[^>]+>`).ReplaceAllString(title, "")
		title = strings.ReplaceAll(title, "&amp;", "&")
		title = strings.ReplaceAll(title, "&#x27;", "'")
		return title
	}
	return ""
}

// extractArticleContent implements a readability-style extraction.
// It identifies the main content area by:
// 1. Removing boilerplate elements (nav, footer, sidebar, ads)
// 2. Finding the element with the most paragraph text
// 3. Preserving structure (headings, lists, paragraphs)
func extractArticleContent(html string) string {
	// Step 1: Remove boilerplate
	boilerplatePatterns := []string{
		`(?is)<nav[^>]*>.*?</nav>`,
		`(?is)<footer[^>]*>.*?</footer>`,
		`(?is)<aside[^>]*>.*?</aside>`,
		`(?is)<header[^>]*>.*?</header>`,
		`(?is)<script[^>]*>.*?</script>`,
		`(?is)<style[^>]*>.*?</style>`,
		`(?is)<head[^>]*>.*?</head>`,
		`(?is)<noscript[^>]*>.*?</noscript>`,
		`(?is)<iframe[^>]*>.*?</iframe>`,
		// Common ad/sidebar class patterns
		`(?is)<[^>]*class="[^"]*(?:sidebar|ad-|advertisement|social-share|cookie|popup|modal|newsletter|comment)[^"]*"[^>]*>.*?</(?:div|section|aside)>`,
		`(?is)<[^>]*id="[^"]*(?:sidebar|ad-|advertisement|social|cookie|popup|modal|newsletter|comments)[^"]*"[^>]*>.*?</(?:div|section|aside)>`,
	}

	cleaned := html
	for _, pattern := range boilerplatePatterns {
		re := regexp.MustCompile(pattern)
		cleaned = re.ReplaceAllString(cleaned, "")
	}

	// Step 2: Try to find <article> or <main> tag first
	articleRe := regexp.MustCompile(`(?is)<(?:article|main)[^>]*>(.*?)</(?:article|main)>`)
	if m := articleRe.FindStringSubmatch(cleaned); len(m) > 1 {
		return htmlToText(m[1])
	}

	// Step 3: Find the div/section with the most <p> content
	// Split into candidate blocks and score them
	type block struct {
		content string
		score   int
	}

	// Extract content divs
	divRe := regexp.MustCompile(`(?is)<(?:div|section)[^>]*>(.*?)</(?:div|section)>`)
	matches := divRe.FindAllStringSubmatch(cleaned, -1)

	var blocks []block
	for _, m := range matches {
		if len(m) < 2 {
			continue
		}
		content := m[1]
		// Score based on paragraph text density
		pRe := regexp.MustCompile(`(?is)<p[^>]*>(.*?)</p>`)
		pMatches := pRe.FindAllStringSubmatch(content, -1)
		score := 0
		for _, pm := range pMatches {
			if len(pm) > 1 {
				text := regexp.MustCompile(`<[^>]+>`).ReplaceAllString(pm[1], "")
				text = strings.TrimSpace(text)
				if len(text) > 25 { // meaningful paragraph
					score += len(text)
				}
			}
		}
		if score > 100 {
			blocks = append(blocks, block{content: content, score: score})
		}
	}

	if len(blocks) == 0 {
		// No good blocks found, fall back to basic extraction
		return htmlToText(cleaned)
	}

	// Pick the highest-scoring block
	sort.Slice(blocks, func(i, j int) bool {
		return blocks[i].score > blocks[j].score
	})

	return htmlToText(blocks[0].content)
}
