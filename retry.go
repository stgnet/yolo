package main

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// RetryConfig holds configuration for retry logic with exponential backoff
type RetryConfig struct {
	MaxRetries   int           // Maximum number of retry attempts
	InitialDelay time.Duration // Initial delay between retries
	MaxDelay     time.Duration // Maximum delay between retries
	Multiplier   float64       // Multiplier for exponential backoff
}

// DefaultRetryConfig provides sensible defaults for HTTP requests
var DefaultRetryConfig = RetryConfig{
	MaxRetries:   3,
	InitialDelay: 1 * time.Second,
	MaxDelay:     15 * time.Second,
	Multiplier:   2.0,
}

// ShouldRetry determines if a request should be retried based on response
func ShouldRetry(resp *http.Response, err error) bool {
	// Network errors are retryable
	if err != nil {
		return true
	}

	// Retry on server errors (5xx) and rate limiting (429)
	if resp.StatusCode == http.StatusTooManyRequests ||
		resp.StatusCode >= 500 {
		return true
	}

	return false
}

// calculateDelay calculates the delay for a given attempt using exponential backoff
func calculateDelay(attempt int, config RetryConfig) time.Duration {
	delay := config.InitialDelay
	for i := 0; i < attempt; i++ {
		delay = time.Duration(float64(delay) * config.Multiplier)
		if delay > config.MaxDelay {
			delay = config.MaxDelay
			break
		}
	}
	return delay
}

// RetryWithBackoff executes an HTTP request with exponential backoff retry logic
func RetryWithBackoff(ctx context.Context, url string, config RetryConfig) (*http.Response, error) {
	client := &http.Client{Timeout: 30 * time.Second}

	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request on attempt %d: %w", attempt+1, err)
		}

		req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; YOLO-Agent/1.0)")

		resp, err := client.Do(req)

		if !ShouldRetry(resp, err) {
			return resp, err
		}

		if attempt < config.MaxRetries {
			delay := calculateDelay(attempt, config)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
				// Continue to next retry
			}
		}
	}

	return nil, fmt.Errorf("all %d retry attempts failed", config.MaxRetries+1)
}

// ExecuteWithRetry is a generic function for executing any operation with retry logic
func ExecuteWithRetry[T any](ctx context.Context, maxRetries int, fn func() (T, error)) (T, error) {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		result, err := fn()
		if err == nil {
			return result, nil
		}

		lastErr = err

		if attempt < maxRetries {
			delay := calculateDelay(attempt, DefaultRetryConfig)
			select {
			case <-ctx.Done():
				var zero T
				return zero, ctx.Err()
			case <-time.After(delay):
				// Continue to next retry
			}
		}
	}

	var zero T
	return zero, lastErr
}
