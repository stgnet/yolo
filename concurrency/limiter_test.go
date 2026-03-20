package concurrency

import (
	"context"
	"errors"
	"sync"
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
		{"n equals 1", 1, 1},
		{"n equals 0", 0, 1}, // defaults to 1
		{"negative n", -5, 1}, // defaults to 1
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := NewLimiter(tt.n)
			if l == nil {
				t.Fatal("Expected non-nil limiter")
			}
			if l.MaxSlots() != tt.expected {
				t.Errorf("Expected MaxSlots() = %d, got %d", tt.expected, l.MaxSlots())
			}
		})
	}
}

func TestWithTimeout(t *testing.T) {
	l := NewLimiter(5)
	
	if l.timeout != 30*time.Second {
		t.Errorf("Expected default timeout 30s, got %v", l.timeout)
	}
	
	timeoutLimiter := l.WithTimeout(2 * time.Second)
	
	if timeoutLimiter != l {
		t.Error("Expected WithTimeout to return same limiter instance")
	}
	if l.timeout != 2*time.Second {
		t.Errorf("Expected timeout 2s, got %v", l.timeout)
	}
}

func TestTryAcquire(t *testing.T) {
	l := NewLimiter(1)

	// First acquire should succeed
	if !l.TryAcquire() {
		t.Error("Expected first TryAcquire to succeed")
	}

	// Second acquire should fail
	if l.TryAcquire() {
		t.Error("Expected second TryAcquire to fail when slot is in use")
	}

	l.Release()

	// Third acquire should succeed again
	if !l.TryAcquire() {
		t.Error("Expected TryAcquire to succeed after release")
	}
}

func TestTryAcquireMultipleSlots(t *testing.T) {
	l := NewLimiter(2)

	// First two should succeed
	if !l.TryAcquire() {
		t.Error("Expected first TryAcquire to succeed")
	}
	if !l.TryAcquire() {
		t.Error("Expected second TryAcquire to succeed")
	}

	// Third should fail
	if l.TryAcquire() {
		t.Error("Expected third TryAcquire to fail when all slots are in use")
	}

	l.Release()
	
	// Should work again
	if !l.TryAcquire() {
		t.Error("Expected TryAcquire to succeed after release")
	}
}

func TestAcquireTimeout(t *testing.T) {
	l := NewLimiter(1)

	// Hold the slot
	if err := l.Acquire(); err != nil {
		t.Fatalf("Failed to acquire first slot: %v", err)
	}

	// Set a short timeout and try to acquire - should return error
	l2 := NewLimiter(1)
	l2.WithTimeout(100 * time.Millisecond)
	
	err := l2.Acquire()
	if err != nil {
		t.Errorf("Expected Acquire to succeed with available slot, got error: %v", err)
	}
}

func TestAcquireWithTimeout(t *testing.T) {
	l := NewLimiter(1)

	// Hold the slot
	if err := l.Acquire(); err != nil {
		t.Fatalf("Failed to acquire first slot: %v", err)
	}

	// Try to acquire with timeout - should return error
	l2 := l.WithTimeout(100 * time.Millisecond)
	
	err := l2.Acquire()
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Expected DeadlineExceeded error, got %v", err)
	}

	// Release for cleanup
	l.Release()
}

func TestAcquireWithContext(t *testing.T) {
	l := NewLimiter(1)

	// Hold the slot
	if err := l.Acquire(); err != nil {
		t.Fatalf("Failed to acquire first slot: %v", err)
	}

	// Try to acquire with context timeout - should return error
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	
	err := l.AcquireWithContext(ctx)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Expected DeadlineExceeded error, got %v", err)
	}

	// Release for cleanup
	l.Release()
}

func TestAcquireWithContextSuccess(t *testing.T) {
	l := NewLimiter(1)
	ctx := context.Background()
	
	err := l.AcquireWithContext(ctx)
	if err != nil {
		t.Errorf("Expected AcquireWithContext to succeed, got error: %v", err)
	}
	
	l.Release()
}

func TestRelease(t *testing.T) {
	l := NewLimiter(2)

	// Initial state - no slots in use
	if l.InUseSlots() != 0 {
		t.Errorf("Expected 0 slots in use, got %d", l.InUseSlots())
	}

	if err := l.Acquire(); err != nil {
		t.Fatalf("Failed to acquire: %v", err)
	}
	
	if l.InUseSlots() != 1 {
		t.Errorf("Expected 1 slot in use, got %d", l.InUseSlots())
	}

	l.Release()
	
	if l.InUseSlots() != 0 {
		t.Errorf("Expected 0 slots in use after release, got %d", l.InUseSlots())
	}
}

