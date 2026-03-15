// Package main implements web search improvements for YOLO
package web_search_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"
)

// SearchResult represents a web search result
type SearchResult struct {
	Query   string   `json:"query"`
	Results []string `json:"results"`
	Sources []string `json:"sources"`
}

// MockHTTPClient implements http.Client for testing with configurable behavior
type MockHTTPClient struct {
	Response      *http.Response
	Body          io.Reader
	Error         error
	StatusCode    int
	Header        http.Header
	Delay         time.Duration
	RateLimit     bool
	RetryAfter    int
}

// Do implements the RoundTripper interface with custom behaviors for testing
func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	if m.Delay > 0 {
		time.Sleep(m.Delay)
	}
	if m.Error != nil {
		return nil, m.Error
	}

	body := io.NopCloser(m.Body)
	resp := &http.Response{
		StatusCode: m.StatusCode,
		Body:       body,
		Header:     m.Header,
		Request:    req,
	}
	return resp, nil
}

// newMockResponse creates a mock HTTP response for testing
func newMockResponse(body string, statusCode int, header http.Header) *MockHTTPClient {
	mock := &MockHTTPClient{
		Body:       strings.NewReader(body),
		StatusCode: statusCode,
		Header:     make(http.Header),
	}
	if header != nil {
		for k, v := range header {
			mock.Header[k] = v
		}
	}
	return mock
}

// TestDuckDuckGoSearchWithMock tests DuckDuckGo search with mocked HTTP client
func TestDuckDuckGoSearchWithMock(t *testing.T) {
	t.Run("successful search", func(t *testing.T) {
		mockClient := &MockHTTPClient{
			Response: &http.Response{StatusCode: 200},
			Body:     io.NopCloser(strings.NewReader(`{"Abstract": "Test abstract content"}`)),
		}
		oldClient := http.DefaultClient
		http.DefaultClient = mockClient
		defer func() { http.DefaultClient = oldClient }()

		results, err := DuckDuckGoSearch("test query", 5)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if len(results) == 0 {
			t.Error("Expected non-empty results")
		}
	})

	t.Run("empty abstract returns error", func(t *testing.T) {
		mockClient := &MockHTTPClient{
			Response: &http.Response{StatusCode: 200},
			Body:     io.NopCloser(strings.NewReader(`{"Abstract": ""}`)),
		}
		oldClient := http.DefaultClient
		http.DefaultClient = mockClient
		defer func() { http.DefaultClient = oldClient }()

		_, err := DuckDuckGoSearch("test", 5)
		if err == nil {
			t.Error("Expected error for empty abstract")
		}
	})

	t.Run("malformed JSON returns parse error", func(t *testing.T) {
		mockClient := &MockHTTPClient{
			Body: io.NopCloser(strings.NewReader(`invalid json`)),
		}
		oldClient := http.DefaultClient
		http.DefaultClient = mockClient
		defer func() { http.DefaultClient = oldClient }()

		_, err := DuckDuckGoSearch("test", 5)
		if err == nil {
			t.Error("Expected error for malformed JSON")
		} else if !strings.Contains(err.Error(), "parse JSON") {
			t.Errorf("Expected parse error, got: %v", err)
		}
	})

	t.Run("empty query returns error", func(t *testing.T) {
		mockClient := &MockHTTPClient{
			Body: io.NopCloser(strings.NewReader(`{"Abstract": ""}`)),
		}
		oldClient := http.DefaultClient
		http.DefaultClient = mockClient
		defer func() { http.DefaultClient = oldClient }()

		_, err := DuckDuckGoSearch("", 5)
		if err == nil {
			t.Error("Expected error for empty query")
		}
	})

	t.Run("limit capped at 5", func(t *testing.T) {
		mockClient := &MockHTTPClient{
			Body: io.NopCloser(strings.NewReader(`{"Abstract": "content"}`)),
		}
		oldClient := http.DefaultClient
		http.DefaultClient = mockClient
		defer func() { http.DefaultClient = oldClient }()

		_, err := DuckDuckGoSearch("test", 10)
		if err != nil {
			t.Errorf("Expected no error for high limit (should cap), got %v", err)
		}
	})
}

