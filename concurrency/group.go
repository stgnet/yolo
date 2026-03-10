// Package concurrency provides thread pool and worker management utilities
package concurrency

import (
	"context"
	"errors"
	"sync"
	"time"
)

// ErrContextDone is returned when the context is cancelled before operation completes
var ErrContextDone = errors.New("context done")

// Group represents a structured concurrency group that ensures all goroutines
// complete before the parent returns. This is inspired by Java's structured
// concurrency and Go's sync.WaitGroup with enhanced error handling.
type Group struct {
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	mu     sync.Mutex
	errs   []error
}

// NewGroup creates a new concurrency group with the given parent context.
// The group manages cancellation and error collection for child goroutines.
func NewGroup(parentCtx context.Context) *Group {
	ctx, cancel := context.WithCancel(parentCtx)
	return &Group{
		ctx:    ctx,
		cancel: cancel,
	}
}

// Go executes a function in a new goroutine as part of this group.
// The goroutine is tracked and will be waited for when Wait() or Run() is called.
// Errors from the function are collected and can be retrieved via Errors().
func (g *Group) Go(f func(context.Context) error) {
	g.wg.Add(1)
	go func() {
		defer g.wg.Done()
		if err := f(g.ctx); err != nil {
			g.mu.Lock()
			g.errs = append(g.errs, err)
			g.mu.Unlock()
		}
	}()
}

// GoWithErr executes a function that takes an error parameter for communication.
// This is useful for fan-in patterns where multiple goroutines need to send results.
func (g *Group) GoWithErr(f func(context.Context, chan<- error)) {
	errChan := make(chan error, 1)
	g.Go(func(ctx context.Context) error {
		f(ctx, errChan)
		select {
		case err := <-errChan:
			return err
		default:
			return nil
		}
	})
}

// Wait blocks until all goroutines in the group have completed.
// Returns immediately if no goroutines are running.
func (g *Group) Wait() {
	g.wg.Wait()
}

// Run executes all queued goroutines and waits for them to complete.
// It also cancels the context after all goroutines finish, ensuring cleanup.
func (g *Group) Run() {
	g.Wait()
	g.cancel()
}

// Cancel cancels the group's context, causing all running goroutines
// to receive a cancelled context. Call Wait() or Run() to wait for completion.
func (g *Group) Cancel() {
	g.cancel()
}

// Context returns the group's context, which should be used by child goroutines
// to check for cancellation.
func (g *Group) Context() context.Context {
	return g.ctx
}

// Errors returns all errors collected from goroutines that returned non-nil errors.
// The slice is a copy, so it's safe to use after the group completes.
func (g *Group) Errors() []error {
	g.mu.Lock()
	defer g.mu.Unlock()
	errs := make([]error, len(g.errs))
	copy(errs, g.errs)
	return errs
}

// Error returns the first error from the goroutines, or nil if all succeeded.
func (g *Group) Error() error {
	errs := g.Errors()
	if len(errs) == 0 {
		return nil
	}
	return errs[0]
}

// FanOut distributes a value to multiple workers and collects their results.
// Each worker receives the input and sends its result to the returned channel.
// The channel is closed after all workers complete.
func (g *Group) FanOut(input interface{}, numWorkers int, worker func(context.Context, interface{}) error) <-chan error {
	resultChan := make(chan error, numWorkers)

	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := worker(g.ctx, input)
			select {
			case resultChan <- err:
			default:
				// Channel full, drop the result (shouldn't happen with buffered channel)
			}
		}()
	}

	// Close the channel after all workers complete
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	return resultChan
}

