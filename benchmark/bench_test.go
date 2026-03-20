package benchmark

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/scottstg/yolo/caching"
	"github.com/scottstg/yolo/concurrency"
)

// BenchmarkSearchCache stores performance metrics
func BenchmarkSearchCacheMiss(b *testing.B) {
	cache := caching.NewCaches(30 * time.Minute)
	query := "test query for benchmarking purposes"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Simulate cache miss (new query each time)
		key := query + "-miss-" + string(rune(i))
		cache.CheckAndFetch(key, func() interface{} {
			return "result"
		})
	}
}

func BenchmarkSearchCacheHit(b *testing.B) {
	cache := caching.NewCaches(30 * time.Minute)
	query := "test query for benchmarking purposes"

	// Pre-populate cache
	result := "cached result data with some content to make it more realistic"
	cache.Set(query, &caching.CachedResult{Content: result, CreatedAt: time.Now()})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.CheckAndFetch(query, func() interface{} {
			return "should not be called"
		})
	}
}

func BenchmarkSearchCacheEviction(b *testing.B) {
	cache := caching.NewCaches(5 * time.Minute) // Short TTL for testing

	// Add expired entries
	for i := 0; i < 100; i++ {
		key := "expired-key-" + string(rune(i))
		cache.Set(key, &caching.CachedResult{Content: "old data", CreatedAt: time.Now().Add(-1 * time.Hour)})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// This should trigger cleanup
		cache.GetStats()
	}
}

// Benchmark for the entire yolo package
func TestBenchmarks(t *testing.T) {
	t.Log("Running all benchmarks - use 'go test -bench=. ./...' to execute")
}

// ==================== Concurrency Benchmarks ====================

// BenchmarkThreadPoolSingleWorker benchmarks thread pool submission with a single worker
func BenchmarkThreadPoolSingleWorker(b *testing.B) {
	pool := concurrency.NewThreadPool(1)
	defer pool.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pool.Submit(func() error {
			return nil
		})
	}
	pool.Wait()
}

// BenchmarkThreadPoolMultipleWorkers benchmarks thread pool submission with multiple workers
func BenchmarkThreadPoolMultipleWorkers(b *testing.B) {
	pool := concurrency.NewThreadPool(4)
	defer pool.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pool.Submit(func() error {
			return nil
		})
	}
	pool.Wait()
}

// BenchmarkThreadPoolWithWork benchmarks thread pool with actual atomic operations
func BenchmarkThreadPoolWithWork(b *testing.B) {
	pool := concurrency.NewThreadPool(4)
	defer pool.Close()

	var counter int64
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pool.Submit(func() error {
			atomic.AddInt64(&counter, 1)
			return nil
		})
	}
	pool.Wait()
}

// BenchmarkThreadPoolPanicRecovery benchmarks overhead of panic recovery in thread pool
func BenchmarkThreadPoolPanicRecovery(b *testing.B) {
	pool := concurrency.NewThreadPool(2)
	defer pool.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pool.Submit(func() error {
			if i%2 == 0 {
				panic("test panic")
			}
			return nil
		})
	}
	pool.Wait()
}

// BenchmarkParallelGroup benchmarks parallel execution group creation and execution
func BenchmarkParallelGroup(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		g := concurrency.NewGroup(context.Background())
		var done int32

		for j := 0; j < 10; j++ {
			g.Go(func(ctx context.Context) error {
				atomic.AddInt32(&done, 1)
				return nil
			})
		}
		g.Run()
	}
}

// BenchmarkParallelGroupWithErrors benchmarks group execution with alternating errors
func BenchmarkParallelGroupWithErrors(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		g := concurrency.NewGroup(context.Background())

		for j := 0; j < 10; j++ {
			g.Go(func(ctx context.Context) error {
				if j%2 == 0 {
					return ctx.Err()
				}
				return nil
			})
		}
		g.Run()
		_ = g.Errors()
	}
}

// BenchmarkBarrier benchmarks synchronization barrier for goroutine coordination
func BenchmarkBarrier(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		barrier := concurrency.NewBarrier(4)
		var done int32

		for j := 0; j < 4; j++ {
			go func() {
				barrier.Wait()
				atomic.AddInt32(&done, 1)
			}()
		}

		// Wait for all goroutines to complete
		for atomic.LoadInt32(&done) != 4 {
			time.Sleep(1 * time.Millisecond)
		}
	}
}

// BenchmarkLimitedConcurrency benchmarks limited concurrency execution pattern
func BenchmarkLimitedConcurrency(b *testing.B) {
	ctx := context.Background()

	jobs := make([]func(context.Context) error, 20)
	for i := range jobs {
		jobs[i] = func(ctx context.Context) error {
			return nil
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		concurrency.LimitedConcurrency(ctx, 4, jobs)
	}
}

// BenchmarkRetryWithBackoffSuccess benchmarks retry with backoff when operation succeeds immediately
func BenchmarkRetryWithBackoffSuccess(b *testing.B) {
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = concurrency.RetryWithBackoff(ctx, 3, 1*time.Millisecond, func(ctx context.Context) error {
			return nil // Success on first try
		})
	}
}

// BenchmarkRetryWithBackoffFailures benchmarks retry with backoff when operation always fails
func BenchmarkRetryWithBackoffFailures(b *testing.B) {
	ctx := context.Background()
	err := errors.New("persistent failure")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = concurrency.RetryWithBackoff(ctx, 3, 1*time.Millisecond, func(ctx context.Context) error {
			return err // Always fails
		})
	}
}