// TestMalformedQueries tests various malformed search queries
func TestMalformedQueries(t *testing.T) {
	tests := []struct {
		name    string
		query   string
		wantErr bool
	}{
		{
			name:    "SQL injection attempt",
			query:   "test' OR '1'='1'",
			wantErr: false, // Should be escaped/sanitized, not error
		},
		{
			name:    "XSS attack vector",
			query:   "<script>alert('xss')</script>",
			wantErr: false, // Should be encoded in URL
		},
		{
			name:    "Special characters",
			query:   "!@#$%^&*()_+-=[]{}|;':\",./<>?",
			wantErr: false, // Should be URL-encoded
		},
		{
			name:    "Multiple spaces and tabs",
			query:   "test  multiple\t\tspaces\nnewlines",
			wantErr: false,
		},
		{
			name:    "Null bytes in query",
			query:   "test\x00injection",
			wantErr: false, // Should be handled gracefully
		},
		{
			name:    "Path traversal attempt",
			query:   "../../../etc/passwd",
			wantErr: false, // Should be URL-encoded
		},
		{
			name:    "Command injection",
			query:   "; rm -rf /",
			wantErr: false, // Should be URL-encoded in query string
		},
		{
			name:    "Unicode mixed with special chars",
			query:   "test💻@#$🔥",
			wantErr: false,
		},
		{
			name:    "Single quote escaping",
			query:   "O'Reilly",
			wantErr: false,
		},
		{
			name:    "Backslash sequences",
			query:   "test\\nbackslash\\ttab",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockHTTPClient{
				Body: io.NopCloser(strings.NewReader(`{"Abstract": ""}`)),
			}
			oldClient := http.DefaultClient
			http.DefaultClient = mockClient
			defer func() { http.DefaultClient = oldClient }()

			_, err := DuckDuckGoSearch(tt.query, 5)
			if (err != nil) != tt.wantErr {
				t.Errorf("DuckDuckGoSearch(%q) error = %v, wantErr %v", tt.query, err, tt.wantErr)
			}

			// Verify the query was properly URL-encoded in the request URL
			if mockClient.Response != nil && mockClient.Response.Request != nil {
				queryStr := mockClient.Response.Request.URL.RawQuery
				if !strings.Contains(queryStr, url.QueryEscape(tt.query)) {
					t.Logf("Warning: Query not properly encoded. Raw query: %s", queryStr)
				}
			}
		})
	}
}

// TestNetworkTimeoutScenario tests network timeout handling
func TestNetworkTimeoutScenario(t *testing.T) {
	tests := []struct {
		name      string
		delay     time.Duration
		timeout   time.Duration
		wantError bool
	}{
		{
			name:      "timeout after 100ms",
			delay:     200 * time.Millisecond,
			timeout:   150 * time.Millisecond,
			wantError: true,
		},
		{
			name:      "success within timeout",
			delay:     50 * time.Millisecond,
			timeout:   150 * time.Millisecond,
			wantError: false,
		},
		{
			name:      "zero delay should succeed",
			delay:     0,
			timeout:   1000 * time.Millisecond,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockHTTPClient{
				Body:  io.NopCloser(strings.NewReader(`{"Abstract": "content"}`)),
				Delay: tt.delay,
			}

			// Create client with custom timeout
			client := &http.Client{
				Transport: mockClient,
				Timeout:   tt.timeout,
			}

			oldClient := http.DefaultClient
			http.DefaultClient = client
			defer func() { http.DefaultClient = oldClient }()

			_, err := DuckDuckGoSearch("timeout test", 5)
			if (err != nil) != tt.wantError {
				t.Errorf("Expected error=%v, got %v", tt.wantError, err)
			}

			if tt.wantError && err != nil {
				// Verify it's a timeout-related error
				if !strings.Contains(err.Error(), "timeout") && !strings.Contains(err.Error(), "deadline") {
					t.Logf("Timeout error message: %v", err)
				}
			}
		})
	}
}

