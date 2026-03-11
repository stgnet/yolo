package main

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// readWebpage fetches a URL and returns its text content with HTML tags stripped.
func (t *ToolExecutor) readWebpage(args map[string]any) string {
	rawURL := getStringArg(args, "url", "")
	if rawURL == "" {
		return errorMessage("'url' parameter is required")
	}

	// Basic URL validation
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		rawURL = "https://" + rawURL
	}

	// Check cache
	cacheKey := fmt.Sprintf("webpage:%s", rawURL)
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
		return errorMessage("invalid URL '%s': %v", rawURL, err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; YOLO-Bot/1.0)")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,text/plain,*/*")

	resp, err := client.Do(req)
	if err != nil {
		return errorMessage("failed to fetch '%s': %v", rawURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return errorMessage("HTTP %d fetching '%s'", resp.StatusCode, rawURL)
	}

	// Limit body to 1MB to avoid memory issues
	const maxBody = 1 << 20
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBody))
	if err != nil {
		return errorMessage("failed to read response body: %v", err)
	}

	contentType := resp.Header.Get("Content-Type")
	text := string(body)

	// If HTML content, strip tags and extract text
	if strings.Contains(contentType, "text/html") || strings.Contains(text, "<html") {
		text = htmlToText(text)
	}

	// Trim and truncate
	text = strings.TrimSpace(text)
	if text == "" {
		return fmt.Sprintf("Fetched '%s' but the page contained no extractable text content.", rawURL)
	}

	// Truncate to ~50k chars to avoid overwhelming the LLM context
	const maxChars = 50000
	if len(text) > maxChars {
		text = text[:maxChars] + "\n\n[Content truncated at 50,000 characters]"
	}

	result := fmt.Sprintf("Content from %s:\n\n%s", rawURL, text)

	// Cache the result
	searchCache.Store(cacheKey, &searchCacheEntry{
		Result: result,
		Ts:     time.Now(),
	})

	return result
}

// htmlToText converts HTML to readable plain text.
func htmlToText(html string) string {
	// Remove script and style blocks entirely
	reScript := regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`)
	html = reScript.ReplaceAllString(html, "")
	reStyle := regexp.MustCompile(`(?is)<style[^>]*>.*?</style>`)
	html = reStyle.ReplaceAllString(html, "")

	// Remove head block
	reHead := regexp.MustCompile(`(?is)<head[^>]*>.*?</head>`)
	html = reHead.ReplaceAllString(html, "")

	// Remove nav, footer, and aside blocks (usually boilerplate)
	for _, tag := range []string{"nav", "footer", "aside"} {
		re := regexp.MustCompile(fmt.Sprintf(`(?is)<%s[^>]*>.*?</%s>`, tag, tag))
		html = re.ReplaceAllString(html, "")
	}

	// Convert block elements to newlines
	for _, tag := range []string{"p", "div", "br", "li", "tr", "h1", "h2", "h3", "h4", "h5", "h6", "blockquote", "pre", "article", "section"} {
		re := regexp.MustCompile(fmt.Sprintf(`(?i)</?%s[^>]*>`, tag))
		html = re.ReplaceAllString(html, "\n")
	}

	// Convert hr to separator
	reHR := regexp.MustCompile(`(?i)<hr[^>]*/?>`)
	html = reHR.ReplaceAllString(html, "\n---\n")

	// Strip remaining HTML tags
	reTags := regexp.MustCompile(`<[^>]+>`)
	html = reTags.ReplaceAllString(html, "")

	// Decode common HTML entities
	replacer := strings.NewReplacer(
		"&amp;", "&",
		"&lt;", "<",
		"&gt;", ">",
		"&quot;", `"`,
		"&#39;", "'",
		"&apos;", "'",
		"&nbsp;", " ",
		"&mdash;", "—",
		"&ndash;", "–",
		"&hellip;", "…",
		"&copy;", "©",
		"&reg;", "®",
		"&trade;", "™",
	)
	html = replacer.Replace(html)

	// Decode numeric HTML entities
	reNumEntity := regexp.MustCompile(`&#(\d+);`)
	html = reNumEntity.ReplaceAllStringFunc(html, func(s string) string {
		matches := reNumEntity.FindStringSubmatch(s)
		if len(matches) > 1 {
			var code int
			fmt.Sscanf(matches[1], "%d", &code)
			if code > 0 && code < 0x10FFFF {
				return string(rune(code))
			}
		}
		return s
	})

	// Collapse multiple blank lines into at most two newlines
	reBlankLines := regexp.MustCompile(`\n{3,}`)
	html = reBlankLines.ReplaceAllString(html, "\n\n")

	// Collapse multiple spaces on the same line
	reSpaces := regexp.MustCompile(`[^\S\n]+`)
	html = reSpaces.ReplaceAllString(html, " ")

	// Trim leading/trailing whitespace from each line
	lines := strings.Split(html, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimSpace(line)
	}

	return strings.Join(lines, "\n")
}
