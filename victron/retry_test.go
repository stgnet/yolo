package victron

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestRetry_SuccessOnFirstAttempt(t *testing.T) {
	ctx := context.Background()
	config := DefaultRetryConfig()

	tracker := NewRetryCallTracker(42, nil)

	result, err := Retry(ctx, config, tracker.Func)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if result != 42 {
		t.Errorf("Expected result 42, got: %d", result)
	}
	if tracker.Calls != 1 {
		t.Errorf("Expected 1 call, got: %d", tracker.Calls)
	}
}

func TestRetry_SuccessOnSecondAttempt(t *testing.T) {
	ctx := context.Background()
	config := RetryConfig{
		MaxRetries: 3,
		BaseDelay:  10 * time.Millisecond,
		MaxDelay:   50 * time.Millisecond,
		Multiplier: 2.0,
		Jitter:     false,
	}

	callCount := 0
	fn := func() (int, error) {
		callCount++
		if callCount == 1 {
			return 0, fmt.Errorf("attempt %d failed", callCount) // Simulate failure on first attempt
		}
		return 42, nil // Success on second attempt
	}

	result, err := Retry(ctx, config, fn)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if result != 42 {
		t.Errorf("Expected result 42, got: %d", result)
	}
	if callCount != 2 {
		t.Errorf("Expected 2 calls, got: %d", callCount)
	}
}

func TestRetry_AllAttemptsFail(t *testing.T) {
	ctx := context.Background()
	config := RetryConfig{
		MaxRetries: 2,
		BaseDelay:  10 * time.Millisecond,
		MaxDelay:   50 * time.Millisecond,
		Multiplier: 2.0,
		Jitter:     false,
	}

	callCount := 0
	fn := func() (int, error) {
		callCount++
		return 0, fmt.Errorf("attempt %d failed", callCount)
	}

	_, actualErr := Retry(ctx, config, fn)

	if actualErr == nil {
		t.Error("Expected error, got nil")
	}

	if callCount != 3 { // Initial + 2 retries = 3 attempts
		t.Errorf("Expected 3 calls, got: %d", callCount)
	}
}

func TestRetry_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	config := RetryConfig{
		MaxRetries: 5,
		BaseDelay:  100 * time.Millisecond,
		MaxDelay:   200 * time.Millisecond,
		Multiplier: 2.0,
		Jitter:     false,
	}

	callCount := 0
	fn := func() (int, error) {
		callCount++
		return 0, fmt.Errorf("attempt %d failed", callCount)
	}

	go func() {
		time.Sleep(150 * time.Millisecond) // Cancel after first delay
		cancel()
	}()

	_, err := Retry(ctx, config, fn)
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled error, got: %v", err)
	}
}

func TestRetryConfig_Defaults(t *testing.T) {
	config := DefaultRetryConfig()

	if config.MaxRetries != 3 {
		t.Errorf("Expected MaxRetries=3, got: %d", config.MaxRetries)
	}
	if config.BaseDelay != 100*time.Millisecond {
		t.Errorf("Expected BaseDelay=100ms, got: %v", config.BaseDelay)
	}
	if config.MaxDelay != 5*time.Second {
		t.Errorf("Expected MaxDelay=5s, got: %v", config.MaxDelay)
	}
	if config.Multiplier != 2.0 {
		t.Errorf("Expected Multiplier=2.0, got: %f", config.Multiplier)
	}
	if !config.Jitter {
		t.Error("Expected Jitter=true")
	}
}

func TestRetry_CalculateDelayExponentialBackoff(t *testing.T) {
	config := RetryConfig{
		BaseDelay:  100 * time.Millisecond,
		MaxDelay:   5 * time.Second,
		Multiplier: 2.0,
		Jitter:     false,
	}

	expectedDelays := []time.Duration{
		100 * time.Millisecond, // attempt 1: 100ms * 2^0 = 100ms
		200 * time.Millisecond, // attempt 2: 100ms * 2^1 = 200ms
		400 * time.Millisecond, // attempt 3: 100ms * 2^2 = 400ms
		800 * time.Millisecond, // attempt 4: 100ms * 2^3 = 800ms
	}

	for i, expected := range expectedDelays {
		attempt := i + 1
		delay := calculateDelay(attempt, config)
		if delay != expected {
			t.Errorf("Attempt %d: Expected delay %v, got %v", attempt, expected, delay)
		}
	}
}

func TestRetry_CalculateDelayRespectsMaxDelay(t *testing.T) {
	config := RetryConfig{
		BaseDelay:  100 * time.Millisecond,
		MaxDelay:   500 * time.Millisecond, // Cap at 500ms
		Multiplier: 2.0,
		Jitter:     false,
	}

	delay := calculateDelay(10, config) // Would be huge without max

	if delay > config.MaxDelay {
		t.Errorf("Expected delay <= %v, got %v", config.MaxDelay, delay)
	}
}

func TestRetry_CalculateDelayWithJitter(t *testing.T) {
	config := RetryConfig{
		BaseDelay:  100 * time.Millisecond,
		MaxDelay:   5 * time.Second,
		Multiplier: 2.0,
		Jitter:     true, // Enable jitter
	}

	// Without jitter, delay for attempt 2 would be exactly 200ms
	// With jitter, it should vary between 150ms and 250ms (±25%)
	delay := calculateDelay(2, config)

	// Expected range: 200ms ± 25% = 150ms to 250ms
	minExpected := 150 * time.Millisecond // 200ms - 25%
	maxExpected := 250 * time.Millisecond // 200ms + 25%

	if delay < minExpected || delay > maxExpected {
		t.Errorf("Expected delay with jitter to be between %v and %v, got %v",
			minExpected, maxExpected, delay)
	}
}

func TestRetry_RetryWithFixedResult(t *testing.T) {
	expectedResult := 123
	expectedErr := fmt.Errorf("test error")

	fn := RetryWithFixedResult(expectedResult, expectedErr)
	result, err := fn()

	if result != expectedResult {
		t.Errorf("Expected result %d, got: %d", expectedResult, result)
	}
	if err == nil || err.Error() != expectedErr.Error() {
		t.Errorf("Expected error '%v', got: %v", expectedErr, err)
	}
}

func TestRetry_RandFloat(t *testing.T) {
	for i := 0; i < 10; i++ {
		val := randFloat()
		if val < 0.0 || val > 1.0 {
			t.Errorf("randFloat should return value between 0 and 1, got: %f", val)
		}
	}
}