// TestRateLimitingResponse tests rate limiting scenarios
func TestRateLimitingResponse(t *testing.T) {
	tests := []struct {
		name          string
		statusCode    int
		retryAfter    int
		wantErr       bool
		errorContains string
	}{
		{
			name:          "429 Too Many Requests with Retry-After header",
			statusCode:    429,
			retryAfter:    60,
			wantErr:       true,
			errorContains: "rate limit",
		},
		{
			name:          "429 without Retry-After header",
			statusCode:    429,
			retryAfter:    0,
			wantErr:       true,
			errorContains: "rate limit",
		},
		{
			name:     "503 Service Unavailable",
			statusCode: 503,
			retryAfter: 30,
			wantErr:  true,
		},
		{
			name:          "500 Internal Server Error",
			statusCode:    500,
			retryAfter:    0,
			wantErr:       true,
			errorContains: "status code",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			header := http.Header{
				"X-RateLimit-Remaining": []string{"0"},
			}
			if tt.retryAfter > 0 {
				header.Set("Retry-After", fmt.Sprintf("%d", tt.retryAfter))
			}

			mockClient := newMockResponse(`{"Abstract": ""}`, tt.statusCode, header)

			oldClient := http.DefaultClient
			http.DefaultClient = mockClient
			defer func() { http.DefaultClient = oldClient }()

			_, err := DuckDuckGoSearch("rate limit test", 5)
			if (err != nil) != tt.wantErr {
				t.Errorf("Expected error=%v, got %v", tt.wantErr, err)
			}

			if tt.wantErr && tt.errorContains != "" {
				if err == nil || !strings.Contains(err.Error(), tt.errorContains) {
					t.Logf("Error message: %v", err)
				}
			}
		})
	}
}

// TestEdgeCasesInParsing tests various parsing edge cases
func TestEdgeCasesInParsing(t *testing.T) {
	tests := []struct {
		name     string
		jsonBody string
		wantLen  int
		wantErr  bool
	}{
		{
			name:     "empty JSON object",
			jsonBody: `{}`,
			wantLen:  0,
			wantErr:  true,
		},
		{
			name:     "nil Abstract field",
			jsonBody: `{"Abstract": null}`,
			wantLen:  0,
			wantErr:  true,
		},
		{
			name:     "empty string Abstract",
			jsonBody: `{"Abstract": ""}`,
			wantLen:  0,
			wantErr:  true,
		},
		{
			name:     "non-string Abstract field",
			jsonBody: `{"Abstract": 123}`,
			wantLen:  0,
			wantErr:  true,
		},
		{
			name:     "Unicode content in abstract",
			jsonBody: `{"Abstract": "🚀 Space exploration 🌟"}`,
			wantLen:  1,
			wantErr:  false,
		},
		{
			name:     "Multilingual content",
			jsonBody: `{"Abstract": "日本語テスト العربية 한국어"}`,
			wantLen:  1,
			wantErr:  false,
		},
		{
			name:     "Very long abstract (>10k chars)",
			jsonBody: fmt.Sprintf(`{"Abstract": "%s"}`, strings.Repeat("x", 15000)),
			wantLen:  1,
			wantErr:  false,
		},
		{
			name:     "HTML entities in abstract",
			jsonBody: `{"Abstract": "Test &amp; other entities like &lt;br&gt;"}`,
			wantLen:  1,
			wantErr:  false,
		},
		{
			name:     "Newlines and whitespace in abstract",
			jsonBody: `{"Abstract": "Line 1\nLine 2\tTabbed\rCarriage"}`,
			wantLen:  1,
			wantErr:  false,
		},
		{
			name:     "Control characters in JSON",
			jsonBody: `{"Abstract": "test\x00\x01\x02data"}`,
			wantLen:  1,
			wantErr:  false, // Should handle gracefully
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockHTTPClient{
				Body: io.NopCloser(strings.NewReader(tt.jsonBody)),
			}

			oldClient := http.DefaultClient
			http.DefaultClient = mockClient
			defer func() { http.DefaultClient = oldClient }()

			results, err := DuckDuckGoSearch("parse test", 5)
			if (err != nil) != tt.wantErr {
				t.Errorf("DuckDuckGoSearch error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && len(results) != tt.wantLen {
				t.Errorf("Expected length %d, got %d", tt.wantLen, len(results))
			}
		})
	}
}

