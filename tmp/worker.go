package tmp

import (
	"context"
	"sync"
	"time"
)

// Observer defines the interface for monitoring worker execution.
// Implementations can track statistics and log errors during load testing.
type Observer interface {
	SetStats(err error) error
	LogError(err error, message string, args ...any)
}

// Worker represents a single worker that executes the target function
// at a specified rate. It manages its own goroutine and job queue.
type Worker struct {
	observer Observer
	target   Target
	done     chan struct{}
	jobs     chan struct{}
	wait     sync.WaitGroup
	id       int
}

// Hire creates a new Worker instance with the given ID and target function.
// The worker is initially created without an observer.
func Hire(id int, target Target) *Worker {
	return &Worker{
		id:       id,
		target:   target,
		observer: nil,
		wait:     sync.WaitGroup{},
		done:     make(chan struct{}),
		jobs:     make(chan struct{}),
	}
}

// WithObserver sets an observer for the worker and wraps the target function
// to automatically call observer methods for statistics and error logging.
func (w *Worker) WithObserver(observer Observer) *Worker {
	target := w.target

	w.target = func(ctx context.Context) error {
		err := target(ctx)
		if err != nil {
			observer.LogError(err, "failure", "worker", w.id)
		}

		return observer.SetStats(err)
	}

	return w
}

// Wait blocks until the worker has finished processing all jobs.
func (w *Worker) Wait() {
	<-w.done
}

// Do starts the worker with the specified RPS (requests per second).
// It creates a single goroutine that processes jobs from the job queue
// at the specified rate until the context is cancelled.
func (w *Worker) Do(ctx context.Context, rps int) {
	defer close(w.done)

	// Create a single goroutine per worker, not per RPS
	w.wait.Add(1)

	go func() {
		defer w.wait.Done()

		for range w.jobs {
			_ = w.target(ctx)
		}
	}()

	ticker := time.NewTicker(time.Second / time.Duration(rps))
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.finish()

			return
		case <-ticker.C:
			select {
			case <-ctx.Done():
				w.finish()

				return
			case w.jobs <- struct{}{}:
			}
		}
	}
}

func (w *Worker) finish() {
	close(w.jobs)
	w.wait.Wait()
}
