package concurrency

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewLimiter(t *testing.T) {
	tests := []struct {
		name     string
		n        int
		expected int
	}{
		{"positive n", 5, 5},
		{"zero n", 0, 1},
		{"negative n", -1, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := NewLimiter(tt.n)
			if l.MaxSlots() != tt.expected {
				t.Errorf("MaxSlots() = %d, want %d", l.MaxSlots(), tt.expected)
			}
		})
	}
}

func TestLimiterTryAcquire(t *testing.T) {
	l := NewLimiter(2)

	// Should succeed when slots available
	if !l.TryAcquire() {
		t.Error("TryAcquire() should return true when slot is available")
	}
	if !l.TryAcquire() {
		t.Error("TryAcquire() should return true when slot is available")
	}

	// Should fail when no slots available
	if l.TryAcquire() {
		t.Error("TryAcquire() should return false when no slots are available")
	}

	l.Release()
	l.Release()
}

func TestLimiterAcquireRelease(t *testing.T) {
	l := NewLimiter(1)

	// First acquire should succeed
	if err := l.Acquire(); err != nil {
		t.Errorf("First Acquire() failed: %v", err)
	}

	// Second acquire should block and timeout
	done := make(chan bool, 1)
	go func() {
		l.WithTimeout(100 * time.Millisecond)
		err := l.Acquire()
		done <- (err != nil)
	}()

	select {
	case success := <-done:
		if !success {
			t.Error("Second Acquire() should timeout")
		}
	case <-time.After(500 * time.Millisecond):
		t.Error("Acquire() should timeout within 500ms")
	}

	l.Release()
}

func TestLimiterExecute(t *testing.T) {
	l := NewLimiter(3)
	var maxConcurrent int32
	var currentConcurrent int32

	executions := 10
	var wg sync.WaitGroup
	wg.Add(executions)

	for i := 0; i < executions; i++ {
		go func() {
			defer wg.Done()
			l.Execute(func() error {
				current := atomic.AddInt32(&currentConcurrent, 1)
				if current > atomic.LoadInt32(&maxConcurrent) {
					atomic.StoreInt32(&maxConcurrent, current)
				}
				time.Sleep(10 * time.Millisecond)
				atomic.AddInt32(&currentConcurrent, -1)
				return nil
			})
		}()
	}

	wg.Wait()

	if maxConcurrent > 3 {
		t.Errorf("Max concurrent was %d, expected at most 3", maxConcurrent)
	}
}

func TestLimiterExecuteWithContext(t *testing.T) {
	l := NewLimiter(1)

	// Start a goroutine that holds the only slot
	go func() {
		l.ExecuteWithContext(context.Background(), func(ctx context.Context) error {
			time.Sleep(200 * time.Millisecond)
			return nil
		})
	}()

	time.Sleep(50 * time.Millisecond) // Let first goroutine acquire the slot

	// Try to execute with a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := l.ExecuteWithContext(ctx, func(ctx context.Context) error {
		return nil
	})

	if err == nil || err != context.Canceled {
		t.Errorf("ExecuteWithContext should return context.Canceled, got: %v", err)
	}
}

func TestLimiterGroup(t *testing.T) {
	lg := NewLimiterGroup(context.Background(), 2)
	var maxConcurrent int32
	var currentConcurrent int32

	executions := 10
	var wg sync.WaitGroup
	wg.Add(executions)

	for i := 0; i < executions; i++ {
		lg.Go(func(ctx context.Context) error {
			current := atomic.AddInt32(&currentConcurrent, 1)
			if current > atomic.LoadInt32(&maxConcurrent) {
				atomic.StoreInt32(&maxConcurrent, current)
			}
			time.Sleep(10 * time.Millisecond)
			atomic.AddInt32(&currentConcurrent, -1)
			wg.Done()
			return nil
		})
	}

	lg.Run()
	wg.Wait()

	if maxConcurrent > 2 {
		t.Errorf("Max concurrent was %d, expected at most 2", maxConcurrent)
	}
}

func TestLimiterGroupCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	lg := NewLimiterGroup(ctx, 1)

	errorsReceived := make(chan error, 1)

	lg.Go(func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			errorsReceived <- ctx.Err()
			return ctx.Err()
		case <-time.After(1 * time.Second):
			return nil
		}
	})

	time.Sleep(50 * time.Millisecond)
	cancel()
	lg.Run()

	select {
	case err := <-errorsReceived:
		if err != context.Canceled {
			t.Errorf("Expected context.Canceled, got: %v", err)
		}
	default:
		t.Error("Should have received cancellation error")
	}
}

func TestLimiterAvailableSlots(t *testing.T) {
	l := NewLimiter(3)

	if l.AvailableSlots() != 3 {
		t.Errorf("AvailableSlots() = %d, want 3", l.AvailableSlots())
	}

	l.Acquire()
	if l.AvailableSlots() != 2 {
		t.Errorf("AvailableSlots() = %d, want 2", l.AvailableSlots())
	}

	l.Acquire()
	l.Acquire()
	if l.AvailableSlots() != 0 {
		t.Errorf("AvailableSlots() = %d, want 0", l.AvailableSlots())
	}

	l.Release()
	l.Release()
	l.Release()
}

func TestLimiterWithTimeout(t *testing.T) {
	l := NewLimiter(1).WithTimeout(100 * time.Millisecond)

	l.Acquire()
	defer l.Release()

	err := l.Acquire()
	if err == nil {
		t.Error("Acquire should timeout")
	}
	if err != context.DeadlineExceeded {
		t.Errorf("Expected DeadlineExceeded, got: %v", err)
	}
}

func BenchmarkLimiterExecute(b *testing.B) {
	l := NewLimiter(10)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l.Execute(func() error {
			time.Sleep(time.Microsecond)
			return nil
		})
	}
}

func BenchmarkLimiterGroup(b *testing.B) {
	ctx := context.Background()
	lg := NewLimiterGroup(ctx, 10)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lg.Go(func(ctx context.Context) error {
			time.Sleep(time.Microsecond)
			return nil
		})
	}
	lg.Run()
}