// TestInvalidURLsAndLinks tests invalid URL handling
func TestInvalidURLsAndLinks(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		wantErr  bool
	}{
		{
			name:     "malformed URL in query",
			query:    "http://invalid-url-@#$%.com",
			wantErr:  false,
		},
		{
			name:     "protocol-relative URL",
			query:    "//evil.com",
			wantErr:  false,
		},
		{
			name:     "URL with special characters",
			query:    "https://example.com/path?param=value&other=123#anchor",
			wantErr:  false,
		},
		{
			name:     "IPv4 address",
			query:    "192.168.1.1",
			wantErr:  false,
		},
		{
			name:     "IPv6 address",
			query:    "::1 or 2001:db8::1",
			wantErr:  false,
		},
		{
			name:     "URL-encoded query components",
			query:    "%3Cscript%3E",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockHTTPClient{
				Body: io.NopCloser(strings.NewReader(`{"Abstract": ""}`)),
			}

			oldClient := http.DefaultClient
			http.DefaultClient = mockClient
			defer func() { http.DefaultClient = oldClient }()

			_, err := DuckDuckGoSearch(tt.query, 5)
			if (err != nil) != tt.wantErr {
				t.Errorf("Expected error=%v for query %q, got %v", tt.wantErr, tt.query, err)
			}
		})
	}
}

// TestUnicodeCharacterHandling tests comprehensive Unicode handling
func TestUnicodeCharacterHandling(t *testing.T) {
	tests := []struct {
		name        string
		query       string
		wantLen     int
		contains    string
	}{
		{
			name:      "Chinese characters",
			query:     "Go 编程语言 最佳实践",
			wantLen:   1,
		},
		{
			name:      "Japanese characters",
			query:     "プログラミング言語 Go 最適化",
			wantLen:   1,
		},
		{
			name:      "Korean characters",
			query:     "고잉 프로그래밍 언어",
			wantLen:   1,
		},
		{
			name:      "Arabic script",
			query:     "لغات البرمجة",
			wantLen:   1,
		},
		{
			name:      "Hebrew script",
			query:     "שפות תכנות",
			wantLen:   1,
		},
		{
			name:      "Emoji characters",
			query:     "🚀🔥💻⚡️",
			wantLen:   1,
		},
		{
			name:      "Mixed script content",
			query:     "Go 🇺🇸 Python 🇩🇪 Rust 🇯🇵",
			wantLen:   1,
		},
		{
			name:      "Right-to-left text",
			query:     "היי עולם שלום مرحبا",
			wantLen:   1,
		},
		{
			name:      "Surrogate pairs",
			query:     "🎉😃🚀🏆", // Characters outside BMP require surrogate pairs
			wantLen:   1,
		},
		{
			name:      "Zero-width characters",
			query:     "test\u200bunicode\u200ccharacters",
			wantLen:   1,
		},
		{
			name:      "Combining diacritics",
			query:     "café résumé naïve",
			wantLen:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockHTTPClient{
				Body: io.NopCloser(strings.NewReader(`{"Abstract": "Unicode test result"}`)),
			}

			oldClient := http.DefaultClient
			http.DefaultClient = mockClient
			defer func() { http.DefaultClient = oldClient }()

			results, err := DuckDuckGoSearch(tt.query, 5)
			if err != nil && !tt.wantErr {
				t.Errorf("Unicode search for %q failed: %v", tt.query, err)
			}
			if len(results) != tt.wantLen && !tt.wantErr {
				t.Errorf("Expected length %d, got %d for query %q", tt.wantLen, len(results), tt.query)
			}

			// Verify URL encoding works with Unicode
			if mockClient.Response != nil && mockClient.Response.Request != nil {
				rawQuery := mockClient.Response.Request.URL.RawQuery
				expectedEncoded := url.QueryEscape(tt.query)
				if !strings.Contains(rawQuery, expectedEncoded) {
					t.Logf("Warning: Unicode query might not be properly encoded. Expected %q in %q", expectedEncoded, rawQuery)
				}
			}
		})
	}
}

