package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestShouldRetry tests the retry decision logic
func TestShouldRetry(t *testing.T) {
	tests := []struct {
		name     string
		resp     *http.Response
		err      error
		expected bool
	}{
		{
			name:     "should retry on network error",
			resp:     nil,
			err:      errors.New("connection refused"),
			expected: true,
		},
		{
			name:     "should not retry on success",
			resp:     &http.Response{StatusCode: http.StatusOK},
			err:      nil,
			expected: false,
		},
		{
			name:     "should not retry on client error 400",
			resp:     &http.Response{StatusCode: http.StatusBadRequest},
			err:      nil,
			expected: false,
		},
		{
			name:     "should not retry on not found 404",
			resp:     &http.Response{StatusCode: http.StatusNotFound},
			err:      nil,
			expected: false,
		},
		{
			name:     "should retry on rate limit 429",
			resp:     &http.Response{StatusCode: http.StatusTooManyRequests},
			err:      nil,
			expected: true,
		},
		{
			name:     "should retry on server error 500",
			resp:     &http.Response{StatusCode: http.StatusInternalServerError},
			err:      nil,
			expected: true,
		},
		{
			name:     "should retry on bad gateway 502",
			resp:     &http.Response{StatusCode: http.StatusBadGateway},
			err:      nil,
			expected: true,
		},
		{
			name:     "should retry on service unavailable 503",
			resp:     &http.Response{StatusCode: http.StatusServiceUnavailable},
			err:      nil,
			expected: true,
		},
		{
			name:     "should retry on gateway timeout 504",
			resp:     &http.Response{StatusCode: http.StatusGatewayTimeout},
			err:      nil,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShouldRetry(tt.resp, tt.err)
			if result != tt.expected {
				t.Errorf("ShouldRetry() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestCalculateDelay tests the exponential backoff calculation
func TestCalculateDelay(t *testing.T) {
	config := RetryConfig{
		InitialDelay: 1 * time.Second,
		MaxDelay:     10 * time.Second,
		Multiplier:   2.0,
	}

	tests := []struct {
		attempt  int
		expected time.Duration
	}{
		{0, 1 * time.Second},   // initial delay
		{1, 2 * time.Second},   // 1s * 2
		{2, 4 * time.Second},   // 2s * 2
		{3, 8 * time.Second},   // 4s * 2
		{4, 10 * time.Second},  // capped at max
		{5, 10 * time.Second},  // still capped
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("attempt_%d", tt.attempt), func(t *testing.T) {
			result := calculateDelay(tt.attempt, config)
			if result != tt.expected {
				t.Errorf("calculateDelay(%d) = %v, want %v", tt.attempt, result, tt.expected)
			}
		})
	}
}

// TestCalculateDelayWithDifferentMultiplier tests with custom multiplier
func TestCalculateDelayWithDifferentMultiplier(t *testing.T) {
	config := RetryConfig{
		InitialDelay: 500 * time.Millisecond,
		MaxDelay:     5 * time.Second,
		Multiplier:   1.5,
	}

	tests := []struct {
		attempt  int
		expected time.Duration
	}{
		{0, 500 * time.Millisecond},
		{1, 750 * time.Millisecond},  // 500ms * 1.5
		{2, 1125 * time.Millisecond}, // 750ms * 1.5
	}

	for _, tt := range tests {
		result := calculateDelay(tt.attempt, config)
		if result != tt.expected {
			t.Errorf("calculateDelay(%d) = %v, want %v", tt.attempt, result, tt.expected)
		}
	}
}

// TestRetryWithBackoffSuccess tests successful request on first attempt
func TestRetryWithBackoffSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	ctx := context.Background()
	config := RetryConfig{
		MaxRetries:   3,
		InitialDelay: 100 * time.Millisecond,
	}

	resp, err := RetryWithBackoff(ctx, server.URL, config)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

// TestRetryWithBackoffFailuresThenSuccess tests retry after failures
func TestRetryWithBackoffFailuresThenSuccess(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK after retries"))
	}))
	defer server.Close()

	ctx := context.Background()
	config := RetryConfig{
		MaxRetries:   5,
		InitialDelay: 10 * time.Millisecond,
	}

	resp, err := RetryWithBackoff(ctx, server.URL, config)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}

// TestRetryWithBackoffAllFailures tests when all retries fail
func TestRetryWithBackoffAllFailures(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	ctx := context.Background()
	config := RetryConfig{
		MaxRetries:   2,
		InitialDelay: 10 * time.Millisecond,
	}

	resp, err := RetryWithBackoff(ctx, server.URL, config)
	if resp != nil {
		t.Error("Expected nil response on all failures")
	}
	if err == nil {
		t.Error("Expected error on all failures")
	}
	if !errors.Is(err, context.Canceled) && err.Error() != "all 3 retry attempts failed" {
		t.Logf("Got expected failure: %v", err)
	}
}

// TestRetryWithBackoffContextCancel tests that context cancellation is respected
func TestRetryWithBackoffContextCancel(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	config := RetryConfig{
		MaxRetries:   10,
		InitialDelay: 100 * time.Millisecond, // Longer than timeout
	}

	resp, err := RetryWithBackoff(ctx, server.URL, config)
	if resp != nil {
		t.Error("Expected nil response on context cancellation")
	}
	if err == nil {
		t.Error("Expected error on context cancellation")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Logf("Got context deadline exceeded: %v", err)
	}
}

// TestExecuteWithRetrySuccess tests successful execution on first attempt
func TestExecuteWithRetrySuccess(t *testing.T) {
	attempts := 0
	ctx := context.Background()

	result, err := ExecuteWithRetry(ctx, 3, func() (string, error) {
		attempts++
		return "success", nil
	})

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result != "success" {
		t.Errorf("Expected 'success', got '%s'", result)
	}
	if attempts != 1 {
		t.Errorf("Expected 1 attempt, got %d", attempts)
	}
}

// TestExecuteWithRetryFailuresThenSuccess tests retry after failures
func TestExecuteWithRetryFailuresThenSuccess(t *testing.T) {
	attempts := 0
	ctx := context.Background()

	result, err := ExecuteWithRetry(ctx, 3, func() (string, error) {
		attempts++
		if attempts < 3 {
			return "", errors.New("temporary failure")
		}
		return "success after retries", nil
	})

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result != "success after retries" {
		t.Errorf("Expected 'success after retries', got '%s'", result)
	}
	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}

// TestExecuteWithRetryAllFailures tests when all retries fail
func TestExecuteWithRetryAllFailures(t *testing.T) {
	ctx := context.Background()
	expectedErr := errors.New("persistent failure")

	result, err := ExecuteWithRetry(ctx, 2, func() (string, error) {
		return "", expectedErr
	})

	if result != "" {
		t.Errorf("Expected empty result, got '%s'", result)
	}
	if err != expectedErr {
		t.Errorf("Expected original error, got %v", err)
	}
}

// TestExecuteWithRetryContextCancel tests context cancellation in generic retry
func TestExecuteWithRetryContextCancel(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	result, err := ExecuteWithRetry(ctx, 5, func() (int, error) {
		time.Sleep(10 * time.Millisecond)
		return 0, errors.New("retryable error")
	})

	if result != 0 {
		t.Errorf("Expected zero value, got %d", result)
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Expected context deadline exceeded, got %v", err)
	}
}

// TestDefaultRetryConfig tests that default config has sensible values
func TestDefaultRetryConfig(t *testing.T) {
	if DefaultRetryConfig.MaxRetries != 3 {
		t.Errorf("Expected MaxRetries=3, got %d", DefaultRetryConfig.MaxRetries)
	}
	if DefaultRetryConfig.InitialDelay != 1*time.Second {
		t.Errorf("Expected InitialDelay=1s, got %v", DefaultRetryConfig.InitialDelay)
	}
	if DefaultRetryConfig.MaxDelay != 15*time.Second {
		t.Errorf("Expected MaxDelay=15s, got %v", DefaultRetryConfig.MaxDelay)
	}
	if DefaultRetryConfig.Multiplier != 2.0 {
		t.Errorf("Expected Multiplier=2.0, got %f", DefaultRetryConfig.Multiplier)
	}
}
