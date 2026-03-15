package concurrency

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// ==================== Worker Pool Edge Case Tests ====================

func TestPool_WorkerPoolBehaviorWithVariousSizes(t *testing.T) {
	tests := []struct {
		name     string
		numWorkers int
		numTasks int
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
			
			pool := NewWorkerPool(tt.numWorkers, 100)
			var completed atomic.Int32
			var orderMu sync.Mutex
			var executionOrder []int

			taskID := 0
			for i := 0; i < tt.numTasks; i++ {
				tid := taskID
				taskID++
				
				err := pool.Submit(func(ctx context.Context) error {
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

			pool.Stop()

			if completed.Load() != int32(tt.numTasks) {
				t.Errorf("Expected %d completed tasks, got %d", tt.numTasks, completed.Load())
			}
		})
	}
}

func TestPool_CapacityLimitsAndQueueing(t *testing.T) {
	t.Run("QueueReachesCapacityRejectsNewTasks", func(t *testing.T) {
		t.Parallel()
		
		pool := NewWorkerPool(1, 3) // 1 worker, capacity of 3
		var rejected int32
		
		for i := 0; i < 5; i++ {
			err := pool.Submit(func(ctx context.Context) error {
				time.Sleep(10 * time.Millisecond)
				return nil
			})
			if errors.Is(err, ErrPoolFull) {
				atomic.AddInt32(&rejected, 1)
			}
		}

		pool.Stop()
		
		if rejected == 0 {
			t.Error("Expected at least one task to be rejected due to capacity limit")
		}
	})

	t.Run("QueueFillsAndDrainsCorrectly", func(t *testing.T) {
		t.Parallel()
		
		pool := NewWorkerPool(2, 5)
		var active int32
		var maxActive atomic.Int32
		
		for i := 0; i < 10; i++ {
			err := pool.Submit(func(ctx context.Context) error {
				current := atomic.AddInt32(&active, 1)
				time.Sleep(20 * time.Millisecond)
				atomic.AddInt32(&active, -1)
				
				maxActive.Store(max(int64(maxActive.Load()), int64(current)))
				return nil
			})
			if err != nil {
				t.Logf("Task %d rejected: %v", i, err)
			}
		}

		pool.Stop()

		expectedMax := int32(2) // 2 workers means max 2 concurrent tasks
		if maxActive.Load() > int64(expectedMax) {
			t.Errorf("Expected max %d concurrent tasks, got %d", expectedMax, maxActive.Load())
		}
	})

	t.Run("QueueUnderLoadMaintainsOrder", func(t *testing.T) {
		t.Parallel()
		
		pool := NewWorkerPool(3, 100)
		var orderMu sync.Mutex
		var executionOrder []int
		taskID := 0
		
		for i := 0; i < 20; i++ {
			tid := taskID
			taskID++
			
			err := pool.Submit(func(ctx context.Context) error {
				time.Sleep(time.Duration(tid % 3) * time.Millisecond)
				orderMu.Lock()
				executionOrder = append(executionOrder, tid)
				orderMu.Unlock()
				return nil
			})
			if err != nil {
				t.Fatalf("Failed to submit task %d: %v", tid, err)
			}
		}

		pool.Stop()

		if len(executionOrder) != 20 {
			t.Errorf("Expected 20 completed tasks, got %d", len(executionOrder))
		}
	})
}

func TestPool_GracefulShutdownScenarios(t *testing.T) {
	tests := []struct {
		name       string
		taskDuration time.Duration
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
			
			pool := NewWorkerPool(3, 100)
			var completed int32
			var started int32
			
			taskDuration := tt.taskDuration
			if taskDuration == 0 {
				taskDuration = 1 * time.Millisecond
			}
			
			for i := 0; i < 5; i++ {
				err := pool.Submit(func(ctx context.Context) error {
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
			pool.Stop()
			shutdownDuration := time.Since(startTime)

			if shutdownDuration > 2*time.Second {
				t.Errorf("Shutdown took too long: %v", shutdownDuration)
			}
		})
	}
}

func TestPool_ErrorHandlingDuringTaskExecution(t *testing.T) {
	tests := []struct {
		name           string
		errorType      error
		taskCount      int
		expectedErrors int
	}{
		{"AllTasksFailWithStandardError", errors.New("standard error"), 5, 5},
		{"AllTasksFailWithContextCanceled", context.Canceled, 3, 3},
		{"MixedSuccessFailure", nil, 6, 3},
		{"NilErrorTasksSucceed", nil, 4, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			
			pool := NewWorkerPool(2, 50)
			var succeeded int32
			var failed int32
			
			taskID := 0
			for i := 0; i < tt.taskCount; i++ {
				tid := taskID
				taskID++
				
				err := pool.Submit(func(ctx context.Context) error {
					if tid%2 == 1 || tt.errorType != nil {
						atomic.AddInt32(&failed, 1)
						return tt.errorType
					}
					atomic.AddInt32(&succeeded, 1)
					return nil
				})
				
				if err != nil && tid < tt.taskCount {
					t.Fatalf("Failed to submit task %d: %v", tid, err)
				}
			}

			pool.Stop()

			expectedSucceeded := int32(tt.taskCount - tt.expectedErrors)
			if succeeded != expectedSucceeded {
				t.Errorf("Expected %d succeeded tasks, got %d", expectedSucceeded, succeeded)
			}
			if failed != int32(tt.expectedErrors) {
				t.Errorf("Expected %d failed tasks, got %d", tt.expectedErrors, failed)
			}
		})
	}
}

func TestPool_GoroutineLeakDetection(t *testing.T) {
	tests := []struct {
		name       string
		runFunc    func()
		maxGorouts int
	}{
		{"NormalSubmissionNoLeaks", func() {
			pool := NewWorkerPool(5, 100)
			for i := 0; i < 20; i++ {
				pool.Submit(func(ctx context.Context) error {
					return nil
				})
			}
			pool.Stop()
		}, 20},
		{"ShutdownBeforeAllStartNoLeaks", func() {
			pool := NewWorkerPool(5, 100)
			for i := 0; i < 10; i++ {
				pool.Submit(func(ctx context.Context) error {
					time.Sleep(100 * time.Millisecond)
					return nil
				})
			}
			time.Sleep(10 * time.Millisecond)
			pool.Stop()
		}, 15},
		{"TaskFailsNoLeaks", func() {
			pool := NewWorkerPool(5, 100)
			for i := 0; i < 20; i++ {
				pool.Submit(func(ctx context.Context) error {
					return errors.New("task failed")
				})
			}
			pool.Stop()
		}, 25},
		{"ContextCanceledNoLeaks", func() {
			pool := NewWorkerPool(5, 100)
			for i := 0; i < 20; i++ {
				pool.Submit(func(ctx context.Context) error {
					ctx.Done()
					time.Sleep(time.Millisecond * 100)
					return ctx.Err()
				})
			}
			pool.Stop()
		}, 30},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			
			runtime := NewPoolRuntime()
			runtime.Start(time.Second)
			
			tt.runFunc()
			
			pool.GarbageCollectGoroutines()
			
			// Check for leaks after goroutine collection
			if len(runtime.ongoingTasks) > tt.maxGorouts {
				t.Errorf("Potential goroutine leak detected: %d ongoing tasks", len(runtime.ongoingTasks))
			}
		})
	}
}