// FanIn collects results from multiple channels into a single channel.
// It spawns goroutines to read from each input channel and write to the output channel.
// The output channel is closed after all input channels are exhausted.
// Goroutines respect context cancellation but continue until channels are closed or context is cancelled.
func (g *Group) FanIn(inputChans ...<-chan error) <-chan error {
	outputChan := make(chan error, len(inputChans)*10) // Buffered for better throughput

	var wg sync.WaitGroup
	for _, ch := range inputChans {
		inputChan := ch // Capture loop variable
		wg.Add(1)
		go func(ctx context.Context, in <-chan error) {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case err, ok := <-in:
					if !ok {
						return
					}
					select {
					case <-ctx.Done():
						return
					case outputChan <- err:
					}
				}
			}
		}(g.ctx, inputChan)
	}

	go func() {
		wg.Wait()
		close(outputChan)
	}()

	return outputChan
}

// Pipeline creates a pipeline of stages where each stage processes data
// from the previous stage's output channel. Returns the final output channel.
func (g *Group) Pipeline(stages ...func(context.Context, <-chan interface{}) <-chan interface{}) <-chan interface{} {
	if len(stages) == 0 {
		return nil
	}

	// Create input channel for first stage
	var prevOutput <-chan interface{}

	for i, stage := range stages {
		inputCh := make(chan interface{}, 100)
		if i == 0 {
			prevOutput = inputCh
		}

		outputCh := stage(g.ctx, inputCh)
		prevOutput = outputCh
	}

	return prevOutput
}

// Barrier is a synchronization primitive that blocks goroutines until
// a specified number of goroutines reach the barrier.
type Barrier struct {
	chans []chan struct{}
	count int
	index int
	mu    sync.Mutex
}

// NewBarrier creates a new barrier for the specified number of goroutines.
func NewBarrier(n int) *Barrier {
	if n <= 0 {
		n = 1
	}
	return &Barrier{
		chans: make([]chan struct{}, n),
		count: n,
	}
}

// Wait blocks until all goroutines have reached the barrier.
func (b *Barrier) Wait() {
	b.mu.Lock()
	idx := b.index
	b.index++
	b.mu.Unlock()

	if idx >= len(b.chans) {
		return
	}

	// First goroutine to reach index creates channel and waits
	ch := make(chan struct{})
	b.chans[idx] = ch

	// Last goroutine closes all channels
	if idx == b.count-1 {
		for _, ch := range b.chans {
			if ch != nil {
				close(ch)
			}
		}
	} else {
		<-ch // Wait for barrier to be released
	}
}

// Reset resets the barrier for reuse. Must not be called while goroutines
// are waiting at the barrier.
func (b *Barrier) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.index = 0
	for i := range b.chans {
		b.chans[i] = nil
	}
}

// LimitedConcurrency runs goroutines with a limit on how many can run concurrently.
// This is similar to a semaphore pattern using channels.
func LimitedConcurrency(ctx context.Context, maxWorkers int, jobs []func(context.Context) error) []error {
	sem := make(chan struct{}, maxWorkers)
	var wg sync.WaitGroup
	errsMu := sync.Mutex{}
	errs := make([]error, 0, len(jobs))

	for _, job := range jobs {
		wg.Add(1)
		go func(f func(context.Context) error) {
			sem <- struct{}{}        // Acquire semaphore
			defer func() { <-sem }() // Release semaphore
			defer wg.Done()

			if ctx.Err() != nil {
				return
			}

			if err := f(ctx); err != nil {
				errsMu.Lock()
				errs = append(errs, err)
				errsMu.Unlock()
			}
		}(job)
	}

	wg.Wait()
	return errs
}

// RetryWithBackoff retries a function with exponential backoff until it succeeds
// or maxRetries is reached. Respects context cancellation.
func RetryWithBackoff(ctx context.Context, maxRetries int, initialDelay time.Duration, f func(context.Context) error) error {
	var lastErr error
	delay := initialDelay

	for attempt := 0; attempt < maxRetries; attempt++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		err := f(ctx)
		if err == nil {
			return nil
		}

		lastErr = err

		if attempt < maxRetries-1 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
				delay *= 2 // Exponential backoff
			}
		}
	}

	return lastErr
}
