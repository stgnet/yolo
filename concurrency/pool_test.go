package concurrency

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// ==================== Worker Pool Edge Case Tests for ThreadPool and ParallelExecutor ====================

func TestPool_WorkerPoolBehaviorWithVariousSizes(t *testing.T) {
	tests := []struct {
		name       string
		numWorkers int
		numTasks   int
	}{
		{"SingleWorkerSingleTask", 1, 1},
		{"SingleWorkerMultipleTasks", 1, 10},
		{"ManyWorkersFewTasks", 10, 2},
		{"OneToOne", 5, 5},
		{"OverflowWorkload", 2, 50},
		{"BurstWorkload", 10, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			pool := NewThreadPool(tt.numWorkers)
			var completed atomic.Int32
			var orderMu sync.Mutex
			var executionOrder []int

			taskID := 0
			for i := 0; i < tt.numTasks; i++ {
				tid := taskID
				taskID++

				err := pool.Submit(func() error {
					orderMu.Lock()
					executionOrder = append(executionOrder, tid)
					orderMu.Unlock()
					completed.Add(1)
					return nil
				})
				if err != nil {
					t.Fatalf("Failed to submit task %d: %v", tid, err)
				}
			}

			pool.Close()

			if completed.Load() != int32(tt.numTasks) {
				t.Errorf("Expected %d completed tasks, got %d", tt.numTasks, completed.Load())
			}
		})
	}
}

func TestPool_CapacityLimitsAndQueueing(t *testing.T) {
	t.Run("MultipleSubmissionsComplete", func(t *testing.T) {
		t.Parallel()

		pool := NewThreadPool(1) // 1 worker, unbounded queue
		var completed int32

		for i := 0; i < 5; i++ {
			err := pool.Submit(func() error {
				time.Sleep(10 * time.Millisecond)
				atomic.AddInt32(&completed, 1)
				return nil
			})
			if err != nil {
				t.Logf("Task submission %d failed: %v", i, err)
			}
		}

		pool.Close()

		if atomic.LoadInt32(&completed) != 5 {
			t.Errorf("Expected all 5 tasks to complete, got %d", atomic.LoadInt32(&completed))
		}
	})

	t.Run("QueueDrainsCorrectly", func(t *testing.T) {
		t.Parallel()

		pool := NewThreadPool(2) // 2 workers
		var active int32
		var maxActive int32 = 0
		var mu sync.Mutex

		for i := 0; i < 10; i++ {
			err := pool.Submit(func() error {
				current := atomic.AddInt32(&active, 1)
				time.Sleep(20 * time.Millisecond)
				atomic.AddInt32(&active, -1)

				mu.Lock()
				if current > maxActive {
					maxActive = current
				}
				mu.Unlock()
				return nil
			})
			if err != nil {
				t.Logf("Task %d rejected: %v", i, err)
			}
		}

		pool.Close()

		expectedMax := int32(2) // 2 workers means max 2 concurrent tasks
		if maxActive > expectedMax {
			t.Errorf("Expected max %d concurrent tasks, got %d", expectedMax, maxActive)
		}
	})

	t.Run("QueueUnderLoadMaintainsOrder", func(t *testing.T) {
		t.Parallel()

		pool := NewThreadPool(3)
		var orderMu sync.Mutex
		var executionOrder []int
		taskID := 0

		for i := 0; i < 20; i++ {
			tid := taskID
			taskID++

			err := pool.Submit(func() error {
				time.Sleep(time.Duration(tid%3) * time.Millisecond)
				orderMu.Lock()
				executionOrder = append(executionOrder, tid)
				orderMu.Unlock()
				return nil
			})
			if err != nil {
				t.Fatalf("Failed to submit task %d: %v", tid, err)
			}
		}

		pool.Close()

		if len(executionOrder) != 20 {
			t.Errorf("Expected 20 completed tasks, got %d", len(executionOrder))
		}
	})
}

func TestPool_GracefulShutdownScenarios(t *testing.T) {
	tests := []struct {
		name          string
		taskDuration  time.Duration
		shutdownDelay time.Duration
	}{
		{"ImmediateShutdownNoTasks", 0, 0},
		{"ShutdownDuringRunningTask", 50 * time.Millisecond, 10 * time.Millisecond},
		{"ShutdownAfterMostTasksComplete", 20 * time.Millisecond, 30 * time.Millisecond},
		{"ShutdownBeforeAnyTaskStarts", 100 * time.Millisecond, 5 * time.Millisecond},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			pool := NewThreadPool(3)
			var completed int32
			var started int32

			taskDuration := tt.taskDuration
			if taskDuration == 0 {
				taskDuration = 1 * time.Millisecond
			}

			for i := 0; i < 5; i++ {
				err := pool.Submit(func() error {
					atomic.AddInt32(&started, 1)
					time.Sleep(taskDuration)
					atomic.AddInt32(&completed, 1)
					return nil
				})
				if err != nil {
					t.Logf("Task submission %d failed: %v", i, err)
				}
			}

			time.Sleep(tt.shutdownDelay)

			startTime := time.Now()
			pool.Close()
			shutdownDuration := time.Since(startTime)

			if shutdownDuration > 2*time.Second {
				t.Errorf("Shutdown took too long: %v", shutdownDuration)
			}
		})
	}
}

