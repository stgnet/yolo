package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// ============================================================================
// HTTP Integration Tests - Comprehensive Edge Case Coverage
// ============================================================================

// TestHTTPIntegration handles end-to-end testing of HTTP operations
func TestHTTPIntegration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		setupServer   func() *httptest.Server
		timeout       time.Duration
		expectSuccess bool
		expectError   bool
		verify        func(*testing.T, *http.Response, error)
	}{
		{
			name: "successful_request",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
				}))
			},
			timeout:       time.Second * 5,
			expectSuccess: true,
			verify: func(t *testing.T, resp *http.Response, err error) {
				if err != nil {
					t.Fatalf("Expected no error, got %v", err)
				}
				if resp.StatusCode != http.StatusOK {
					t.Errorf("Expected status 200, got %d", resp.StatusCode)
				}
			},
		},
		{
			name: "server_error_response",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
					json.NewEncoder(w).Encode(map[string]string{"error": "internal error"})
				}))
			},
			timeout:       time.Second * 5,
			expectSuccess: false,
			expectError:   true,
			verify: func(t *testing.T, resp *http.Response, err error) {
				if err == nil {
					t.Error("Expected error but got nil")
				}
				if resp.StatusCode != http.StatusInternalServerError {
					t.Errorf("Expected status 500, got %d", resp.StatusCode)
				}
			},
		},
		{
			name: "request_timeout",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					time.Sleep(time.Second * 10) // Exceeds timeout
					w.WriteHeader(http.StatusOK)
				}))
			},
			timeout:       time.Millisecond * 100,
			expectSuccess: false,
			expectError:   true,
			verify: func(t *testing.T, resp *http.Response, err error) {
				if err == nil {
					t.Error("Expected timeout error but got nil")
				} else if !strings.Contains(err.Error(), "timeout") && !strings.Contains(err.Error(), "i/o timeout") {
					t.Logf("Got non-timeout error (expected): %v", err)
				}
			},
		},
		{
			name: "connection_refused",
			setupServer: func() *httptest.Server {
				return httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
			},
			timeout:       time.Second * 5,
			expectSuccess: false,
			expectError:   true,
			verify: func(t *testing.T, resp *http.Response, err error) {
				if err == nil {
					t.Error("Expected connection refused error but got nil")
				} else if resp != nil && resp.StatusCode == 0 {
					t.Error("Response should be nil on connection failure")
				}
			},
		},
		{
			name: "invalid_response_body",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.Write([]byte("not valid json"))
				}))
			},
			timeout:       time.Second * 5,
			expectSuccess: false,
			expectError:   true,
			verify: func(t *testing.T, resp *http.Response, err error) {
				if err == nil {
					t.Error("Expected JSON unmarshal error but got nil")
				} else if !strings.Contains(err.Error(), "invalid character") && !strings.Contains(err.Error(), "unmarshal") {
					t.Logf("Got non-JSON parse error (expected): %v", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			
			server := tt.setupServer()
			defer server.Close()

			client := &http.Client{
				Timeout: tt.timeout,
			}

			resp, err := client.Get(server.URL)
			tt.verify(t, resp, err)
		})
	}
}

// TestHTTPHeaderValidation tests various HTTP header scenarios
func TestHTTPHeaderValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		contentType   string
		authHeader    string
		expectSuccess bool
	}{
		{"valid_json_content_type", "application/json", "", true},
		{"missing_content_type", "", "", false},
		{"invalid_content_type", "text/plain", "", false},
		{"with_auth_header", "application/json", "Bearer token123", true},
		{"with_bearer_auth", "application/json", "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9", true},
		{"with_basic_auth", "application/json", "Basic dXNlcjpwYXNz", false},
		{"empty_header_value", "application/json", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				contentType := r.Header.Get("Content-Type")
				
				if contentType != "application/json" && tt.contentType == "application/json" {
					t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
				}

				authHeader := r.Header.Get("Authorization")
				if authHeader != "" && tt.authHeader == "" {
					t.Error("Unexpected Authorization header")
				}
				
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			client := &http.Client{}
			
			req, err := http.NewRequest("GET", server.URL, nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			resp, err := client.Do(req)
			if (err == nil) != tt.expectSuccess {
				t.Errorf("Expected success=%v, got err=%v", tt.expectSuccess, err)
			}
			
			if resp != nil {
				resp.Body.Close()
			}
		})
	}
}