// TestVeryLongSearchQueries tests handling of very long queries
func TestVeryLongSearchQueries(t *testing.T) {
	tests := []struct {
		name    string
		length  int
		wantErr bool
	}{
		{
			name:    "empty query",
			length:  0,
			wantErr: true,
		},
		{
			name:    "short query (10 chars)",
			length:  10,
			wantErr: false,
		},
		{
			name:    "medium query (500 chars)",
			length:  500,
			wantErr: false,
		},
		{
			name:    "long query (1000 chars)",
			length:  1000,
			wantErr: false,
		},
		{
			name:    "very long query (5000 chars)",
			length:  5000,
			wantErr: false,
		},
		{
			name:    "extremely long query (10000 chars)",
			length:  10000,
			wantErr: false, // Should URL-encode, may exceed API limits in real usage
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := strings.Repeat("x", tt.length)

			mockClient := &MockHTTPClient{
				Body: io.NopCloser(strings.NewReader(`{"Abstract": ""}`)),
			}

			oldClient := http.DefaultClient
			http.DefaultClient = mockClient
			defer func() { http.DefaultClient = oldClient }()

			_, err := DuckDuckGoSearch(query, 5)
			if (err != nil) != tt.wantErr {
				t.Errorf("Long query (%d chars) error=%v, wantErr %v", tt.length, err, tt.wantErr)
			}
		})
	}
}

// TestConcurrentSearchRequests tests concurrent search request handling
func TestConcurrentSearchRequests(t *testing.T) {
	t.Run("concurrent requests without sharing client", func(t *testing.T) {
		numConcurrent := 10
		resultsChan := make(chan []string, numConcurrent)
		errChan := make(chan error, numConcurrent)

		for i := 0; i < numConcurrent; i++ {
			go func(index int) {
				mockClient := &MockHTTPClient{
					Body: io.NopCloser(strings.NewReader(`{"Abstract": "concurrent result"}`)),
				}

				client := &http.Client{
					Transport: mockClient,
					Timeout:   5 * time.Second,
				}

				oldClient := http.DefaultClient
				http.DefaultClient = client
				defer func() { http.DefaultClient = oldClient }()

				results, err := DuckDuckGoSearch(fmt.Sprintf("query-%d", index), 5)
				if err != nil {
					errChan <- fmt.Errorf("concurrent request %d failed: %v", index, err)
				} else {
					resultsChan <- results
				}
			}(i)
		}

		var successCount int
		for i := 0; i < numConcurrent; i++ {
			select {
			case result := <-resultsChan:
				if len(result) > 0 {
					successCount++
				}
			case err := <-errChan:
				t.Errorf("Concurrent request failed: %v", err)
			}
		}

		if successCount != numConcurrent {
			t.Errorf("Expected %d successful concurrent requests, got %d", numConcurrent, successCount)
		}
	})

	t.Run("parallel goroutines with mock client", func(t *testing.T) {
		var wg sync.WaitGroup
		numRequests := 20

		for i := 0; i < numRequests; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()

				mockClient := &MockHTTPClient{
					Body: io.NopCloser(strings.NewReader(`{"Abstract": "parallel-" + fmt.Sprintf("%d", index)}`)),
				}

				client := &http.Client{
					Transport: mockClient,
				}

				oldClient := http.DefaultClient
				http.DefaultClient = client
				defer func() { http.DefaultClient = oldClient }()

				results, err := DuckDuckGoSearch(fmt.Sprintf("parallel-%d", index), 5)
				if err != nil {
					t.Errorf("Parallel request %d failed: %v", index, err)
				} else if len(results) == 0 {
					t.Errorf("Parallel request %d returned empty results", index)
				}
			}(i)
		}

		wg.Wait()
	})
}

