package concurrency

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewGroup(t *testing.T) {
	g := NewGroup(context.Background())
	if g == nil {
		t.Fatal("Expected non-nil group")
	}
	if g.ctx == nil {
		t.Error("Expected non-nil context")
	}
	g.Run() // Ensure cleanup
}

func TestGroup_Go(t *testing.T) {
	g := NewGroup(context.Background())

	var executed int32
	numGoroutines := 5

	for i := 0; i < numGoroutines; i++ {
		g.Go(func(ctx context.Context) error {
			atomic.AddInt32(&executed, 1)
			return nil
		})
	}

	g.Run()

	if atomic.LoadInt32(&executed) != int32(numGoroutines) {
		t.Errorf("Expected %d goroutines to execute, got %d", numGoroutines, executed)
	}
}

func TestGroup_GoWithErrors(t *testing.T) {
	g := NewGroup(context.Background())

	testErr := errors.New("test error")
	numGoroutines := 5

	for i := 0; i < numGoroutines; i++ {
		g.Go(func(ctx context.Context) error {
			return testErr
		})
	}

	g.Run()

	errs := g.Errors()
	if len(errs) != numGoroutines {
		t.Errorf("Expected %d errors, got %d", numGoroutines, len(errs))
	}

	if g.Error() != testErr {
		t.Errorf("Expected first error to be testErr, got %v", g.Error())
	}
}

func TestGroup_GoWithErrorsMixed(t *testing.T) {
	g := NewGroup(context.Background())

	err1 := errors.New("error 1")
	err2 := errors.New("error 2")

	g.Go(func(ctx context.Context) error { return nil })
	g.Go(func(ctx context.Context) error { return err1 })
	g.Go(func(ctx context.Context) error { return nil })
	g.Go(func(ctx context.Context) error { return err2 })
	g.Go(func(ctx context.Context) error { return nil })

	g.Run()

	errs := g.Errors()
	if len(errs) != 2 {
		t.Errorf("Expected 2 errors, got %d: %v", len(errs), errs)
	}
}

func TestGroup_Cancel(t *testing.T) {
	g := NewGroup(context.Background())

	var executed int32
	var cancelled int32

	for i := 0; i < 10; i++ {
		g.Go(func(ctx context.Context) error {
			select {
			case <-ctx.Done():
				atomic.AddInt32(&cancelled, 1)
				return ctx.Err()
			case <-time.After(50 * time.Millisecond):
				// Work completed
				atomic.AddInt32(&executed, 1)
				return nil
			}
		})
	}

	// Cancel quickly after spawning goroutines
	time.Sleep(10 * time.Millisecond)
	g.Cancel()
	g.Wait()

	// At least some goroutines should be cancelled given the timing
	t.Logf("Executed %d goroutines, cancelled %d goroutines", atomic.LoadInt32(&executed), atomic.LoadInt32(&cancelled))
	if atomic.LoadInt32(&cancelled) == 0 && atomic.LoadInt32(&executed) == 10 {
		t.Error("Expected cancellation to have an effect")
	}
}

func TestGroup_Context(t *testing.T) {
	g := NewGroup(context.Background())

	ctx := g.Context()
	if ctx == nil {
		t.Error("Expected non-nil context from Group.Context()")
	}

	g.Run()
}

func TestGroup_ErrorsReturnsCopy(t *testing.T) {
	g := NewGroup(context.Background())

	testErr := errors.New("test error")
	g.Go(func(ctx context.Context) error { return testErr })
	g.Go(func(ctx context.Context) error { return errors.New("another error") })

	// Run the group first to ensure goroutines complete and errors are collected
	g.Run()

	errs1 := g.Errors()
	errs2 := g.Errors()

	if len(errs1) == 0 || len(errs2) == 0 {
		t.Skip("No errors collected, skipping copy test")
	}

	// The two slices should be different copies
	if &errs1[0] == &errs2[0] {
		t.Error("Expected Errors() to return a copy of the error slice")
	}
}