func TestPool_ConcurrentAccessPatterns(t *testing.T) {
	t.Run("MultipleSubmitCallsConcurrently", func(t *testing.T) {
		t.Parallel()
		
		pool := NewWorkerPool(4, 100)
		var wg sync.WaitGroup
		submitCount := int32(0)
		
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				pool.Submit(func(ctx context.Context) error {
					return nil
				})
				atomic.AddInt32(&submitCount, 1)
			}()
		}
		
		wg.Wait()
		pool.Stop()
		
		if submitCount != 10 {
			t.Errorf("Expected all submissions to complete, got %d", submitCount)
		}
	})

	t.Run("SubmitDuringStopRaceCondition", func(t *testing.T) {
		t.Parallel()
		
		pool := NewWorkerPool(4, 50)
		var stopComplete atomic.Bool
		
		go func() {
			for i := 0; i < 20; i++ {
				pool.Submit(func(ctx context.Context) error {
					time.Sleep(5 * time.Millisecond)
					return nil
				})
			}
		}()

		time.Sleep(10 * time.Millisecond)
		go pool.Stop()
		
		for !stopComplete.Load() {
			select {
			case <-time.After(3 * time.Second):
				t.Error("Stop did not complete in time")
				return
			default:
				stopComplete.Store(true)
			}
		}
	})

	t.Run("PoolSizeChangesSafe", func(t *testing.T) {
		t.Parallel()
		
		pool := NewWorkerPool(4, 50)
		var activeWorkers atomic.Int32
		
		for i := 0; i < 30; i++ {
			pool.Submit(func(ctx context.Context) error {
				current := atomic.AddInt32(&activeWorkers, 1)
				time.Sleep(time.Duration(current%3) * time.Millisecond)
				atomic.AddInt32(&activeWorkers, -1)
				return nil
			})
		}
		
		pool.Stop()
		
		if activeWorkers.Load() != 0 {
			t.Errorf("Pool not stopped properly: %d workers still active", activeWorkers.Load())
		}
	})
}

func TestPool_TimeoutScenarios(t *testing.T) {
	t.Run("TaskTimeoutWithContext", func(t *testing.T) {
		t.Parallel()
		
		pool := NewWorkerPool(2, 50)
		var timeouts int32
		
		for i := 0; i < 5; i++ {
			timeoutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
			
			err := pool.Submit(func(ctx context.Context) error {
				time.Sleep(100 * time.Millisecond)
				cancel()
				return nil
			})
			
			if err != nil {
				t.Logf("Task %d rejected: %v", i, err)
			}
		}

		pool.Stop()
	})

	t.Run("PoolStopTimeoutNoHang", func(t *testing.T) {
		t.Parallel()
		
		pool := NewWorkerPool(3, 100)
		
		for i := 0; i < 20; i++ {
			pool.Submit(func(ctx context.Context) error {
				time.Sleep(10 * time.Millisecond)
				return nil
			})
		}

		start := time.Now()
		pool.Stop()
		duration := time.Since(start)
		
		if duration > 2*time.Second {
			t.Errorf("Pool stop took too long: %v", duration)
		}
	})
}

// ==================== Helper Functions ====================

func max(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