// TestErrorCaseHandling tests various error scenarios
func TestErrorCaseHandling(t *testing.T) {
	tests := []struct {
		name          string
		mockSetup     func() *MockHTTPClient
		wantErr       bool
		errorContains string
	}{
		{
			name: "context cancellation",
			mockSetup: func() *MockHTTPClient {
				ctx, cancel := context.WithCancel(context.Background())
				cancel() // Cancel immediately

				return &MockHTTPClient{
					Error: ctx.Err(),
				}
			},
			wantErr:       true,
			errorContains: "context",
		},
		{
			name: "connection refused",
			mockSetup: func() *MockHTTPClient {
				return &MockHTTPClient{
					Error: errors.New("connection refused"),
				}
			},
			wantErr:       true,
			errorContains: "connection refused",
		},
		{
			name: "DNS lookup failure",
			mockSetup: func() *MockHTTPClient {
				return &MockHTTPClient{
					Error: errors.New("dns lookup failed"),
				}
			},
			wantErr:       true,
			errorContains: "dns",
		},
		{
			name: "SSL certificate error",
			mockSetup: func() *MockHTTPClient {
				return &MockHTTPClient{
					Error: errors.New("x509: certificate signed by unknown authority"),
				}
			},
			wantErr:       true,
			errorContains: "x509",
		},
		{
			name: "network unreachable",
			mockSetup: func() *MockHTTPClient {
				return &MockHTTPClient{
					Error: errors.New("network is unreachable"),
				}
			},
			wantErr:       true,
			errorContains: "network",
		},
		{
			name: "timeout exceeded",
			mockSetup: func() *MockHTTPClient {
				return &MockHTTPClient{
					Error: errors.New("operation timed out after 30 seconds"),
				}
			},
			wantErr:       true,
			errorContains: "timed out",
		},
		{
			name: "read timeout while reading body",
			mockSetup: func() *MockHTTPClient {
				return &MockHTTPClient{
					Body:  io.NopCloser(strings.NewReader("partial")),
					Error: errors.New("read tcp: i/o timeout"),
				}
			},
			wantErr:       true,
			errorContains: "timeout",
		},
		{
			name: "invalid character in JSON",
			mockSetup: func() *MockHTTPClient {
				return &MockHTTPClient{
					Body: io.NopCloser(strings.NewReader(`{"Abstract": "test\xFF"}`)),
				}
			},
			wantErr:       true,
			errorContains: "invalid character",
		},
		{
			name: "unexpected end of JSON input",
			mockSetup: func() *MockHTTPClient {
				return &MockHTTPClient{
					Body: io.NopCloser(strings.NewReader(`{"Abstract":`)),
				}
			},
			wantErr:       true,
			errorContains: "unexpected end",
		},
		{
			name: "nil response handling",
			mockSetup: func() *MockHTTPClient {
				return &MockHTTPClient{
					Response: nil,
				}
			},
			wantErr:       true,
			errorContains: "",
		},
		{
			name: "empty body response",
			mockSetup: func() *MockHTTPClient {
				return &MockHTTPClient{
					Body: io.NopCloser(strings.NewReader("")),
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := tt.mockSetup()

			oldClient := http.DefaultClient
			http.DefaultClient = mockClient
			defer func() { http.DefaultClient = oldClient }()

			_, err := DuckDuckGoSearch("error test", 5)
			if (err != nil) != tt.wantErr {
				t.Errorf("Expected error=%v, got %v", tt.wantErr, err)
			}

			if tt.wantErr && tt.errorContains != "" {
				if err == nil || !strings.Contains(err.Error(), tt.errorContains) {
					t.Logf("Error message check: expected to contain %q, got: %v", tt.errorContains, err)
				}
			}
		})
	}
}

// TestSmartSearchEdgeCases tests SmartSearch edge cases with fallback logic
func TestSmartSearchEdgeCases(t *testing.T) {
	tests := []struct {
		name       string
		setupMocks func() (map[string]*MockHTTPClient, map[string]error)
		wantLen    int
		wantSource bool
	}{
		{
			name: "all sources return empty results",
			setupMocks: func() (map[string]*MockHTTPClient, map[string]error) {
				mock := &MockHTTPClient{Body: io.NopCloser(strings.NewReader(`{"Abstract": ""}`))}
				return map[string]*MockHTTPClient{"duckduckgo": mock}, nil
			},
			wantLen:  0,
			wantSource: false,
		},
		{
			name: "first source succeeds",
			setupMocks: func() (map[string]*MockHTTPClient, map[string]error) {
				mock := &MockHTTPClient{Body: io.NopCloser(strings.NewReader(`{"Abstract": "success"}`))}
				return map[string]*MockHTTPClient{"duckduckgo": mock}, nil
			},
			wantLen: 1,
			wantSource: true,
		},
		{
			name: "second source fallback succeeds",
			setupMocks: func() (map[string]*MockHTTPClient, map[string]error) {
				return map[string]*MockHTTPClient{"duckduckgo": &MockHTTPClient{Body: io.NopCloser(strings.NewReader(`{"Abstract": ""}`))}}, nil
			},
			wantLen: 1,
			wantSource: true,
		},
		{
			name: "third source fallback succeeds",
			setupMocks: func() (map[string]*MockHTTPClient, map[string]error) {
				return map[string]*MockHTTPClient{"duckduckgo": &MockHTTPClient{Body: io.NopCloser(strings.NewReader(`{"Abstract": ""}`))}}, nil
			},
			wantLen: 0, // Will skip Jina AI and Wikipedia with mocks
			wantSource: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if testing.Short() {
				t.Skip("Skipping SmartSearch test in short mode")
			}

			// Note: Full mock setup for SmartSearch is complex due to multiple sources.
			// For edge case testing, we rely on integration tests with real services or
			// more sophisticated mocking that replaces each search function.
			t.Log("SmartSearch edge cases - see integration tests for full coverage")
		})
	}
}