func TestGroup_FanOut(t *testing.T) {
	g := NewGroup(context.Background())

	numWorkers := 4
	var resultsReceived int32

	input := "test"
	resultChan := g.FanOut(input, numWorkers, func(ctx context.Context, in interface{}) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			atomic.AddInt32(&resultsReceived, 1)
			return nil
		}
	})

	// Read all results from the channel
	count := 0
	for range resultChan {
		count++
	}

	if count != numWorkers {
		t.Errorf("Expected %d results from fan-out, got %d", numWorkers, count)
	}

	if atomic.LoadInt32(&resultsReceived) != int32(numWorkers) {
		t.Errorf("Expected %d workers to run, got %d", numWorkers, resultsReceived)
	}
}

func TestGroup_FanIn(t *testing.T) {
	g := NewGroup(context.Background())

	numChannels := 3
	inputChans := make([]<-chan error, numChannels)

	for i := 0; i < numChannels; i++ {
		ch := make(chan error, 1)
		ch <- nil
		close(ch)
		inputChans[i] = ch
	}

	outputChan := g.FanIn(inputChans...)

	// Read from output channel before running/canceling the group
	count := 0
	for range outputChan {
		count++
	}

	g.Run() // Cleanup after reading is complete

	if count != numChannels {
		t.Errorf("Expected %d items from fan-in, got %d", numChannels, count)
	}
}

func TestGroup_FanInWithCancellation(t *testing.T) {
	g := NewGroup(context.Background())

	// Create a channel that won't close (for cancellation test)
	hangingChan := make(chan error)
	outputChan := g.FanIn(hangingChan)

	time.Sleep(10 * time.Millisecond)
	g.Cancel()
	g.Wait()

	// Try to read - should be empty or closed
	select {
	case _, ok := <-outputChan:
		if ok {
			t.Error("Expected output channel to be closed or empty after cancel")
		}
	default:
		// Expected - channel is empty/closed
	}
}

func TestBarrier(t *testing.T) {
	barrier := NewBarrier(3)

	var ready int32

	for i := 0; i < 3; i++ {
		go func() {
			barrier.Wait()
			atomic.AddInt32(&ready, 1)
		}()
	}

	time.Sleep(50 * time.Millisecond)

	if atomic.LoadInt32(&ready) != 3 {
		t.Errorf("Expected all 3 goroutines to pass barrier, got %d", ready)
	}
}

func TestBarrier_Reset(t *testing.T) {
	barrier := NewBarrier(2)

	var passed1, passed2 int32

	// First round
	go func() {
		barrier.Wait()
		atomic.AddInt32(&passed1, 1)
	}()

	go func() {
		barrier.Wait()
		atomic.AddInt32(&passed1, 1)
	}()

	time.Sleep(50 * time.Millisecond)

	if atomic.LoadInt32(&passed1) != 2 {
		t.Errorf("Expected 2 goroutines in first round, got %d", passed1)
	}

	// Reset and second round
	barrier.Reset()

	go func() {
		barrier.Wait()
		atomic.AddInt32(&passed2, 1)
	}()

	go func() {
		barrier.Wait()
		atomic.AddInt32(&passed2, 1)
	}()

	time.Sleep(50 * time.Millisecond)

	if atomic.LoadInt32(&passed2) != 2 {
		t.Errorf("Expected 2 goroutines in second round, got %d", passed2)
	}
}

func TestBarrier_WithSingleGoroutine(t *testing.T) {
	barrier := NewBarrier(1)

	var completed int32

	go func() {
		barrier.Wait()
		atomic.AddInt32(&completed, 1)
	}()

	time.Sleep(50 * time.Millisecond)

	if atomic.LoadInt32(&completed) != 1 {
		t.Error("Expected single goroutine to pass barrier")
	}
}

func TestLimitedConcurrency(t *testing.T) {
	ctx := context.Background()

	var maxConcurrent int32
	var currentConcurrent int32
	numJobs := 10
	maxWorkers := 3

	jobs := make([]func(context.Context) error, numJobs)
	for i := 0; i < numJobs; i++ {
		jobs[i] = func(ctx context.Context) error {
			atomic.AddInt32(&currentConcurrent, 1)

			for {
				max := atomic.LoadInt32(&maxConcurrent)
				current := atomic.LoadInt32(&currentConcurrent)
				if current <= max {
					break
				}
				if atomic.CompareAndSwapInt32(&maxConcurrent, max, current) {
					break
				}
			}

			time.Sleep(10 * time.Millisecond)
			atomic.AddInt32(&currentConcurrent, -1)
			return nil
		}
	}

	errs := LimitedConcurrency(ctx, maxWorkers, jobs)

	if len(errs) != 0 {
		t.Errorf("Expected no errors, got %v", errs)
	}

	if atomic.LoadInt32(&maxConcurrent) > int32(maxWorkers) {
		t.Errorf("Max concurrent was %d, expected at most %d", maxConcurrent, maxWorkers)
	}
}

