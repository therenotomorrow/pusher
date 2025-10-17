package pusher

import (
	"context"
	"sync"
	"time"
)

type Observer interface {
	SetStats(err error) error
	LogError(err error, message string, args ...any)
}

type Worker struct {
	observer Observer
	target   Target
	done     chan struct{}
	jobs     chan struct{}
	wait     sync.WaitGroup
	id       int
}

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

func (w *Worker) Wait() {
	<-w.done
}

func (w *Worker) Do(ctx context.Context, rps int) {
	defer close(w.done)

	w.wait.Add(rps)

	for range rps {
		go func() {
			defer w.wait.Done()

			for range w.jobs {
				_ = w.target(ctx)
			}
		}()
	}

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
