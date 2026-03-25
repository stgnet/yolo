package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ─── extractArticleContent tests ────────────────────────────────────

func TestExtractArticleContent_ArticleTag(t *testing.T) {
	html := `<html><body>
		<nav>Navigation here</nav>
		<article>
			<h1>Article Title</h1>
			<p>This is the main article content with enough text to be meaningful.</p>
			<p>Another paragraph with more content for the article body.</p>
		</article>
		<footer>Footer stuff</footer>
	</body></html>`

	result := extractArticleContent(html)
	assert.Contains(t, result, "Article Title")
	assert.Contains(t, result, "main article content")
	assert.NotContains(t, result, "Navigation here")
	assert.NotContains(t, result, "Footer stuff")
}

func TestExtractArticleContent_MainTag(t *testing.T) {
	html := `<html><body>
		<nav>Nav</nav>
		<main>
			<p>Main content area with substantial text for testing purposes.</p>
		</main>
		<aside>Sidebar</aside>
	</body></html>`

	result := extractArticleContent(html)
	assert.Contains(t, result, "Main content area")
	assert.NotContains(t, result, "Nav")
	assert.NotContains(t, result, "Sidebar")
}

func TestExtractArticleContent_ScoreBasedExtraction(t *testing.T) {
	html := `<html><body>
		<div class="sidebar">
			<p>Short sidebar text</p>
		</div>
		<div class="content">
			<p>This is a substantial paragraph with enough text to score well in the content detection algorithm that looks for text density.</p>
			<p>Another significant paragraph that adds to the content score and helps identify this as the main content block of the page.</p>
			<p>A third paragraph provides even more confidence that this div contains the primary article content.</p>
		</div>
		<div class="ads">Buy stuff</div>
	</body></html>`

	result := extractArticleContent(html)
	assert.Contains(t, result, "substantial paragraph")
	assert.Contains(t, result, "content score")
}

func TestExtractArticleContent_RemovesBoilerplate(t *testing.T) {
	html := `<html>
		<head><title>Test</title></head>
		<body>
			<script>alert('js')</script>
			<style>.foo { color: red; }</style>
			<nav>Menu items</nav>
			<p>Real content here with enough text.</p>
			<footer>Copyright 2025</footer>
			<noscript>Enable JS</noscript>
		</body>
	</html>`

	result := extractArticleContent(html)
	assert.NotContains(t, result, "alert")
	assert.NotContains(t, result, "color: red")
	assert.NotContains(t, result, "Menu items")
	assert.NotContains(t, result, "Copyright")
	assert.NotContains(t, result, "Enable JS")
}

// ─── extractTitle tests ─────────────────────────────────────────────

func TestExtractTitle(t *testing.T) {
	assert.Equal(t, "Hello World", extractTitle(`<html><head><title>Hello World</title></head></html>`))
	assert.Equal(t, "Test & Page", extractTitle(`<title>Test &amp; Page</title>`))
	assert.Equal(t, "", extractTitle(`<html><body>No title here</body></html>`))
}

// ─── readWebpageReadable tests ──────────────────────────────────────

func TestReadWebpageReadable_MissingURL(t *testing.T) {
	te := &ToolExecutor{baseDir: "/tmp", agent: &YoloAgent{}}
	result := te.readWebpageReadable(map[string]any{})
	assert.Contains(t, result, "Error:")
}

// ─── screenshot tests ───────────────────────────────────────────────

func TestScreenshot_MissingURL(t *testing.T) {
	te := &ToolExecutor{baseDir: "/tmp", agent: &YoloAgent{}}
	result := te.screenshot(map[string]any{})
	assert.Contains(t, result, "Error:")
}

func TestScreenshot_NoToolAvailable(t *testing.T) {
	te := &ToolExecutor{baseDir: "/tmp", agent: &YoloAgent{}}
	// This will fail since we likely don't have playwright or chromium in test env
	result := te.screenshot(map[string]any{"url": "https://example.com"})
	// Should get an error about no screenshot tool, not a panic
	assert.Contains(t, result, "Error:")
}

// ─── searxngSearch tests ────────────────────────────────────────────

func TestSearxngSearch_NoURL(t *testing.T) {
	te := &ToolExecutor{baseDir: "/tmp", agent: &YoloAgent{}}
	// Without SEARXNG_URL env, should return empty
	result := te.searxngSearch("test query", 5)
	assert.Equal(t, "", result)
}

// ─── webSearchEnhanced tests ────────────────────────────────────────

func TestWebSearchEnhanced_MissingQuery(t *testing.T) {
	te := &ToolExecutor{baseDir: "/tmp", agent: &YoloAgent{}}
	result := te.webSearchEnhanced(map[string]any{})
	assert.Contains(t, result, "Error:")
}

func TestWebSearchEnhanced_ExplicitSearxng(t *testing.T) {
	te := &ToolExecutor{baseDir: "/tmp", agent: &YoloAgent{}}
	// Without SEARXNG_URL, should error
	result := te.webSearchEnhanced(map[string]any{"query": "test", "engine": "searxng"})
	assert.Contains(t, result, "Error:")
}

// ─── HTML extraction integration tests ──────────────────────────────

func TestReadabilityVsFullExtraction(t *testing.T) {
	html := `<html>
		<head><title>Test Article</title></head>
		<body>
			<nav><a href="/">Home</a> | <a href="/about">About</a></nav>
			<article>
				<h1>Test Article Title</h1>
				<p>This is the first paragraph of the article with meaningful content that should be preserved by the readability extraction algorithm.</p>
				<p>Second paragraph continues the article with additional information and context that adds value to the reader.</p>
			</article>
			<aside>
				<h3>Related Posts</h3>
				<ul><li>Post 1</li><li>Post 2</li></ul>
			</aside>
			<footer><p>Copyright 2025</p></footer>
		</body>
	</html>`

	// Article mode should extract only the main content
	article := extractArticleContent(html)
	assert.Contains(t, article, "first paragraph")
	assert.Contains(t, article, "Second paragraph")
	assert.NotContains(t, article, "Home")
	assert.NotContains(t, article, "Related Posts")

	// Full mode should have the body content (nav/footer stripped by htmlToText)
	full := htmlToText(html)
	assert.Contains(t, full, "first paragraph")
	assert.Contains(t, full, "Test Article Title")
}

func TestExtractArticleContent_FallbackToFullText(t *testing.T) {
	// A page with no article/main tag and no high-scoring divs
	html := `<html><body>
		<p>Simple page with just paragraphs but not enough density in any single div to trigger scoring.</p>
	</body></html>`

	result := extractArticleContent(html)
	assert.Contains(t, result, "Simple page")
}

func TestExtractArticleContent_ClassBasedRemoval(t *testing.T) {
	html := `<html><body>
		<div class="advertisement-banner">Buy this product!</div>
		<div class="main-content">
			<p>Real article content with enough text to be identified as the main content block by the scoring algorithm.</p>
			<p>More content to increase the score above the threshold for this block to be selected.</p>
		</div>
		<div class="sidebar-widget">Widget content</div>
		<div class="cookie-notice">Accept cookies</div>
	</body></html>`

	result := extractArticleContent(html)
	assert.Contains(t, result, "Real article content")
	assert.NotContains(t, result, "Buy this product")
	assert.NotContains(t, result, "Accept cookies")
}
