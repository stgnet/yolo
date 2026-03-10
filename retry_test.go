package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestShouldRetry(t *testing.T) {
	tests := []struct {
		name     string
		resp     *http.Response
		err      error
		expected bool
	}{
		{
			name:     "network error should retry",
			resp:     nil,
			err:      &netError{},
			expected: true,
		},
		{
			name:     "500 status should retry",
			resp:     &http.Response{StatusCode: 500},
			err:      nil,
			expected: true,
		},
		{
			name:     "503 status should retry",
			resp:     &http.Response{StatusCode: 503},
			err:      nil,
			expected: true,
		},
		{
			name:     "429 status should retry",
			resp:     &http.Response{StatusCode: 429},
			err:      nil,
			expected: true,
		},
		{
			name:     "200 status should not retry",
			resp:     &http.Response{StatusCode: 200},
			err:      nil,
			expected: false,
		},
		{
			name:     "404 status should not retry",
			resp:     &http.Response{StatusCode: 404},
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShouldRetry(tt.resp, tt.err)
			if result != tt.expected {
				t.Errorf("ShouldRetry() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestCalculateDelay(t *testing.T) {
	config := RetryConfig{
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     2 * time.Second,
		Multiplier:   2.0,
	}

	tests := []struct {
		attempt  int
		expected time.Duration
	}{
		{0, 100 * time.Millisecond},
		{1, 200 * time.Millisecond},
		{2, 400 * time.Millisecond},
		{3, 800 * time.Millisecond},
		{4, 1600 * time.Millisecond},
		{5, 2 * time.Second},  // capped at MaxDelay
		{10, 2 * time.Second}, // still capped
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("attempt_%d", tt.attempt), func(t *testing.T) {
			result := calculateDelay(tt.attempt, config)
			if result != tt.expected {
				t.Errorf("calculateDelay(%d) = %v, expected %v", tt.attempt, result, tt.expected)
			}
		})
	}
}

func TestRetryWithBackoff_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))
	defer server.Close()

	ctx := context.Background()
	resp, err := RetryWithBackoff(ctx, server.URL, DefaultRetryConfig)
	if err != nil {
		t.Fatalf("RetryWithBackoff() failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestRetryWithBackoff_RetriesOnFailure(t *testing.T) {
	attemptCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		if attemptCount < 3 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success after retries"))
	}))
	defer server.Close()

	ctx := context.Background()
	config := RetryConfig{
		MaxRetries:   5,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
		Multiplier:   2.0,
	}

	resp, err := RetryWithBackoff(ctx, server.URL, config)
	if err != nil {
		t.Fatalf("RetryWithBackoff() failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	if attemptCount != 3 {
		t.Errorf("Expected 3 attempts, got %d", attemptCount)
	}
}

func TestRetryWithBackoff_MaxRetriesExceeded(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	ctx := context.Background()
	config := RetryConfig{
		MaxRetries:   2,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     50 * time.Millisecond,
		Multiplier:   2.0,
	}

	resp, err := RetryWithBackoff(ctx, server.URL, config)
	if err == nil {
		t.Fatal("Expected error when all retries fail, got nil")
	}

	if resp != nil {
		t.Error("Expected nil response on failure, got non-nil")
	}
}

func TestRetryWithBackoff_ContextCancellation(t *testing.T) {
	attemptCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		time.Sleep(100 * time.Millisecond) // Slow enough to trigger cancellation
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	config := RetryConfig{
		MaxRetries:   5,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     50 * time.Millisecond,
		Multiplier:   2.0,
	}

	start := time.Now()
	resp, err := RetryWithBackoff(ctx, server.URL, config)
	duration := time.Since(start)

	if err == nil || err != context.DeadlineExceeded {
		t.Errorf("Expected context.DeadlineExceeded error, got %v", err)
	}

	if resp != nil {
		t.Error("Expected nil response on cancellation, got non-nil")
	}

	if duration >= 100*time.Millisecond {
		t.Errorf("Expected quick timeout (<100ms), took %v", duration)
	}
}

func TestExecuteWithRetry_Success(t *testing.T) {
	attempt := 0
	fn := func() (string, error) {
		attempt++
		if attempt < 3 {
			return "", fmt.Errorf("temporary failure")
		}
		return "success", nil
	}

	ctx := context.Background()
	result, err := ExecuteWithRetry(ctx, 5, fn)
	if err != nil {
		t.Fatalf("ExecuteWithRetry() failed: %v", err)
	}

	if result != "success" {
		t.Errorf("Expected 'success', got '%s'", result)
	}

	if attempt != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempt)
	}
}

func TestExecuteWithRetry_MaxRetriesExceeded(t *testing.T) {
	fn := func() (string, error) {
		return "", fmt.Errorf("persistent failure")
	}

	ctx := context.Background()
	result, err := ExecuteWithRetry(ctx, 2, fn)
	if err == nil {
		t.Fatal("Expected error when all retries fail, got nil")
	}

	if result != "" {
		t.Errorf("Expected empty string on failure, got '%s'", result)
	}
}

// netError is a mock network error for testing
type netError struct{}

func (e *netError) Error() string {
	return "network error"
}
