package victron

import (
	"context"
	"fmt"
	"math/rand"
	"time"
)

// RetryConfig holds configuration for retry operations.
type RetryConfig struct {
	MaxRetries int
	BaseDelay  time.Duration
	MaxDelay   time.Duration
	Multiplier float64
	Jitter     bool
}

// DefaultRetryConfig returns a sensible default retry configuration.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries: 3,
		BaseDelay:  100 * time.Millisecond,
		MaxDelay:   5 * time.Second,
		Multiplier: 2.0,
		Jitter:     true,
	}
}

// Retry executes the given function with exponential backoff retry logic.
// It returns the result of the last attempt and any error encountered.
func Retry[T any](ctx context.Context, config RetryConfig, fn func() (T, error)) (T, error) {
	var result T
	var lastErr error

	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		if attempt > 0 {
			delay := calculateDelay(attempt, config)

			// Check if context is cancelled before waiting
			select {
			case <-ctx.Done():
				return result, ctx.Err()
			case <-time.After(delay):
				// Continue to next retry
			}
		}

		result, err := fn()
		if err == nil {
			return result, nil
		}

		lastErr = err

		// Don't retry if this is the last attempt
		if attempt == config.MaxRetries {
			break
		}
	}

	return result, fmt.Errorf("operation failed after %d attempts: %w",
		config.MaxRetries+1, lastErr)
}

// calculateDelay calculates the delay before the next retry using exponential backoff.
func calculateDelay(attempt int, config RetryConfig) time.Duration {
	// Calculate exponential delay: baseDelay * (multiplier ^ (attempt - 1))
	delay := float64(config.BaseDelay) * pow(config.Multiplier, float64(attempt-1))

	// Cap at max delay
	if delay > float64(config.MaxDelay) {
		delay = float64(config.MaxDelay)
	}

	// Add jitter if enabled (±25% variation)
	if config.Jitter {
		variation := delay * 0.25
		delay = delay + (variation * randFloat()) - variation
	}

	return time.Duration(delay)
}

// pow is a simple power function for float64 values.
func pow(base, exponent float64) float64 {
	result := 1.0
	for i := 0; i < int(exponent); i++ {
		result *= base
	}
	return result
}

// randFloat returns a random float between 0 and 1.
func randFloat() float64 {
	return rand.Float64()
}

// Test functions for coverage (used by retry_test.go)

// RetryWithFixedResult is a test helper that creates a function returning fixed values.
func RetryWithFixedResult(result int, err error) func() (int, error) {
	return func() (int, error) {
		return result, err
	}
}

// RetryWithCalls tracks the number of times a function is called.
type RetryCallTracker struct {
	Calls  int
	Result int
	Error  error
}

func NewRetryCallTracker(result int, err error) *RetryCallTracker {
	return &RetryCallTracker{Result: result, Error: err}
}

func (t *RetryCallTracker) Func() (int, error) {
	t.Calls++
	return t.Result, t.Error
}