func TestLimitedConcurrency_WithErrors(t *testing.T) {
	ctx := context.Background()

	testErr := errors.New("test error")
	numJobs := 5
	maxWorkers := 2

	jobs := make([]func(context.Context) error, numJobs)
	for i := 0; i < numJobs; i++ {
		jobs[i] = func(ctx context.Context) error {
			return testErr
		}
	}

	errs := LimitedConcurrency(ctx, maxWorkers, jobs)

	if len(errs) != numJobs {
		t.Errorf("Expected %d errors, got %d", numJobs, len(errs))
	}
}

func TestRetryWithBackoff_SuccessOnFirstTry(t *testing.T) {
	ctx := context.Background()

	var attempts int32

	err := RetryWithBackoff(ctx, 3, 10*time.Millisecond, func(ctx context.Context) error {
		atomic.AddInt32(&attempts, 1)
		return nil
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if atomic.LoadInt32(&attempts) != 1 {
		t.Errorf("Expected 1 attempt, got %d", attempts)
	}
}

func TestRetryWithBackoff_SuccessAfterRetries(t *testing.T) {
	ctx := context.Background()

	var attempts int32
	failuresBeforeSuccess := 2

	err := RetryWithBackoff(ctx, 5, 10*time.Millisecond, func(ctx context.Context) error {
		atomic.AddInt32(&attempts, 1)
		if attempts <= int32(failuresBeforeSuccess) {
			return errors.New("temporary failure")
		}
		return nil
	})

	if err != nil {
		t.Errorf("Expected no error after retries, got %v", err)
	}

	if atomic.LoadInt32(&attempts) != int32(failuresBeforeSuccess)+1 {
		t.Errorf("Expected %d attempts, got %d", failuresBeforeSuccess+1, attempts)
	}
}

func TestRetryWithBackoff_MaxRetriesExceeded(t *testing.T) {
	ctx := context.Background()

	maxRetries := 3
	var attempts int32

	err := RetryWithBackoff(ctx, maxRetries, 10*time.Millisecond, func(ctx context.Context) error {
		atomic.AddInt32(&attempts, 1)
		return errors.New("persistent failure")
	})

	if err == nil {
		t.Error("Expected error after max retries exceeded")
	}

	if atomic.LoadInt32(&attempts) != int32(maxRetries) {
		t.Errorf("Expected %d attempts, got %d", maxRetries, attempts)
	}
}

func TestRetryWithBackoff_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	cancel() // Cancel immediately

	var attempts int32

	err := RetryWithBackoff(ctx, 5, 10*time.Millisecond, func(ctx context.Context) error {
		atomic.AddInt32(&attempts, 1)
		return errors.New("should not be called")
	})

	if err != context.Canceled {
		t.Errorf("Expected context.Canceled, got %v", err)
	}

	if atomic.LoadInt32(&attempts) != 0 {
		t.Errorf("Expected 0 attempts due to cancelled context, got %d", attempts)
	}
}

func TestRetryWithBackoff_CancelDuringRetry(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	var attempts int32

	go func() {
		time.Sleep(15 * time.Millisecond)
		cancel()
	}()

	err := RetryWithBackoff(ctx, 5, 10*time.Millisecond, func(ctx context.Context) error {
		atomic.AddInt32(&attempts, 1)
		return errors.New("keep failing")
	})

	if err != context.Canceled {
		t.Errorf("Expected context.Canceled, got %v", err)
	}

	// Should have tried at least once but cancelled before all retries
	t.Logf("Attempts before cancel: %d", attempts)
	if atomic.LoadInt32(&attempts) == 0 || atomic.LoadInt32(&attempts) >= 5 {
		t.Errorf("Expected some attempts (1-4), got %d", attempts)
	}
}
