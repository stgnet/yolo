package http

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestHTTPError_Error(t *testing.T) {
	err := &HTTPError{Code: 500, Message: "Internal Server Error", RetryAfter: 30 * time.Second}
	expected := "HTTP 500: Internal Server Error"
	if err.Error() != expected {
		t.Errorf("Expected %q, got %q", expected, err.Error())
	}
}

func TestNewHTTPClient(t *testing.T) {
	client := NewHTTPClient()
	if client == nil {
		t.Fatal("Expected non-nil HTTPClient")
	}
	if client.client == nil {
		t.Fatal("Expected non-nil internal client")
	}
	if client.client.Timeout != 30*time.Second {
		t.Errorf("Expected timeout 30s, got %v", client.client.Timeout)
	}
}

func TestHTTPClient_Do_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Hello"))
	}))
	defer server.Close()

	client := NewHTTPClient()
	req := Request{
		Method:     http.MethodGet,
		URL:        server.URL,
		Timeout:    5 * time.Second,
		RetryCount: 0,
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}
	if string(resp.Body) != "Hello" {
		t.Errorf("Expected body 'Hello', got %q", string(resp.Body))
	}
	if resp.URL != server.URL {
		t.Errorf("Expected URL %q, got %q", server.URL, resp.URL)
	}
}

func TestHTTPClient_Do_500Error_NoRetry(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error"))
	}))
	defer server.Close()

	client := NewHTTPClient()
	req := Request{
		Method:     http.MethodGet,
		URL:        server.URL,
		Timeout:    5 * time.Second,
		RetryCount: 0, // No retries configured
	}

	start := time.Now()
	resp, err := client.Do(req)
	duration := time.Since(start)

	if resp != nil {
		t.Error("Expected nil response on error")
	}
	if err == nil {
		t.Fatal("Expected error")
	}

	httpErr, ok := err.(*HTTPError)
	if !ok {
		t.Fatalf("Expected *HTTPError, got %T", err)
	}
	if httpErr.Code != 500 {
		t.Errorf("Expected code 500, got %d", httpErr.Code)
	}

	// Should not retry with RetryCount=0
	if duration >= time.Second {
		t.Errorf("Request took too long (%v), should not have retried", duration)
	}
}

func TestHTTPClient_Do_RetryOn500(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Error"))
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Success"))
		}
	}))
	defer server.Close()

	client := NewHTTPClient()
	req := Request{
		Method:     http.MethodGet,
		URL:        server.URL,
		Timeout:    5 * time.Second,
		RetryCount: 5,
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Unexpected error after retries: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}
	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}

func TestHTTPClient_Do_Cancelled(t *testing.T) {
	cancelChan := make(chan struct{})
	close(cancelChan) // Immediately cancelled

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewHTTPClient()
	req := Request{
		Method:     http.MethodGet,
		URL:        server.URL,
		Timeout:    5 * time.Second,
		RetryCount: 2,
		Cancel:     cancelChan,
	}

	_, err := client.Do(req)
	if err == nil {
		t.Error("Expected error on cancelled request")
	}
}

func TestHTTPClient_Do_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second) // Longer than timeout
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewHTTPClient()
	req := Request{
		Method:     http.MethodGet,
		URL:        server.URL,
		Timeout:    100 * time.Millisecond,
		RetryCount: 0, // No retries
		Cancel:     make(chan struct{}), // Enable cancel tracking
	}

	start := time.Now()
	_, err := client.Do(req)
	duration := time.Since(start)

	if err == nil {
		t.Fatal("Expected timeout error")
	}

	// The error could be *HTTPError or *url.Error wrapping context.Canceled
	if httpErr, ok := err.(*HTTPError); ok {
		if httpErr.Code != 408 {
			t.Errorf("Expected code 408 (timeout), got %d", httpErr.Code)
		}
	} else if urlErr, ok := err.(*url.Error); ok && strings.Contains(urlErr.Err.Error(), "context canceled") {
		// This is also acceptable - timeout results in context cancellation
	} else {
		t.Logf("Got error type %T: %v", err, err)
	}

	// Should timeout quickly, not wait full 2 seconds
	if duration >= 500*time.Millisecond {
		t.Errorf("Request took too long (%v), should timeout quickly", duration)
	}
}

func TestHTTPClient_Get(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("GET success"))
	}))
	defer server.Close()

	client := NewHTTPClient()
	resp, err := client.Get(server.URL, 5*time.Second, 0)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}
	if string(resp.Body) != "GET success" {
		t.Errorf("Expected body 'GET success', got %q", string(resp.Body))
	}
}

func TestHTTPClient_Post(t *testing.T) {
	var receivedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedBody = make([]byte, r.ContentLength)
		r.Body.Read(receivedBody)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("POST success"))
	}))
	defer server.Close()

	client := NewHTTPClient()
	body := []byte(`{"key":"value"}`)
	resp, err := client.Post(server.URL, body, 5*time.Second, 0)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}
	if string(receivedBody) != `{"key":"value"}` {
		t.Errorf("Expected body %q, got %q", `{"key":"value"}`, string(receivedBody))
	}
}