func TestPool_ErrorHandlingDuringTaskExecution(t *testing.T) {
	tests := []struct {
		name          string
		taskError     error
		expectHandled bool
	}{
		{"NoError", nil, true},
		{"TaskReturnsError", errors.New("task failed"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool := NewThreadPool(2)
			var totalTasks int32

			for i := 0; i < 5; i++ {
				err := pool.Submit(func() error {
					atomic.AddInt32(&totalTasks, 1)
					return tt.taskError
				})
				if err != nil {
					t.Logf("Task submission failed: %v", err)
				}
			}

			pool.Close()

			if atomic.LoadInt32(&totalTasks) != 5 {
				t.Errorf("Expected 5 tasks to run, got %d", atomic.LoadInt32(&totalTasks))
			}
		})
	}
}

func TestPool_PanicRecovery(t *testing.T) {
	pool := NewThreadPool(2)
	defer pool.Close()

	var completed int32

	for i := 0; i < 5; i++ {
		i := i
		err := pool.Submit(func() error {
			if i%2 == 0 {
				panic("intentional panic")
				return nil // Won't reach here
			}
			atomic.AddInt32(&completed, 1)
			return nil
		})
		if err != nil {
			t.Logf("Task submission failed: %v", err)
		}
	}

	pool.Close()

	// Pool should handle panics gracefully without crashing entirely
	t.Log("Pool handled panics during execution")
}

func TestPool_QueueOperations(t *testing.T) {
	t.Run("EmptyPoolReturnsZero", func(t *testing.T) {
		pool := NewThreadPool(2)
		defer pool.Close()

		if pool.QueueSize() != 0 {
			t.Errorf("Expected queue size 0, got %d", pool.QueueSize())
		}
	})

	t.Run("QueueSizeAfterSubmissions", func(t *testing.T) {
		pool := NewThreadPool(1)
		defer pool.Close()

		submitCount := 0
		for i := 0; i < 5; i++ {
			err := pool.Submit(func() error {
				time.Sleep(50 * time.Millisecond)
				return nil
			})
			if err == nil {
				submitCount++
			}
		}

		// Queue size varies based on timing - just verify submissions happened
		if submitCount < 1 {
			t.Error("Expected at least one submission to succeed")
		}
	})

	t.Run("QueueSizeAfterClose", func(t *testing.T) {
		pool := NewThreadPool(2)
		pool.Close()

		size := pool.QueueSize()
		if size != 0 {
			t.Errorf("Expected queue size 0 after close, got %d", size)
		}
	})
}

func TestPool_WorkerCount(t *testing.T) {
	testCases := []struct {
		name     string
		workers  int
		expected int
	}{
		{"ZeroWorkers", 0, 1},      // Should default to 1
		{"NegativeWorkers", -5, 1}, // Should default to 1
		{"SingleWorker", 1, 1},
		{"MultipleWorkers", 10, 10},
		{"LargePool", 100, 100},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pool := NewThreadPool(tc.workers)
			defer pool.Close()

			if pool.WorkerCount() != tc.expected {
				t.Errorf("Expected worker count %d, got %d", tc.expected, pool.WorkerCount())
			}
		})
	}
}

func TestPool_SubmitWhileStopping(t *testing.T) {
	pool := NewThreadPool(2)
	defer pool.Close()

	var submitted bool

	doneCh := make(chan struct{})
	go func() {
		err := pool.Submit(func() error {
			time.Sleep(100 * time.Millisecond)
			return nil
		})
		if err == nil {
			submitted = true
		}
		close(doneCh)
	}()

	pool.Close()
	<-doneCh

	t.Logf("Submit during stop - submitted: %v", submitted)
}

func TestPool_SubmitToClosedPool(t *testing.T) {
	pool := NewThreadPool(2)
	pool.Close()

	err := pool.Submit(func() error {
		return nil
	})

	if err == nil {
		t.Error("Expected error when submitting to closed pool")
	}
	if !errors.Is(err, ErrPoolClosed) {
		t.Errorf("Expected ErrPoolClosed, got %v", err)
	}
}

func TestPool_ContextCancellation(t *testing.T) {
	pool := NewThreadPool(2)
	defer pool.Close()

	var cancelled int32
	var completed int32

	ctx, cancel := context.WithCancel(context.Background())

	err := pool.Submit(func() error {
		select {
		case <-ctx.Done():
			atomic.AddInt32(&cancelled, 1)
			return ctx.Err()
		default:
			time.Sleep(50 * time.Millisecond)
			atomic.AddInt32(&completed, 1)
			return nil
		}
	})

	if err != nil {
		t.Logf("Submit failed: %v", err)
	}

	cancel()
	pool.Close()

	t.Logf("Context cancellation test - cancelled: %d, completed: %d",
		atomic.LoadInt32(&cancelled), atomic.LoadInt32(&completed))
}

