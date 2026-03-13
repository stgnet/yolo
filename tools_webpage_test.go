package main

import (
	"strings"
	"testing"
)

func TestReadWebpageToolDefinition(t *testing.T) {
	found := false
	for _, tool := range ollamaTools {
		if tool.Function.Name == "read_webpage" {
			found = true
			if tool.Function.Description == "" {
				t.Error("read_webpage tool missing description")
			}
			if len(tool.Function.Parameters.Properties) != 1 {
				t.Errorf("read_webpage should have 1 parameter, got %d", len(tool.Function.Parameters.Properties))
			}
			if _, ok := tool.Function.Parameters.Properties["url"]; !ok {
				t.Error("read_webpage missing required parameter: url")
			}
			break
		}
	}
	if !found {
		t.Error("read_webpage tool not found in ollamaTools")
	}
}

func TestReadWebpageMissingURL(t *testing.T) {
	executor := NewToolExecutor("/tmp", nil)
	result := executor.readWebpage(map[string]any{})

	if !strings.Contains(result, "Error:") && !strings.Contains(result, "required") {
		t.Errorf("Expected error for missing URL, got: %s", result[:min(200, len(result))])
	}
}

func TestReadWebpageEmptyURL(t *testing.T) {
	executor := NewToolExecutor("/tmp", nil)
	result := executor.readWebpage(map[string]any{"url": ""})

	if !strings.Contains(result, "Error:") && !strings.Contains(result, "required") {
		t.Errorf("Expected error for empty URL, got: %s", result[:min(200, len(result))])
	}
}

func TestReadWebpageURLWithoutScheme(t *testing.T) {
	executor := NewToolExecutor("/tmp", nil)
	// This will fail to fetch but should add https:// prefix
	result := executor.readWebpage(map[string]any{"url": "example.com"})

	// Should not have URL error, will have fetch error instead
	if strings.Contains(result, "invalid URL") {
		t.Errorf("Expected URL to be prefixed with https://, got error: %s", result[:min(200, len(result))])
	}
}

func TestHTMLToTextStripScript(t *testing.T) {
	html := `<html><body><script>malicious()</script><p>Hello World</p></body></html>`
	text := htmlToText(html)

	if strings.Contains(text, "script") || strings.Contains(text, "malicious") {
		t.Errorf("Expected script tags to be removed, got: %s", text)
	}
	if !strings.Contains(text, "Hello World") {
		t.Errorf("Expected 'Hello World' in output, got: %s", text)
	}
}

func TestHTMLToTextStripStyle(t *testing.T) {
	html := `<html><body><style>body { color: red; }</style><p>Hello</p></body></html>`
	text := htmlToText(html)

	if strings.Contains(text, "style") || strings.Contains(text, "color") {
		t.Errorf("Expected style tags to be removed, got: %s", text)
	}
	if !strings.Contains(text, "Hello") {
		t.Errorf("Expected 'Hello' in output, got: %s", text)
	}
}

func TestHTMLToTextRemoveTags(t *testing.T) {
	html := `<div><p>Hello <strong>World</strong></p></div>`
	text := htmlToText(html)

	if strings.Contains(text, "<") || strings.Contains(text, ">") {
		t.Errorf("Expected all HTML tags to be removed, got: %s", text)
	}
	if !strings.Contains(text, "Hello") || !strings.Contains(text, "World") {
		t.Errorf("Expected text content preserved, got: %s", text)
	}
}

func TestHTMLToTextDecodeEntities(t *testing.T) {
	html := `Hello &amp; World &lt;test&gt;`
	text := htmlToText(html)

	if !strings.Contains(text, "&") || !strings.Contains(text, "<") || !strings.Contains(text, ">") {
		t.Errorf("Expected HTML entities to be decoded, got: %s", text)
	}
}

func TestHTMLToTextCollapseWhitespace(t *testing.T) {
	html := `<p>Hello    World</p><p>Multiple     spaces</p>`
	text := htmlToText(html)

	// Should collapse multiple spaces but preserve single spaces
	if strings.Contains(text, "    ") {
		t.Errorf("Expected multiple spaces to be collapsed, got: %s", text)
	}
}

func TestHTMLToTextRemoveHead(t *testing.T) {
	html := `<html><head><title>Test</title></head><body>Hello</body></html>`
	text := htmlToText(html)

	if strings.Contains(text, "head") || strings.Contains(text, "title") {
		t.Errorf("Expected head section to be removed, got: %s", text)
	}
	if !strings.Contains(text, "Hello") {
		t.Errorf("Expected body content preserved, got: %s", text)
	}
}
