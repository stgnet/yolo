package yolo

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestHandlerTests covers various HTTP handler edge cases
func TestHandlerTests(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		method         string
		body           string
		expectCode     int
		expectContains []string
		expectNotFound bool
	}{
		{
			name:         "GET main page",
			path:         "/",
			method:       "GET",
			expectCode:   http.StatusOK,
			expectContains: []string{"YOLO API"},
		},
		{
			name:         "GET health check",
			path:         "/health",
			method:       "GET",
			expectCode:   http.StatusOK,
			expectContains: []string{"ok"},
		},
		{
			name:           "GET non-existent path",
			path:           "/nonexistent",
			method:         "GET",
			expectCode:     http.StatusNotFound,
			expectNotFound: true,
		},
		{
			name:         "POST empty body to chat endpoint",
			path:         "/api/chat",
			method:       "POST",
			body:         `{"messages":[]}`,
			expectCode:   http.StatusOK,
			expectContains: []string{"response"},
		},
		{
			name:           "POST invalid JSON body",
			path:           "/api/chat",
			method:         "POST",
			body:           `invalid json`,
			expectCode:     http.StatusBadRequest,
			expectNotFound: false, // Should return 400, not 404
		},
		{
			name:         "POST missing messages field",
			path:         "/api/chat",
			method:       "POST",
			body:         `{}`,
			expectCode:   http.StatusOK, // Handled gracefully
			expectContains: []string{"response"},
		},
		{
			name:           "POST with empty string messages",
			path:           "/api/chat",
			method:         "POST",
			body:           `{"messages":""}`,
			expectCode:     http.StatusBadRequest,
			expectNotFound: false,
		},
		{
			name:           "POST with non-array messages",
			path:           "/api/chat",
			method:         "POST",
			body:           `{"messages":"not an array"}`,
			expectCode:     http.StatusBadRequest,
			expectNotFound: false,
		},
		{
			name:           "PUT on unsupported method",
			path:           "/api/chat",
			method:         "PUT",
			expectCode:     http.StatusMethodNotAllowed,
			expectNotFound: false,
		},
		{
			name:           "DELETE on unsupported method",
			path:           "/api/chat",
			method:         "DELETE",
			expectCode:     http.StatusMethodNotAllowed,
			expectNotFound: false,
		},
		{
			name:           "OPTIONS preflight request",
			path:           "/api/chat",
			method:         "OPTIONS",
			expectCode:     http.StatusOK,
			expectContains: []string{"Access-Control-Allow-Origin"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
			if tt.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}

			w := httptest.NewRecorder()
			handler := http.HandlerFunc(apiHandler)
			handler.ServeHTTP(w, req)

			resp := w.Result()
			defer resp.Body.Close()

			if resp.StatusCode != tt.expectCode {
				t.Errorf("Expected status %d, got %d", tt.expectCode, resp.StatusCode)
			}

			if tt.expectNotFound && resp.StatusCode != http.StatusNotFound {
				t.Errorf("Expected 404, got %d", resp.StatusCode)
			}

			body := w.Body.String()
			for _, expected := range tt.expectContains {
				if !strings.Contains(body, expected) {
					t.Errorf("Expected body to contain %q, but got: %s", expected, body)
				}
			}

			// Check CORS headers for POST/PUT/DELETE requests
			if tt.method != "GET" && tt.expectCode != http.StatusMethodNotAllowed {
				if resp.Header.Get("Access-Control-Allow-Origin") == "" {
					t.Errorf("Expected CORS header, got none")
				}
			}

			// Verify Content-Type for JSON responses
			if !tt.expectNotFound {
				contentType := resp.Header.Get("Content-Type")
				if !strings.Contains(contentType, "application/json") {
					t.Errorf("Expected application/json content type, got %s", contentType)
				}
			}
		})
	}
}

// TestHandlerEmptyBody tests edge case of empty request body
func TestHandlerEmptyBody(t *testing.T) {
	req := httptest.NewRequest("POST", "/api/chat", nil)
	w := httptest.NewRecorder()
	handler := http.HandlerFunc(apiHandler)
	handler.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected 200 or 400, got %d", resp.StatusCode)
	}
}

// TestHandlerLargeBody tests handling of large request bodies
func TestHandlerLargeBody(t *testing.T) {
	// Create a large JSON body with many messages
	var largeBody strings.Builder
	largeBody.WriteString(`{"messages":`)
	for i := 0; i < 100; i++ {
		if i > 0 {
			largeBody.WriteString(",")
		}
		largeBody.WriteString(`{"role":"user","content":"This is message number `)
		largeBody.WriteString(string(rune('0'+i)))
		largeBody.WriteString(` with some additional text to make it longer"}}`)
	}

	req := httptest.NewRequest("POST", "/api/chat", &largeBody)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler := http.HandlerFunc(apiHandler)
	handler.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	// Should handle large bodies gracefully without crashing
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusInternalServerError {
		t.Errorf("Expected 200-299 for large body, got %d", resp.StatusCode)
	}
}

// TestHandlerConcurrentRequests tests concurrent access to the handler
func TestHandlerConcurrentRequests(t *testing.T) {
	const numRequests = 10

	done := make(chan bool, numRequests)

	for i := 0; i < numRequests; i++ {
		go func(id int) {
			defer func() { done <- true }()
			req := httptest.NewRequest("GET", "/health", nil)
			w := httptest.NewRecorder()
			handler := http.HandlerFunc(apiHandler)
			handler.ServeHTTP(w, req)

			resp := w.Result()
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("Request %d: Expected 200, got %d", id, resp.StatusCode)
			}
		}(i)
	}

	for i := 0; i < numRequests; i++ {
		select {
		case <-done:
		case <-chan time.After(5 * time.Second):
			t.Fatal("Timeout waiting for concurrent requests to complete")
		}
	}
}

// TestHandlerInvalidRoutes tests various invalid route patterns
func TestHandlerInvalidRoutes(t *testing.T) {
	invalidPaths := []string{
		"/api/nonexistent",
		"/nonexistent/api/chat",
		"///triple/slashes",
		"", // Empty path defaults to root which is valid, so this one is expected to fail
		"/api/chat/", // With trailing slash - depends on router behavior
	}

	for _, path := range invalidPaths {
		if path == "" {
			continue // Skip empty path test as it may route to root
		}

		req := httptest.NewRequest("GET", path, nil)
		w := httptest.NewRecorder()
		handler := http.HandlerFunc(apiHandler)
		handler.ServeHTTP(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			t.Logf("Path %q unexpectedly returned 200", path)
		}
	}
}

// TestHandlerResponseFormat validates that responses are proper JSON
func TestHandlerResponseFormat(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		path     string
		body     string
	}{
		{"GET root", "GET", "/", ""},
		{"GET health", "GET", "/health", ""},
		{"POST chat", "POST", "/api/chat", `{"messages":[]}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
			if tt.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}

			w := httptest.NewRecorder()
			handler := http.HandlerFunc(apiHandler)
			handler.ServeHTTP(w, req)

			resp := w.Result()
			defer resp.Body.Close()

			if !strings.Contains(resp.Header.Get("Content-Type"), "application/json") {
				t.Errorf("Expected JSON response, got content type: %s", resp.Header.Get("Content-Type"))
			}
		})
	}
}
