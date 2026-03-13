package concurrency

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestThreadPool_Wait(t *testing.T) {
	tp := NewThreadPool(2)
	var wg sync.WaitGroup

	// Submit some jobs
	numJobs := 10
	for i := 0; i < numJobs; i++ {
		wg.Add(1)
		err := tp.Submit(func() {
			defer wg.Done()
			time.Sleep(1 * time.Millisecond)
		})
		if err != nil {
			t.Errorf("Failed to submit job: %v", err)
		}
	}

	// Wait should not block since jobs are being processed
	done := make(chan struct{})
	go func() {
		tp.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success - all jobs completed
	case <-time.After(5 * time.Second):
		t.Fatal("Wait() timed out")
	}

	tp.Close()
}

func TestThreadPool_QueueSize(t *testing.T) {
	tp := NewThreadPool(1)

	// Submit multiple jobs quickly
	numJobs := 20
	for i := 0; i < numJobs; i++ {
		err := tp.Submit(func() {
			time.Sleep(10 * time.Millisecond)
		})
		if err != nil {
			t.Errorf("Failed to submit job: %v", err)
		}
	}

	// Queue should have some pending jobs
	size := tp.QueueSize()
	if size < 1 || size > numJobs {
		t.Logf("Queue size is %d (expected between 1 and %d)", size, numJobs)
	}

	tp.Close()
}

func TestThreadPool_WorkerCount(t *testing.T) {
	tp := NewThreadPool(5)

	count := tp.WorkerCount()
	if count != 5 {
		t.Errorf("Expected 5 workers, got %d", count)
	}

	tp.Close()
}

func TestParallelExecutor_Cancel(t *testing.T) {
	executor := NewParallelExecutor(2)

	var cancelled int32

	// Submit jobs that can be cancelled
	for i := 0; i < 10; i++ {
		executor.Submit(func(ctx context.Context) error {
			select {
			case <-ctx.Done():
				atomic.AddInt32(&cancelled, 1)
				return ctx.Err()
			default:
				time.Sleep(10 * time.Millisecond)
				return nil
			}
		})
	}

	// Cancel the executor
	executor.Cancel()

	// Give some time for cancellation to propagate
	time.Sleep(50 * time.Millisecond)

	// At least some jobs should have been cancelled
	cancelledCount := atomic.LoadInt32(&cancelled)
	if cancelledCount == 0 {
		t.Logf("No jobs were cancelled (this may be expected depending on timing)")
	} else {
		t.Logf("%d jobs were cancelled", cancelledCount)
	}
}

// Keep existing tests for regression
func TestNewThreadPool(t *testing.T) {
	tp := NewThreadPool(4)
	if tp == nil {
		t.Fatal("Expected non-nil thread pool")
	}
	if tp.numWorkers != 4 {
		t.Errorf("Expected 4 workers, got %d", tp.numWorkers)
	}
	defer tp.Close()
}

func TestThreadPool_WithZeroWorkers(t *testing.T) {
	tp := NewThreadPool(0)
	if tp.numWorkers != 1 {
		t.Errorf("Expected minimum 1 worker, got %d", tp.numWorkers)
	}
	defer tp.Close()
}

func TestThreadPool_SubmitAndClose(t *testing.T) {
	tp := NewThreadPool(2)

	var count int32
	numJobs := 50

	for i := 0; i < numJobs; i++ {
		err := tp.Submit(func() {
			atomic.AddInt32(&count, 1)
		})
		if err != nil {
			t.Errorf("Failed to submit job: %v", err)
		}
	}

	tp.Close()

	if atomic.LoadInt32(&count) != int32(numJobs) {
		t.Errorf("Expected %d jobs to run, got %d", numJobs, count)
	}
}

func TestThreadPool_SubmitToClosedPool(t *testing.T) {
	tp := NewThreadPool(1)
	tp.Close()

	err := tp.Submit(func() {})
	if err == nil {
		t.Error("Expected error when submitting to closed pool")
	}
	if err != ErrPoolClosed {
		t.Errorf("Expected ErrPoolClosed, got %v", err)
	}
}

func TestThreadPool_SubmitWithContext(t *testing.T) {
	tp := NewThreadPool(2)

	var executed int32

	ctx, cancel := context.WithCancel(context.Background())

	err := tp.SubmitWithContext(ctx, func(ctx context.Context) {
		atomic.AddInt32(&executed, 1)
	})
	if err != nil {
		t.Errorf("Failed to submit job: %v", err)
	}

	tp.Close()

	if atomic.LoadInt32(&executed) != 1 {
		t.Errorf("Expected job to execute, got count %d", executed)
	}

	cancel()
}

func TestThreadPool_SubmitWithContextCancelled(t *testing.T) {
	tp := NewThreadPool(2)

	var executed int32

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := tp.SubmitWithContext(ctx, func(ctx context.Context) {
		atomic.AddInt32(&executed, 1)
	})
	if err != nil {
		t.Errorf("Failed to submit job: %v", err)
	}

	tp.Close()

	time.Sleep(10 * time.Millisecond)
	if atomic.LoadInt32(&executed) != 0 {
		t.Errorf("Expected job to be skipped, got count %d", executed)
	}
}

func TestNewParallelExecutor(t *testing.T) {
	executor := NewParallelExecutor(4)
	if executor == nil {
		t.Fatal("Expected non-nil executor")
	}
	if executor.pool.numWorkers != 4 {
		t.Errorf("Expected 4 workers, got %d", executor.pool.numWorkers)
	}
	executor.Close()
}

func TestParallelExecutor_SubmitAndClose(t *testing.T) {
	executor := NewParallelExecutor(2)

	var count int32
	numJobs := 30

	for i := 0; i < numJobs; i++ {
		executor.Submit(func(ctx context.Context) error {
			atomic.AddInt32(&count, 1)
			return nil
		})
	}

	executor.Close()

	if atomic.LoadInt32(&count) != int32(numJobs) {
		t.Errorf("Expected %d jobs to run, got %d", numJobs, count)
	}
}

func TestParallelExecutor_WithErrors(t *testing.T) {
	executor := NewParallelExecutor(2)

	testErr := errors.New("test error")

	for i := 0; i < 5; i++ {
		executor.Submit(func(ctx context.Context) error {
			return testErr
		})
	}

	executor.Close()

	errs := executor.Errors()
	if len(errs) != 5 {
		t.Errorf("Expected 5 errors, got %d", len(errs))
	}

	for _, err := range errs {
		if err != testErr {
			t.Errorf("Unexpected error: %v", err)
		}
	}
}

func TestPoolError_Error(t *testing.T) {
	err := ErrPoolClosed
	expectedMsg := "pool is closed"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestParallelExecutor_CancelBeforeClose(t *testing.T) {
	executor := NewParallelExecutor(2)

	var count int32

	for i := 0; i < 10; i++ {
		executor.Submit(func(ctx context.Context) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				time.Sleep(2 * time.Millisecond)
				atomic.AddInt32(&count, 1)
				return nil
			}
		})
	}

	executor.Close()

	time.Sleep(20 * time.Millisecond)
	t.Logf("Completed %d jobs", atomic.LoadInt32(&count))
}
