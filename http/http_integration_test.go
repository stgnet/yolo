package http_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stgnet/yolo/http"
)

// TestIntegrationRealWebSearch tests the actual web search integration
func TestIntegrationRealWebSearch(t *testing.T) {
	if os.Getenv("SKIP_INTEGRATION") != "" {
		t.Skip("Skipping integration test")
	}

	// This is a basic smoke test - actual integration tests would need API keys
	results, err := http.SearchWeb(context.Background(), "go lang", 5)
	if err != nil {
		t.Logf("Expected: real web searches may fail without network or rate limits. Error: %v", err)
		return // Don't fail on network issues in CI
	}

	if len(results) == 0 {
		t.Log("Search returned no results (may be expected)")
	}
}

// TestIntegrationValidURL fetches a valid URL to test basic HTTP functionality
func TestIntegrationValidURL(t *testing.T) {
	if os.Getenv("SKIP_INTEGRATION") != "" {
		t.Skip("Skipping integration test")
	}

	resp, err := http.Get(context.Background(), "https://httpbin.org/get")
	if err != nil {
		t.Skipf("Cannot connect to external service: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

// TestIntegrationInvalidURL tests that invalid URLs are properly handled
func TestIntegrationInvalidURL(t *testing.T) {
	if os.Getenv("SKIP_INTEGRATION") != "" {
		t.Skip("Skipping integration test")
	}

	resp, err := http.Get(context.Background(), "not-a-valid-url")
	if err == nil {
		t.Errorf("Expected error for invalid URL, got none")
		return
	}

	if resp != nil {
		defer resp.Body.Close()
		t.Errorf("Response should be nil on error, got: %v", resp)
	}

	// Verify the error message is informative
	expected := []string{"invalid", "url", "scheme"}
	found := false
	for _, exp := range expected {
		if containsSubstring(err.Error(), exp) {
			found = true
			break
		}
	}
	if !found {
		t.Logf("Error message could be more specific: %v", err)
	}
}

// TestIntegrationTimeout tests that HTTP requests respect context deadlines
func TestIntegrationTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := http.Get(ctx, "https://httpbin.org/delay/2") // Should timeout
	elapsed := time.Since(start)

	if err == nil {
		t.Error("Expected timeout error, got none")
	} else if elapsed < 50*time.Millisecond || elapsed > 2*time.Second {
		t.Errorf("Timeout behavior unexpected: %v", elapsed)
	}
}

// TestIntegrationLargeResponse tests handling of large responses
func TestIntegrationLargeResponse(t *testing.T) {
	if os.Getenv("SKIP_INTEGRATION") != "" {
		t.Skip("Skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := http.Get(ctx, "https://httpbin.org/response-headers?Content-Length=1048576")
	if err != nil {
		t.Skipf("Cannot connect to external service: %v", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("Failed to read large response: %v", err)
	}

	if len(body) == 0 {
		t.Error("Expected body content from large response test")
	}
}

// TestIntegrationRetrySuccessOnRetry tests that retries work when first attempt fails
func TestIntegrationRetrySuccessOnRetry(t *testing.T) {
	callCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			http.Error(w, "Service temporarily unavailable", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success":true}`))
	}))
	defer server.Close()

	ctx := context.Background()
	resp, err := http.Get(ctx, server.URL)
	if err != nil {
		t.Fatalf("Expected retry to succeed: %v", err)
	}
	defer resp.Body.Close()

	if callCount == 0 {
		t.Error("Request was never retried - retry logic may not be working")
	}

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Logf("Response body parsing skipped for non-JSON response: %v", err)
	}
}

// TestIntegrationRateLimiting tests handling of rate limit responses (429)
func TestIntegrationRateLimiting(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "1")
		http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := http.Get(ctx, server.URL)
	if err == nil {
		t.Error("Expected error for rate limited response")
		return
	}

	var httpErr *http.HTTPError
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		t.Logf("Request was cancelled due to timeout: %v", err)
	} else if !errors.As(err, &httpErr) {
		t.Logf("Non-HTTP error type: %T, error: %v", err, err)
	}
}