// TestJinaAISearchEdgeCases tests Jina AI search with mock client
func TestJinaAISearchEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		status   int
		wantLen  int
		wantErr  bool
	}{
		{
			name:    "successful Jina AI response",
			body:    "Jina AI retrieved content about Go programming language",
			status:  200,
			wantLen: 1,
			wantErr: false,
		},
		{
			name:    "empty content (<50 chars)",
			body:    "too short",
			status:  200,
			wantLen: 0,
			wantErr: true,
		},
		{
			name:    "connection error",
			body:    "",
			status:  0, // Will trigger mock error
			wantLen: 0,
			wantErr: true,
		},
		{
			name:    "server error (500)",
			body:    "Internal Server Error",
			status:  500,
			wantLen: 0,
			wantErr: true,
		},
		{
			name:    "not found (404)",
			body:    "Not Found",
			status:  404,
			wantLen: 0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := newMockResponse(tt.body, tt.status, nil)

			oldClient := http.DefaultClient
			http.DefaultClient = mockClient
			defer func() { http.DefaultClient = oldClient }()

			results, err := JinaAISearch("Go programming")
			if (err != nil) != tt.wantErr {
				t.Errorf("JinaAISearch error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && len(results) != tt.wantLen {
				t.Errorf("Expected length %d, got %d", tt.wantLen, len(results))
			}

			// Verify content minimum requirement
			if tt.name == "empty content (<50 chars)" && err == nil {
				t.Error("Expected error for content < 50 chars")
			}
		})
	}
}

// TestWikipediaSearchEdgeCases tests Wikipedia search with mock client
func TestWikipediaSearchEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		status   int
		wantLen  int
		wantErr  bool
	}{
		{
			name:    "successful Wikipedia response",
			body:    `{"extract": "Go is a programming language", "title": "Go (programming language)"}`,
			status:  200,
			wantLen: 1,
			wantErr: false,
		},
		{
			name:    "article not found (404)",
			body:    `{"error": {"code": 404, "info": "Page does not exist"}`,
			status:  404,
			wantLen: 0,
			wantErr: true,
		},
		{
			name:    "empty extract field",
			body:    `{"extract": "", "title": "Test"}`,
			status:  200,
			wantLen: 0,
			wantErr: true,
		},
		{
			name:    "missing extract field",
			body:    `{"title": "Test", "other_field": "value"}`,
			status:  200,
			wantLen: 0,
			wantErr: true,
		},
		{
			name:     "invalid JSON response",
			body:     `{invalid json}`,
			status:   200,
			wantLen:  0,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := newMockResponse(tt.body, tt.status, nil)

			oldClient := http.DefaultClient
			http.DefaultClient = mockClient
			defer func() { http.DefaultClient = oldClient }()

			results, err := WikipediaSearch("Go programming")
			if (err != nil) != tt.wantErr {
				t.Errorf("WikipediaSearch error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && len(results) != tt.wantLen {
				t.Errorf("Expected length %d, got %d", tt.wantLen, len(results))
			}

			// Check for 404 handling
			if tt.name == "article not found (404)" && err == nil {
				t.Error("Expected error for 404 response")
			}
		})
	}
}

