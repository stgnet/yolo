// Package concurrency provides thread pool and worker management utilities
package concurrency

import (
	"context"
	"sync"
)

// ThreadPool manages a fixed number of worker goroutines
type ThreadPool struct {
	numWorkers int
	jobs       chan func()
	wg         sync.WaitGroup
	mu         sync.Mutex
	closed     bool
}

// NewThreadPool creates a new thread pool with the specified number of workers
func NewThreadPool(numWorkers int) *ThreadPool {
	if numWorkers <= 0 {
		numWorkers = 1
	}
	tp := &ThreadPool{
		numWorkers: numWorkers,
		jobs:       make(chan func(), 1000),
	}

	// Start workers
	for i := 0; i < numWorkers; i++ {
		tp.wg.Add(1)
		go tp.worker()
	}

	return tp
}

// worker processes jobs from the job channel
func (tp *ThreadPool) worker() {
	defer tp.wg.Done()

	for job := range tp.jobs {
		if job != nil {
			job()
		}
	}
}

// Submit adds a job to the queue
func (tp *ThreadPool) Submit(job func()) error {
	tp.mu.Lock()
	closed := tp.closed
	tp.mu.Unlock()

	if closed {
		return ErrPoolClosed
	}

	tp.jobs <- job
	return nil
}

// Close stops accepting new jobs and waits for all current jobs to complete
func (tp *ThreadPool) Close() {
	tp.mu.Lock()
	defer tp.mu.Unlock()

	if !tp.closed {
		tp.closed = true
		close(tp.jobs)
		tp.wg.Wait()
	}
}

// SubmitWithContext submits a job that respects context cancellation
func (tp *ThreadPool) SubmitWithContext(ctx context.Context, job func(context.Context)) error {
	wrappedJob := func() {
		select {
		case <-ctx.Done():
			return
		default:
			job(ctx)
		}
	}

	return tp.Submit(wrappedJob)
}

// ErrPoolClosed is returned when submitting to a closed pool
var ErrPoolClosed = &poolError{"pool is closed"}

// poolError represents pool-related errors
type poolError struct {
	msg string
}

func (e *poolError) Error() string {
	return e.msg
}

// WorkerFunc defines the signature for worker functions
type WorkerFunc func(context.Context) error

// ParallelExecutor executes multiple functions concurrently with a thread pool
type ParallelExecutor struct {
	pool   *ThreadPool
	ctx    context.Context
	cancel context.CancelFunc
	mu     sync.Mutex
	errors []error
}

// NewParallelExecutor creates a new parallel executor with the given worker count
func NewParallelExecutor(numWorkers int) *ParallelExecutor {
	ctx, cancel := context.WithCancel(context.Background())
	return &ParallelExecutor{
		pool:   NewThreadPool(numWorkers),
		ctx:    ctx,
		cancel: cancel,
	}
}

// Submit adds a job to the executor
func (e *ParallelExecutor) Submit(job WorkerFunc) {
	e.pool.SubmitWithContext(e.ctx, func(ctx context.Context) {
		if err := job(ctx); err != nil {
			e.mu.Lock()
			e.errors = append(e.errors, err)
			e.mu.Unlock()
		}
	})
}

// Close stops accepting new jobs and waits for all current jobs to complete
func (e *ParallelExecutor) Close() {
	e.pool.Close()
}

// Cancel stops the executor context and closes the pool
func (e *ParallelExecutor) Cancel() {
	e.cancel()
	e.pool.Close()
}

// Errors returns any errors that occurred during execution
func (e *ParallelExecutor) Errors() []error {
	e.mu.Lock()
	defer e.mu.Unlock()
	errors := make([]error, len(e.errors))
	copy(errors, e.errors)
	return errors
}
