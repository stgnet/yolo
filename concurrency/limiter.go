// Package concurrency provides thread pool and worker management utilities
package concurrency

import (
	"context"
	"sync"
	"time"
)

// Limiter controls the number of concurrent operations using a semaphore pattern.
// It's useful for rate limiting, connection pooling, or controlling resource usage.
type Limiter struct {
	sema    chan struct{}
	timeout time.Duration
}

// NewLimiter creates a new limiter that allows at most n concurrent operations.
func NewLimiter(n int) *Limiter {
	if n <= 0 {
		n = 1
	}
	return &Limiter{
		sema:    make(chan struct{}, n),
		timeout: 30 * time.Second, // Default timeout for acquiring semaphore
	}
}

// WithTimeout sets the timeout for acquiring a slot (default: 30 seconds).
func (l *Limiter) WithTimeout(timeout time.Duration) *Limiter {
	l.timeout = timeout
	return l
}

// TryAcquire attempts to acquire a slot without blocking.
// Returns true if a slot was acquired, false otherwise.
func (l *Limiter) TryAcquire() bool {
	select {
	case l.sema <- struct{}{}:
		return true
	default:
		return false
	}
}

// Acquire blocks until a slot is available or timeout is reached.
// Returns an error if the timeout was exceeded.
func (l *Limiter) Acquire() error {
	ctx, cancel := context.WithTimeout(context.Background(), l.timeout)
	defer cancel()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case l.sema <- struct{}{}:
		return nil
	}
}

// AcquireWithContext acquires a slot respecting the given context.
func (l *Limiter) AcquireWithContext(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case l.sema <- struct{}{}:
		return nil
	}
}

// Release returns a slot to the pool. Call this in a defer after acquiring.
func (l *Limiter) Release() {
	select {
	case <-l.sema:
	default:
		// This should never happen if Release is called correctly
	}
}

// Execute runs a function with concurrency limiting.
// Automatically acquires and releases the semaphore.
func (l *Limiter) Execute(fn func() error) error {
	if err := l.Acquire(); err != nil {
		return err
	}
	defer l.Release()
	return fn()
}

// ExecuteWithContext runs a function with concurrency limiting and context.
func (l *Limiter) ExecuteWithContext(ctx context.Context, fn func(context.Context) error) error {
	if err := l.AcquireWithContext(ctx); err != nil {
		return err
	}
	defer l.Release()
	return fn(ctx)
}

// AvailableSlots returns the number of currently available slots.
func (l *Limiter) AvailableSlots() int {
	return cap(l.sema) - len(l.sema)
}

// InUseSlots returns the number of currently in-use slots.
func (l *Limiter) InUseSlots() int {
	return len(l.sema)
}

// MaxSlots returns the maximum number of concurrent operations allowed.
func (l *Limiter) MaxSlots() int {
	return cap(l.sema)
}

// LimiterGroup combines a Group with a Limiter for structured concurrency
// with rate limiting. All goroutines respect the same concurrency limit.
type LimiterGroup struct {
	*Group
	limiter *Limiter
	mu      sync.Mutex
	running int
}

// NewLimiterGroup creates a new group with built-in rate limiting.
func NewLimiterGroup(parentCtx context.Context, maxConcurrent int) *LimiterGroup {
	return &LimiterGroup{
		Group:   NewGroup(parentCtx),
		limiter: NewLimiter(maxConcurrent),
	}
}

// Go executes a function in a new goroutine with rate limiting.
// Blocks until a slot is available, then starts the goroutine.
func (lg *LimiterGroup) Go(f func(context.Context) error) {
	lg.mu.Lock()
	if lg.limiter.InUseSlots() >= lg.limiter.MaxSlots() {
		// We'll wait for a slot in the wrapped function
	}
	lg.mu.Unlock()

	// Wrap the function to acquire/release semaphore
	wrapped := func(ctx context.Context) error {
		if err := lg.limiter.AcquireWithContext(ctx); err != nil {
			return err
		}
		defer lg.limiter.Release()
		lg.mu.Lock()
		lg.running++
		lg.mu.Unlock()

		err := f(ctx)

		lg.mu.Lock()
		lg.running--
		lg.mu.Unlock()

		return err
	}

	lg.Group.Go(wrapped)
}

// Running returns the number of currently running goroutines.
func (lg *LimiterGroup) Running() int {
	lg.mu.Lock()
	defer lg.mu.Unlock()
	return lg.running
}