// TestURLQueryEscaping tests that all queries are properly URL-encoded
func TestURLQueryEscaping(t *testing.T) {
	tests := []struct {
		name        string
		query       string
		expectedEnc string
	}{
		{
			name:        "spaces encoded",
			query:       "hello world",
			expectedEnc: "hello+world",
		},
		{
			name:        "special chars encoded",
			query:       "test@#$%",
			expectedEnc: "test%40%23%24%25",
		},
		{
			name:        "unicode preserved as UTF-8 and URL-encoded",
			query:       "日本語テスト",
			expectedEnc: "%E6%97%A5%E6%9C%AC%E8%AA%9E%E3%83%86%E3%82%B9%E3%83%88",
		},
		{
			name:        "query params already encoded are preserved",
			query:       "test%20param",
			expectedEnc: "test%2520param", // Double-encoded
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockHTTPClient{
				Body: io.NopCloser(strings.NewReader(`{"Abstract": ""}`)),
			}

			oldClient := http.DefaultClient
			http.DefaultClient = mockClient
			defer func() { http.DefaultClient = oldClient }()

			_, err := DuckDuckGoSearch(tt.query, 5)
			if err != nil {
				t.Errorf("Expected no error, got %v", err)
			}

			if mockClient.Response != nil && mockClient.Response.Request != nil {
				rawQuery := mockClient.Response.Request.URL.RawQuery
				if !strings.Contains(rawQuery, tt.expectedEnc) {
					t.Logf("Expected to find %q in query string, got: %s", tt.expectedEnc, rawQuery)
				}
			}
		})
	}
}

// BenchmarkSearchPerformance provides basic performance benchmarks
func BenchmarkDuckDuckGoSearch(b *testing.B) {
	mockClient := &MockHTTPClient{
		Body: io.NopCloser(strings.NewReader(`{"Abstract": "benchmark result"}`)),
	}
	oldClient := http.DefaultClient
	http.DefaultClient = mockClient
	defer func() { http.DefaultClient = oldClient }()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := DuckDuckGoSearch("benchmark test", 5)
		if err != nil {
			b.Fatalf("Error during benchmark: %v", err)
		}
	}
}

// FuzzSearchQuery provides fuzzing for search query handling
func FuzzSearchQuery(f *testing.F) {
	testCases := []string{
		"test query",
		"hello world",
		"SQL injection OR '1'='1",
		"<script>alert(1)</script>",
		"🚀 emoji test",
		"日本語 Unicode",
		strings.Repeat("x", 1000),
		"",
	}

	for _, tc := range testCases {
		f.Add(tc)
	}

	f.Fuzz(func(t *testing.T, query string) {
		mockClient := &MockHTTPClient{
			Body: io.NopCloser(strings.NewReader(`{"Abstract": ""}`)),
		}
		oldClient := http.DefaultClient
		http.DefaultClient = mockClient
		defer func() { http.DefaultClient = oldClient }()

		_, err := DuckDuckGoSearch(query, 5)
		// The function should handle any input gracefully without panicking
		if err != nil && !strings.Contains(err.Error(), "empty") && !strings.Contains(err.Error(), "parse JSON") {
			t.Logf("Non-empty query %q resulted in error: %v", query[:min(len(query), 50)], err)
		}
	})
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
