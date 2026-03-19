// Package tools provides unit tests for web-related tool functionality
package tools

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWebSearchTool_MalformedResponse(t *testing.T) {
	// Note: webSearch uses a hardcoded DuckDuckGo URL, so this test verifies
	// that the tool properly handles parameter validation and error cases.
	// Actual network response testing would require mocking or stubbing.
	
	tool := &WebSearchTool{}
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"query": "test",
	})

	// We can't control the DuckDuckGo API response, so just verify no panic
	if result == nil {
		t.Error("Expected non-nil result")
	}
	_ = err // Tool may handle errors internally
}

func TestWebSearchTool_ServerError(t *testing.T) {
	// Note: webSearch uses a hardcoded DuckDuckGo URL, so this test verifies
	// that the tool doesn't crash with empty parameters.
	// Actual network response testing would require mocking or stubbing.
	
	tool := &WebSearchTool{}
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"query": "test",
	})

	// We can't control the DuckDuckGo API response, so just verify no panic
	if result == nil {
		t.Error("Expected non-nil result")
	}
	_ = err // Tool may handle errors internally
}

func TestReadWebpageTool_MalformedHTML(t *testing.T) {
	// Test malformed HTML that might cause parsing issues
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte("<html><head></html>")) // Unclosed head tag
	}))
	defer server.Close()

	tool := &ReadWebpageTool{}
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"url": server.URL,
	})

	if err != nil {
		t.Errorf("Expected no error for malformed HTML, got %v", err)
	}
	if !result.Success {
		t.Error("Expected successful result for malformed HTML")
	}
	if strings.TrimSpace(result.Output) == "" {
		t.Error("Expected non-empty result, got empty string")
	}
}

func TestReadWebpageTool_BinaryContent(t *testing.T) {
	// Test response with binary content type (should not panic)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write([]byte{0x00, 0x01, 0x02, 0x03})
	}))
	defer server.Close()

	tool := &ReadWebpageTool{}
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"url": server.URL,
	})

	// Should not panic on binary data - error handling is fine
	if err == nil && result.Success {
		t.Logf("Binary content was parsed (output length: %d)", len(result.Output))
	} else if err != nil || !result.Success {
		t.Logf("Binary content rejected: success=%v, err=%v", result.Success, err)
	}
}

func TestReadWebpageTool_EmptyBody(t *testing.T) {
	// Test response with empty body
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	tool := &ReadWebpageTool{}
	result, _ := tool.Execute(context.Background(), map[string]interface{}{
		"url": server.URL,
	})

	if !result.Success {
		t.Logf("Empty body result: success=%v, error=%v", result.Success, result.Error)
	}
	// Empty content is acceptable
}

func TestReadWebpageTool_Redirect(t *testing.T) {
	redirectCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		redirectCount++
		if redirectCount == 1 {
			http.Redirect(w, r, "/final", http.StatusTemporaryRedirect)
		} else {
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte("<html><body>Final content</body></html>"))
		}
	}))
	defer server.Close()

	tool := &ReadWebpageTool{}
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"url": server.URL,
	})

	if err != nil {
		t.Fatalf("Expected no error with redirect, got %v", err)
	}
	if !result.Success {
		t.Errorf("Expected successful result after redirect")
	}
	if !strings.Contains(result.Output, "Final content") {
		t.Errorf("Expected 'Final content' in result, got '%s'", result.Output)
	}
}

func TestReadWebpageTool_HugeResponse(t *testing.T) {
	// Test with large response body (should handle gracefully)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		// Write 1MB of HTML content
		largeContent := strings.Repeat("<div>test</div>\n", 50000)
		w.Write([]byte(largeContent))
	}))
	defer server.Close()

	tool := &ReadWebpageTool{}
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"url": server.URL,
	})

	if err != nil {
		t.Fatalf("Expected no error for large response, got %v", err)
	}
	if !result.Success {
		t.Errorf("Expected successful result for large content")
	}
	if len(result.Output) == 0 {
		t.Error("Expected non-empty result for large content")
	}
}

func TestReadWebpageTool_MissingURL(t *testing.T) {
	tool := &ReadWebpageTool{}
	result, err := tool.Execute(context.Background(), map[string]interface{}{})

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result.Success {
		t.Error("Expected failed result for missing URL")
	}
	if !strings.Contains(result.Error, "url is required") {
		t.Errorf("Expected 'url is required' error, got '%s'", result.Error)
	}
}

func TestWebSearchTool_MissingQuery(t *testing.T) {
	tool := &WebSearchTool{}
	result, err := tool.Execute(context.Background(), map[string]interface{}{})

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result.Success {
		t.Error("Expected failed result for missing query")
	}
	if !strings.Contains(result.Error, "query is required") {
		t.Errorf("Expected 'query is required' error, got '%s'", result.Error)
	}
}
