package concurrency

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

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