func TestHTTPClient_DoRateLimitedWithRetryAfter(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte("Rate limited"))
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Success"))
		}
	}))
	defer server.Close()

	client := NewHTTPClient()
	req := Request{
		Method:     http.MethodGet,
		URL:        server.URL,
		Timeout:    10 * time.Second,
		RetryCount: 2,
	}

	start := time.Now()
	resp, err := client.Do(req)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("Unexpected error after retry: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}
	if attempts != 2 {
		t.Errorf("Expected 2 attempts, got %d", attempts)
	}

	// Should have waited at least 1 second for retry
	if duration < time.Second {
		t.Errorf("Expected to wait for Retry-After header, but only took %v", duration)
	}
}

func TestHTTPClient_DoConnectionRefused(t *testing.T) {
	client := NewHTTPClient()
	req := Request{
		Method:     http.MethodGet,
		URL:        "http://localhost:59999/nonexistent", // Unlikely to be in use
		Timeout:    1 * time.Second,
		RetryCount: 0,
	}

	_, err := client.Do(req)
	if err == nil {
		t.Error("Expected connection error")
	}

	httpErr, ok := err.(*HTTPError)
	if !ok {
		t.Fatalf("Expected *HTTPError, got %T", err)
	}
	if httpErr.Code != 503 {
		t.Errorf("Expected code 503 (connection refused), got %d", httpErr.Code)
	}
}

func TestHTTPClient_withCancel_ContextTimeout(t *testing.T) {
	client := NewHTTPClient()
	cancelChan := make(chan struct{})
	req := Request{
		Timeout: 50 * time.Millisecond,
		Cancel:  cancelChan,
	}

	ctx := client.withCancel(req)
	
	// Wait for context to be cancelled by timeout
	select {
	case <-ctx.Done():
		// Expected - context should be cancelled after timeout
	case <-time.After(500 * time.Millisecond):
		t.Error("Context should have been cancelled by timeout")
	}
}

func TestHTTPClient_withCancel_ExplicitCancel(t *testing.T) {
	client := NewHTTPClient()
	cancelChan := make(chan struct{})
	req := Request{
		Timeout: 5 * time.Second,
		Cancel:  cancelChan,
	}

	ctx := client.withCancel(req)
	close(cancelChan)

	// Wait for context to be cancelled (with small delay for goroutine scheduling)
	select {
	case <-ctx.Done():
		// Expected - context should be cancelled immediately
	case <-time.After(100 * time.Millisecond):
		t.Error("Context should have been cancelled")
	}
}

func TestHTTPClient_withCancel_NoCancellation(t *testing.T) {
	client := NewHTTPClient()
	req := Request{
		Timeout: 0,    // No timeout
		Cancel:  nil,  // No cancel channel
	}

	ctx := client.withCancel(req)
	
	// Context should not be immediately done (should be context.Background())
	select {
	case <-ctx.Done():
		t.Error("Context should not be done immediately when no cancellation configured")
	default:
		// Expected - context is still active
	}
}

func TestHTTPClient_mapHTTPError(t *testing.T) {
	client := NewHTTPClient()

	// Test context deadline exceeded
	err := client.mapHTTPError(context.DeadlineExceeded)
	if err == nil {
		t.Fatal("Expected error for context deadline exceeded")
	}
	httpErr, ok := err.(*HTTPError)
	if !ok {
		t.Fatalf("Expected *HTTPError, got %T", err)
	}
	if httpErr.Code != 408 {
		t.Errorf("Expected code 408 for timeout, got %d", httpErr.Code)
	}

	// Test connection refused (simulated with string)
	simErr := &net.OpError{Op: "dial", Net: "tcp", Err: errors.New("connection refused")}
	err = client.mapHTTPError(simErr)
	if err == nil {
		t.Fatal("Expected error for connection refused")
	}
	httpErr, ok = err.(*HTTPError)
	if !ok {
		t.Fatalf("Expected *HTTPError, got %T", err)
	}
	if httpErr.Code != 503 {
		t.Errorf("Expected code 503 for connection refused, got %d", httpErr.Code)
	}
}

func TestResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Custom-Header", "custom-value")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("Created"))
	}))
	defer server.Close()

	client := NewHTTPClient()
	req := Request{
		Method:     http.MethodPost,
		URL:        server.URL,
		Timeout:    5 * time.Second,
		RetryCount: 0,
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("Expected status %d, got %d", http.StatusCreated, resp.StatusCode)
	}
	if string(resp.Body) != "Created" {
		t.Errorf("Expected body 'Created', got %q", string(resp.Body))
	}
	if resp.Headers.Get("X-Custom-Header") != "custom-value" {
		t.Errorf("Expected header 'X-Custom-Header: custom-value', got %q", resp.Headers.Get("X-Custom-Header"))
	}
	if resp.Duration <= 0 {
		t.Error("Expected positive duration")
	}
}

func TestHTTPClient_Do_UserAgentHeader(t *testing.T) {
	var userAgent string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userAgent = r.Header.Get("User-Agent")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewHTTPClient()
	req := Request{
		Method:     http.MethodGet,
		URL:        server.URL,
		Timeout:    5 * time.Second,
		RetryCount: 0,
	}

	_, err := client.Do(req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if userAgent == "" {
		t.Error("Expected User-Agent header to be set")
	}
	if !strings.Contains(userAgent, "YOLO/1.0") {
		t.Errorf("Expected User-Agent to contain 'YOLO/1.0', got %q", userAgent)
	}
}