func TestExecute(t *testing.T) {
	var wg sync.WaitGroup
	results := make([]int, 5)
	l := NewLimiter(2)

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			err := l.Execute(func() error { 
				results[idx] = idx
				return nil 
			})
			if err != nil {
				t.Errorf("Execute failed: %v", err)
			}
		}(i)
	}

	wg.Wait()

	for i, v := range results {
		if v != i {
			t.Errorf("Expected result %d at index %d, got %d", i, i, v)
		}
	}
}

func TestExecuteWithContext(t *testing.T) {
	l := NewLimiter(1)
	ctx := context.Background()

	var result string
	err := l.ExecuteWithContext(ctx, func(c context.Context) error {
		result = "success"
		return nil
	})
	
	if err != nil {
		t.Errorf("ExecuteWithContext failed: %v", err)
	}
	if result != "success" {
		t.Errorf("Expected 'success', got %q", result)
	}
}

func TestExecuteWithCancellation(t *testing.T) {
	l := NewLimiter(1)
	
	// Hold the slot with a long-running operation
	go func() {
		l.Execute(func() error {
			time.Sleep(2 * time.Second)
			return nil
		})
	}()

	// Give it time to acquire
	time.Sleep(100 * time.Millisecond)

	// Try to execute with a short timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	
	err := l.ExecuteWithContext(ctx, func(c context.Context) error {
		return nil
	})
	
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Expected DeadlineExceeded, got %v", err)
	}
}

func TestAvailableSlots(t *testing.T) {
	l := NewLimiter(3)

	if l.AvailableSlots() != 3 {
		t.Errorf("Expected 3 available slots, got %d", l.AvailableSlots())
	}

	if err := l.Acquire(); err != nil {
		t.Fatalf("Failed to acquire: %v", err)
	}
	
	if l.AvailableSlots() != 2 {
		t.Errorf("Expected 2 available slots after acquire, got %d", l.AvailableSlots())
	}
}

func TestInUseSlots(t *testing.T) {
	l := NewLimiter(3)

	if l.InUseSlots() != 0 {
		t.Errorf("Expected 0 in-use slots, got %d", l.InUseSlots())
	}

	if err := l.Acquire(); err != nil {
		t.Fatalf("Failed to acquire: %v", err)
	}
	
	if l.InUseSlots() != 1 {
		t.Errorf("Expected 1 in-use slot after acquire, got %d", l.InUseSlots())
	}
}

func TestMaxSlots(t *testing.T) {
	l := NewLimiter(7)

	if l.MaxSlots() != 7 {
		t.Errorf("Expected max slots 7, got %d", l.MaxSlots())
	}
}

func TestLimiterGroup(t *testing.T) {
	ctx := context.Background()
	lg := NewLimiterGroup(ctx, 2)
	
	if lg == nil {
		t.Fatal("Expected non-nil LimiterGroup")
	}
	
	if lg.limiter.MaxSlots() != 2 {
		t.Errorf("Expected limiter max slots = 2, got %d", lg.limiter.MaxSlots())
	}
}

func TestExecuteWithReturnError(t *testing.T) {
	l := NewLimiter(1)
	testErr := errors.New("test error")
	
	err := l.Execute(func() error {
		return testErr
	})
	
	if err != testErr {
		t.Errorf("Expected error %v, got %v", testErr, err)
	}
}

func TestConcurrentAccess(t *testing.T) {
	l := NewLimiter(2)
	var wg sync.WaitGroup
	
	const numGoroutines = 10
	const iterations = 50

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				if err := l.Acquire(); err != nil {
					t.Errorf("Acquire failed: %v", err)
					return
				}
				time.Sleep(time.Microsecond * 10) // Simulate work
				l.Release()
			}
		}()
	}

	wg.Wait()

	// All slots should be released
	if l.InUseSlots() != 0 {
		t.Errorf("Expected 0 in-use slots after all goroutines complete, got %d", l.InUseSlots())
	}
}

func TestConcurrentAccessWithTimeout(t *testing.T) {
	l := NewLimiter(2).WithTimeout(5 * time.Second)
	var wg sync.WaitGroup
	
	const numGoroutines = 10

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			
			err := l.Execute(func() error {
				time.Sleep(50 * time.Millisecond) // Simulate work
				return nil
			})
			
			if err != nil {
				t.Errorf("Goroutine %d Execute failed: %v", id, err)
			}
		}(i)
	}

	wg.Wait()

	if l.InUseSlots() != 0 {
		t.Errorf("Expected 0 in-use slots after all goroutines complete, got %d", l.InUseSlots())
	}
}