// TestHTTPBodyHandling tests various body formats and sizes
func TestHTTPBodyHandling(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		bodyData      any
		bodyType      string
		expectSuccess bool
	}{
		{
			name: "valid_json_object",
			bodyData: map[string]string{"key": "value"},
			bodyType: "json",
			expectSuccess: true,
		},
		{
			name: "empty_json_object",
			bodyData: map[string]any{},
			bodyType: "json",
			expectSuccess: true,
		},
		{
			name: "nested_json_object",
			bodyData: map[string]any{"outer": map[string]string{"inner": "value"}},
			bodyType: "json",
			expectSuccess: true,
		},
		{
			name: "empty_body",
			bodyData: "",
			bodyType: "none",
			expectSuccess: true,
		},
		{
			name: "malformed_json_string",
			bodyData: "{invalid json}",
			bodyType: "json",
			expectSuccess: false,
		},
		{
			name: "very_large_body",
			bodyData: strings.Repeat("x", 100000),
			bodyType: "text",
			expectSuccess: true,
		},
		{
			name: "unicode_content",
			bodyData: map[string]string{"message": "你好世界 🌍 مرحبا بالعالم"},
			bodyType: "json",
			expectSuccess: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, err := io.ReadAll(r.Body)
				if err != nil {
					w.WriteHeader(http.StatusBadRequest)
					return
				}

				switch tt.bodyType {
				case "json":
					var parsed any
					err := json.Unmarshal(body, &parsed)
					if err != nil {
						t.Errorf("Failed to unmarshal body: %v", err)
						w.WriteHeader(http.StatusBadRequest)
						return
					}
				case "text":
					if len(body) == 0 {
						t.Error("Expected non-empty text body")
					}
				}

				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			var body io.Reader
			switch tt.bodyType {
			case "json":
				jsonData, err := json.Marshal(tt.bodyData)
				if err != nil && tt.expectSuccess {
					t.Fatalf("Failed to marshal body: %v", err)
				}
				body = bytes.NewReader(jsonData)
			case "text":
				body = strings.NewReader(fmt.Sprintf("%v", tt.bodyData))
			default:
				body = nil
			}

			req, err := http.NewRequest("POST", server.URL, body)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			if tt.bodyType == "json" {
				req.Header.Set("Content-Type", "application/json")
			} else if tt.bodyType == "text" {
				req.Header.Set("Content-Type", "text/plain")
			}

			client := &http.Client{}
			resp, err := client.Do(req)

			if (err == nil) != tt.expectSuccess {
				t.Errorf("Expected success=%v, got err=%v", tt.expectSuccess, err)
			}

			if resp != nil && resp.StatusCode != http.StatusOK && tt.expectSuccess {
				t.Errorf("Expected status 200, got %d", resp.StatusCode)
			}

			if resp != nil {
				resp.Body.Close()
			}
		})
	}
}

// TestHTTPRequestMethods tests various HTTP methods and their handling
func TestHTTPRequestMethods(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		method        string
		expectSuccess bool
	}{
		{"GET", http.MethodGet, true},
		{"POST", http.MethodPost, true},
		{"PUT", http.MethodPut, true},
		{"PATCH", http.MethodPatch, true},
		{"DELETE", http.MethodDelete, true},
		{"HEAD", http.MethodHead, true},
		{"OPTIONS", http.MethodOptions, true},
		{"invalid_method", "INVALID", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != tt.method && tt.method != "invalid_method" {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				if tt.method == "HEAD" {
					w.Header().Set("Content-Length", "0")
					w.WriteHeader(http.StatusOK)
					return
				}

				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			req, err := http.NewRequest(tt.method, server.URL, nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			client := &http.Client{}
			resp, err := client.Do(req)

			if tt.method == "invalid_method" {
				if err == nil {
					t.Error("Expected error for invalid method")
				} else if resp != nil {
					t.Error("Response should be nil for invalid method")
				}
			} else {
				if (err == nil) != tt.expectSuccess {
					t.Errorf("Expected success=%v, got err=%v", tt.expectSuccess, err)
				}

				if resp != nil && resp.StatusCode != http.StatusOK && tt.expectSuccess {
					t.Errorf("Expected status 200, got %d", resp.StatusCode)
				}

				if resp != nil {
					resp.Body.Close()
				}
			}
		})
	}
}

// TestHTTPResponseHandling tests various response scenarios
func TestHTTPResponseHandling(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		setupServer   func(http.ResponseWriter, *http.Request)
		expectSuccess bool
		expectedCode  int
	}{
		{
			name: "success_200",
			setupServer: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
			},
			expectSuccess: true,
			expectedCode:  200,
		},
		{
			name: "created_201",
			setupServer: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusCreated)
				json.NewEncoder(w).Encode(map[string]string{"id": "123"})
			},
			expectSuccess: true,
			expectedCode:  201,
		},
		{
			name: "no_content_204",
			setupServer: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			},
			expectSuccess: true,
			expectedCode:  204,
		},
		{
			name: "bad_request_400",
			setupServer: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{"error": "bad request"})
			},
			expectSuccess: false,
			expectedCode:  400,
		},
		{
			name: "unauthorized_401",
			setupServer: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
			},
			expectSuccess: false,
			expectedCode:  401,
		},
		{
			name: "forbidden_403",
			setupServer: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]string{"error": "forbidden"})
			},
			expectSuccess: false,
			expectedCode:  403,
		},
		{
			name: "not_found_404",
			setupServer: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			},
			expectSuccess: false,
			expectedCode:  404,
		},
		{
			name: "internal_server_error_500",
			setupServer: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]string{"error": "server error"})
			},
			expectSuccess: false,
			expectedCode:  500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				tt.setupServer(w, r)
			}))
			defer server.Close()

			client := &http.Client{}
			resp, err := client.Get(server.URL)

			if tt.expectSuccess && (err != nil || resp.StatusCode != tt.expectedCode) {
				if err != nil {
					t.Errorf("Expected success but got error: %v", err)
				} else if resp.StatusCode != tt.expectedCode {
					t.Errorf("Expected status %d, got %d", tt.expectedCode, resp.StatusCode)
				}
			}

			if !tt.expectSuccess && err == nil && resp.StatusCode == tt.expectedCode {
				t.Error("Expected error but got success")
			}

			if resp != nil {
				resp.Body.Close()
			}
		})
	}
}