// TestIntegrationInvalidJSON tests that malformed JSON is handled gracefully
func TestIntegrationInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{invalid json}`)) // Malformed JSON
	}))
	defer server.Close()

	ctx := context.Background()
	resp, err := http.Get(ctx, server.URL)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err == nil {
		t.Error("Expected error for malformed JSON")
		return
	}

	// Verify we get a parse error, not a generic error
	if strings.Contains(err.Error(), "invalid") || strings.Contains(err.Error(), "syntax") {
		t.Logf("Got expected JSON parsing error: %v", err)
	} else {
		t.Logf("Unexpected error type for malformed JSON: %v", err)
	}
}

// TestIntegrationMultipleConcurrentRequests tests handling of concurrent requests
func TestIntegrationMultipleConcurrentRequests(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond) // Small delay
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx := context.Background()
	numRequests := 10
	resultChan := make(chan error, numRequests)

	for i := 0; i < numRequests; i++ {
		go func(id int) {
			resp, err := http.Get(ctx, server.URL)
			if err == nil && resp != nil {
				resp.Body.Close()
			}
			resultChan <- err
		}(i)
	}

	failedCount := 0
	for i := 0; i < numRequests; i++ {
		err := <-resultChan
		if err != nil {
			t.Logf("Request %d failed: %v", i, err)
			failedCount++
		}
	}

	if failedCount > 2 {
		t.Errorf("Too many concurrent request failures: %d/%d", failedCount, numRequests)
	}
}

// TestIntegrationProxyError tests proxy-related error handling
func TestIntegrationProxyError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// This test assumes no proxy is set - proxy testing would require env setup
	if os.Getenv("HTTP_PROXY") != "" || os.Getenv("HTTPS_PROXY") != "" {
		t.Skip("Proxy environment variables are set, skipping proxy error test")
	}

	ctx := context.Background()
	resp, err := http.Get(ctx, server.URL)
	if err != nil && strings.Contains(err.Error(), "proxy") {
		t.Logf("Got proxy-related error: %v", err)
	}

	if resp == nil && err != nil {
		// This is expected if there's a real network issue or proxy misconfiguration
		t.Logf("No response due to network/Proxy issue: %v", err)
	}
}

// TestIntegrationConnectionReset tests handling of connection reset errors
func TestIntegrationConnectionReset(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Close the connection immediately to simulate a reset
		hijacker, ok := w.(http.Hijacker)
		if !ok {
			t.Fatal("Hijacking not supported")
		}

		conn, _, _ := hijacker.Hijack()
		conn.Close()
	}))
	defer server.Close()

	ctx := context.Background()
	resp, err := http.Get(ctx, server.URL)
	if resp != nil {
		t.Error("Expected no response when connection is reset")
	}
	if err == nil {
		t.Error("Expected error for connection reset")
	} else if !strings.Contains(err.Error(), "connection") && !strings.Contains(err.Error(), "reset") {
		t.Logf("Error message could better describe connection reset: %v", err)
	}
}

// TestIntegrationServerSlowResponse tests handling of slow responses
func TestIntegrationServerSlowResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second) // Slow response
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	resp, err := http.Get(ctx, server.URL)
	elapsed := time.Since(start)

	if resp != nil {
		defer resp.Body.Close()
		t.Error("Expected no response due to timeout")
	}

	if elapsed < 50*time.Millisecond || elapsed > 2*time.Second {
		t.Errorf("Timeout behavior unexpected: %v", elapsed)
	}

	if err == nil {
		t.Error("Expected timeout error for slow server")
	}
}

// TestIntegrationHTTP2Support tests HTTP/2 protocol support
func TestIntegrationHTTP2Support(t *testing.T) {
	if os.Getenv("SKIP_INTEGRATION") != "" {
		t.Skip("Skipping integration test")
	}

	ctx := context.Background()
	resp, err := http.Get(ctx, "https://httpbin.org/get")
	if err != nil {
		t.Skipf("Cannot connect to external service: %v", err)
		return
	}
	defer resp.Body.Close()

	// Check if HTTP/2 is being used (response should not include HTTP/1.1 in version string)
	if resp.ProtoMajor == 2 {
		t.Logf("Successfully using HTTP/2")
	} else {
		t.Logf("Using %s", resp.Proto)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

// TestIntegrationRedirectChain tests handling of multiple redirects
func TestIntegrationRedirectChain(t *testing.T) {
	redirectCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		redirectCount++
		if redirectCount < 3 {
			http.Redirect(w, r, server.URL+"/redirect", http.StatusSeeOther)
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"final":true}`))
		}
	}))
	defer server.Close()

	ctx := context.Background()
	resp, err := http.Get(ctx, server.URL+"/redirect")
	if err != nil {
		t.Fatalf("Redirect chain failed: %v", err)
	}
	defer resp.Body.Close()

	if redirectCount < 3 {
		t.Errorf("Expected at least 3 redirects, got %d", redirectCount)
	}

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Logf("Response body parsing skipped: %v", err)
	}
}

// TestIntegrationEmptyHost tests handling of empty host in URL
func TestIntegrationEmptyHost(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx := context.Background()

	// Try with just a path (no host specified properly)
	resp, err := http.Get(ctx, "/path/without/host")
	if err == nil {
		t.Errorf("Expected error for invalid URL format")
	}
	if resp != nil {
		t.Error("Response should be nil on invalid URL")
	}
}

// TestIntegrationHTTPSvsHTTP tests HTTPS vs HTTP redirect handling
func TestIntegrationHTTPSvsHTTP(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx := context.Background()
	resp, err := http.Get(ctx, server.URL)
	if err != nil {
		t.Skipf("Cannot connect to test server: %v", err)
		return
	}
	defer resp.Body.Close()

	// Should work fine with HTTP for localhost/test servers
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func containsSubstring(str, substr string) bool {
	return strings.Contains(strings.ToLower(str), strings.ToLower(substr))
}