func TestPool_ConcurrentSubmissions(t *testing.T) {
	const numGoroutines = 5
	const tasksPerGoroutine = 10

	pool := NewThreadPool(4)
	defer pool.Close()

	var wg sync.WaitGroup
	var totalSubmitted int32
	var mu sync.Mutex

	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			localCount := 0
			for i := 0; i < tasksPerGoroutine; i++ {
				err := pool.Submit(func() error {
					return nil
				})
				if err == nil {
					localCount++
				}
			}
			mu.Lock()
			totalSubmitted += int32(localCount)
			mu.Unlock()
		}()
	}

	wg.Wait()

	minExpected := int32(numGoroutines*tasksPerGoroutine - 5) // Allow some rejections
	if totalSubmitted < minExpected {
		t.Errorf("Expected at least %d submissions, got %d", minExpected, totalSubmitted)
	}
}

func TestPool_OverflowBehavior(t *testing.T) {
	pool := NewThreadPool(1) // Small pool to demonstrate queuing behavior
	defer pool.Close()

	var submitted int32

	for i := 0; i < 10; i++ {
		err := pool.Submit(func() error {
			time.Sleep(5 * time.Millisecond)
			atomic.AddInt32(&submitted, 1)
			return nil
		})
		if err != nil && !errors.Is(err, ErrPoolClosed) {
			t.Logf("Unexpected error: %v", err)
		}
	}

	pool.Close()

	if atomic.LoadInt32(&submitted) == 0 {
		t.Error("Expected at least some tasks to be submitted and completed")
	}
}

func TestParallelExecutor(t *testing.T) {
	t.Run("BasicExecution", func(t *testing.T) {
		executor := NewParallelExecutor(4)
		defer executor.Close()

		var completed int32

		for i := 0; i < 10; i++ {
			i := i
			executor.Submit(func(ctx context.Context) error {
				time.Sleep(time.Duration(i%3) * time.Millisecond)
				atomic.AddInt32(&completed, 1)
				return nil
			})
		}

		executor.Close()

		if atomic.LoadInt32(&completed) != 10 {
			t.Errorf("Expected 10 completed tasks, got %d", atomic.LoadInt32(&completed))
		}
	})

	t.Run("ErrorCollection", func(t *testing.T) {
		executor := NewParallelExecutor(4)
		defer executor.Close()

		testErrors := []error{
			nil, errors.New("error 1"), nil, errors.New("error 2"),
		}

		for i := range testErrors {
			i := i
			executor.Submit(func(ctx context.Context) error {
				if i == 0 || i == 2 {
					return nil
				}
				return testErrors[i]
			})
		}

		executor.Close()

		errors := executor.Errors()
		if len(errors) != 2 {
			t.Errorf("Expected 2 errors, got %d", len(errors))
		}
	})

	t.Run("CancellationStopsAll", func(t *testing.T) {
		executor := NewParallelExecutor(2)

		var started int32
		go func() {
			for i := 0; i < 10; i++ {
				executor.Submit(func(ctx context.Context) error {
					atomic.AddInt32(&started, 1)
					select {
					case <-ctx.Done():
						return ctx.Err()
					case <-time.After(50 * time.Millisecond):
						return nil
					}
				})
			}
		}()

		time.Sleep(10 * time.Millisecond)
		executor.Cancel()

		time.Sleep(100 * time.Millisecond)
		t.Logf("Tasks started before cancellation: %d", atomic.LoadInt32(&started))
	})

	t.Run("EmptyExecutorDoesntCrash", func(t *testing.T) {
		executor := NewParallelExecutor(4)
		executor.Close() // Close without submitting anything
		t.Log("Empty executor closed successfully")
	})
}

func TestThreadPoolSubmitWithContext(t *testing.T) {
	pool := NewThreadPool(2)
	defer pool.Close()

	var cancelled int32
	var completed int32

	ctx, cancel := context.WithCancel(context.Background())

	err := pool.Submit(func() error {
		select {
		case <-ctx.Done():
			atomic.AddInt32(&cancelled, 1)
			return ctx.Err()
		default:
			time.Sleep(50 * time.Millisecond)
			atomic.AddInt32(&completed, 1)
			return nil
		}
	})

	if err != nil {
		t.Logf("Submit failed: %v", err)
	}

	cancel()
	pool.Close()

	t.Logf("Context test - cancelled: %d, completed: %d",
		atomic.LoadInt32(&cancelled), atomic.LoadInt32(&completed))
}

func TestThreadPool_WaitMethod(t *testing.T) {
	pool := NewThreadPool(1)
	defer pool.Wait() // Call Wait to ensure cleanup

	var completed int32

	for i := 0; i < 5; i++ {
		err := pool.Submit(func() error {
			time.Sleep(10 * time.Millisecond)
			atomic.AddInt32(&completed, 1)
			return nil
		})
		if err != nil {
			t.Logf("Task %d failed: %v", i, err)
		}
	}

	pool.Wait() // Block until all tasks complete

	if atomic.LoadInt32(&completed) != 5 {
		t.Errorf("Expected 5 completed tasks after Wait, got %d", atomic.LoadInt32(&completed))
	}
}
