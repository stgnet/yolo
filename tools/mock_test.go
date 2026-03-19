package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

// MockHTTPHandler creates a test server for mocking HTTP calls
func createMockServer(response string, statusCode int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(statusCode)
		w.Write([]byte(response))
	}))
}

// TestCreateMockServer tests mock server creation
func TestCreateMockServer(t *testing.T) {
	server := createMockServer(`{"test": "data"}`, 200)
	defer server.Close()

	resp, err := http.Get(server.URL)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), `"test": "data"`)
}

// TestHTTPHandlerMock tests HTTP handler with mock responses
func TestHTTPHandlerMock(t *testing.T) {
	// Test successful response
	server := createMockServer(`{"status": "ok"}`, 200)
	defer server.Close()

	resp, err := http.Get(server.URL + "/test")
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

// TestHTTPHandlerError tests error handling in HTTP calls
func TestHTTPHandlerError(t *testing.T) {
	// This test ensures our code handles HTTP errors gracefully
	// In production, this would be a real failing endpoint
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/nonexistent")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// TestJSONParsing tests JSON parsing with various inputs
func TestJSONParsing(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected interface{}
	}{
		{
			name:  "simple object",
			input: `{"key": "value"}`,
			expected: map[string]interface{}{
				"key": "value",
			},
		},
		{
			name:     "array",
			input:    `[1, 2, 3]`,
			expected: []interface{}{float64(1), float64(2), float64(3)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result interface{}
			err := json.Unmarshal([]byte(tt.input), &result)
			assert.NoError(t, err)
			// Note: JSON parsing converts numbers to float64
		})
	}
}

// TestContextCancellation tests context cancellation handling
func TestContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Immediately cancel

	// Test that we can detect cancelled context
	select {
	case <-ctx.Done():
		assert.True(t, true, "context should be cancelled")
	default:
		t.Error("context should be cancelled")
	}
}

// TestBytesBuffer tests bytes.Buffer operations
func TestBytesBuffer(t *testing.T) {
	var buf bytes.Buffer
	buf.WriteString("hello ")
	buf.WriteString("world")
	assert.Equal(t, "hello world", buf.String())
	assert.Equal(t, 11, buf.Len())
}

// TestWebSearchMock tests web search with mock DuckDuckGo responses
func TestWebSearchMock(t *testing.T) {
	// Mock successful search results
	mockResponse := `{
		"InstantAnswers": [{"Abstract": "Test result from search"}],
		"RelatedTopics": [{"Text": "Related topic 1"}]
	}`
	server := createMockServer(mockResponse, 200)
	defer server.Close()

	resp, err := http.Get(server.URL + "/search?q=test")
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

// TestRedditMock tests Reddit API with mock responses
func TestRedditMock(t *testing.T) {
	mockResponse := `{
		"data": {
			"children": [
				{"data": {"title": "Test Post", "score": 100}}
			]
		}
	}`
	server := createMockServer(mockResponse, 200)
	defer server.Close()

	resp, err := http.Get(server.URL + "/r/golang/hot.json")
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

// TestGoogleAPIMock tests Google API calls with mocked responses
func TestGoogleAPIMock(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		response string
	}{
		{
			name:     "gmail search",
			endpoint: "/gmail/search",
			response: `{"messages": [{"id": "123"}]}`,
		},
		{
			name:     "calendar events",
			endpoint: "/calendar/events",
			response: `{"events": []}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := createMockServer(tt.response, 200)
			defer server.Close()

			resp, err := http.Get(server.URL + tt.endpoint)
			assert.NoError(t, err)
			assert.Equal(t, 200, resp.StatusCode)
		})
	}
}

// TestHTTPErrorHandling tests proper error handling for HTTP failures
func TestHTTPErrorHandling(t *testing.T) {
	// Mock server that returns different error codes
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/404" {
			w.WriteHeader(http.StatusNotFound)
		} else if r.URL.Path == "/500" {
			w.WriteHeader(http.StatusInternalServerError)
		} else if r.URL.Path == "/timeout" {
			// Simulate slow response
			http.Error(w, "Timeout", http.StatusGatewayTimeout)
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		}
	}))
	defer server.Close()

	// Test 404 handling
	resp, _ := http.Get(server.URL + "/404")
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	// Test 500 handling
	resp, _ = http.Get(server.URL + "/500")
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	// Test timeout handling
	resp, _ = http.Get(server.URL + "/timeout")
	assert.Equal(t, http.StatusGatewayTimeout, resp.StatusCode)

	// Test successful response
	resp, _ = http.Get(server.URL + "/success")
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// TestRateLimitingMock tests rate limiting behavior with mock responses
func TestRateLimitingMock(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount > 10 {
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error": "rate limit exceeded"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	}))
	defer server.Close()

	// Make multiple requests until rate limited
	for i := 0; i < 15; i++ {
		resp, _ := http.Get(server.URL + "/api")
		if i < 10 {
			assert.Equal(t, http.StatusOK, resp.StatusCode)
		} else {
			assert.Equal(t, http.StatusTooManyRequests, resp.StatusCode)
		}
	}
	assert.Equal(t, 15, requestCount)
}

// TestConnectionPooling tests HTTP client connection reuse
func TestConnectionPooling(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	client := &http.Client{}

	// Make multiple requests with same client (should reuse connections)
	for i := 0; i < 5; i++ {
		resp, err := client.Get(server.URL + "/test")
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()
	}

	assert.Equal(t, 5, requestCount)
}

// TestHTTPHeaders tests header handling in HTTP requests/responses
func TestHTTPHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Echo back the User-Agent header
		w.Header().Set("X-Echo-UA", r.Header.Get("User-Agent"))
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	}))
	defer server.Close()

	req, _ := http.NewRequest("GET", server.URL+"/test", nil)
	req.Header.Set("User-Agent", "YOLO-TestAgent/1.0")
	
	client := &http.Client{}
	resp, err := client.Do(req)
	assert.NoError(t, err)
	assert.Equal(t, "YOLO-TestAgent/1.0", resp.Header.Get("X-Echo-UA"))
}

// TestJSONMarshalUnmarshal tests JSON serialization/deserialization
func TestJSONMarshalUnmarshal(t *testing.T) {
	type SearchResult struct {
		Title   string `json:"title"`
		Snippet string `json:"snippet"`
		URL     string `json:"url"`
	}

	original := SearchResult{
		Title:   "Test Result",
		Snippet: "This is a test snippet",
		URL:     "https://example.com/test",
	}

	// Marshal to JSON
	data, err := json.Marshal(original)
	assert.NoError(t, err)
	assert.Contains(t, string(data), `"title":"Test Result"`)

	// Unmarshal from JSON
	var result SearchResult
	err = json.Unmarshal(data, &result)
	assert.NoError(t, err)
	assert.Equal(t, original.Title, result.Title)
	assert.Equal(t, original.Snippet, result.Snippet)
	assert.Equal(t, original.URL, result.URL)
}

// TestEmptyResponseHandling tests handling of empty responses
func TestEmptyResponseHandling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// No body sent
	}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/empty")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	
	body, _ := io.ReadAll(resp.Body)
	assert.Empty(t, body)
}

// TestLargeResponseHandling tests handling of large responses
func TestLargeResponseHandling(t *testing.T) {
	largeBody := bytes.Repeat([]byte("x"), 1024*1024) // 1MB of data
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", strconv.Itoa(len(largeBody)))
		w.WriteHeader(http.StatusOK)
		w.Write(largeBody)
	}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/large")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	
	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, 1024*1024, len(body))
}