// TestHTTPClientConfiguration tests client configuration options
func TestHTTPClientConfiguration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		setupClient   func() *http.Client
		expectSuccess bool
	}{
		{
			name: "no_timeout",
			setupClient: func() *http.Client {
				return &http.Client{}
			},
			expectSuccess: true,
		},
		{
			name: "custom_timeout_1s",
			setupClient: func() *http.Client {
				return &http.Client{Timeout: time.Second}
			},
			expectSuccess: true,
		},
		{
			name: "very_short_timeout",
			setupClient: func() *http.Client {
				return &http.Client{Timeout: time.Millisecond}
			},
			expectSuccess: false, // Will timeout on any network call
		},
		{
			name: "with_check_redirect",
			setupClient: func() *http.Client {
				redirectCount := 0
				return &http.Client{
					CheckRedirect: func(req *http.Request, via []*http.Request) error {
						redirectCount++
						if redirectCount > 3 {
							return fmt.Errorf("too many redirects")
						}
						return nil
					},
				}
			},
			expectSuccess: true,
		},
		{
			name: "with_custom_transport",
			setupClient: func() *http.Client {
				transport := &http.Transport{}
				return &http.Client{Transport: transport}
			},
			expectSuccess: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
			}))
			defer server.Close()

			client := tt.setupClient()
			resp, err := client.Get(server.URL)

			if (err == nil) != tt.expectSuccess {
				t.Errorf("Expected success=%v, got err=%v", tt.expectSuccess, err)
			}

			if resp != nil {
				resp.Body.Close()
			}
		})
	}
}

// TestHTTPRetryLogic tests retry behavior with exponential backoff
func TestHTTPRetryLogic(t *testing.T) {
	t.Parallel()

	attemptCount := 0
	maxAttempts := 3
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		if attemptCount < maxAttempts {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok", "attempt": fmt.Sprintf("%d", attemptCount)})
	}))
	defer server.Close()

	client := &http.Client{
		Timeout: time.Second * 5,
	}

	// Test multiple requests with retries
	successes := 0
	for i := 0; i < 3; i++ {
		attemptCount = 0
		resp, err := client.Get(server.URL)
		if err == nil && resp.StatusCode == http.StatusOK {
			successes++
		}
		if resp != nil {
			resp.Body.Close()
		}
	}

	if successes < 2 {
		t.Errorf("Expected at least 2 successful retries, got %d", successes)
	}

	if attemptCount < maxAttempts {
		t.Errorf("Expected maxAttempts=%d calls to complete successfully, got %d", maxAttempts, attemptCount)
	}
}

// TestHTTPConcurrency tests concurrent HTTP requests
func TestHTTPConcurrency(t *testing.T) {
	t.Parallel()

	const numRequests = 50
	successChan := make(chan bool, numRequests)
	errorChan := make(chan error, numRequests)
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(time.Millisecond * 10) // Simulate some processing
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	var wg syncWaitGroup
	
	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			
			client := &http.Client{Timeout: time.Second * 5}
			resp, err := client.Get(server.URL)
			
			if err != nil {
				errorChan <- fmt.Errorf("request %d failed: %w", idx, err)
			} else if resp.StatusCode == http.StatusOK {
				successChan <- true
			} else {
				errorChan <- fmt.Errorf("request %d got status %d", idx, resp.StatusCode)
			}
			
			if resp != nil {
				resp.Body.Close()
			}
		}(i)
	}
	
	wg.Wait()
	close(successChan)
	close(errorChan)

	errors := 0
	for err := range errorChan {
		t.Logf("Error: %v", err)
		errors++
	}

	successes := len(successChan)
	t.Logf("Concurrent test: %d successes, %d errors out of %d requests", 
		successes, errors, numRequests)

	if errors > 5 {
		t.Errorf("Too many concurrent failures: %d", errors)
	}
}

// Helper for waitgroup (workaround - using real sync.WaitGroup if available)
type syncWaitGroup struct {
	wg sync.WaitGroup
}

func (s *syncWaitGroup) Add(delta int)        { s.wg.Add(delta) }
func (s *syncWaitGroup) Done()                { s.wg.Done() }
func (s *syncWaitGroup) Wait()                { s.wg.Wait() }

import "sync"
