package pusher

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/therenotomorrow/ex"
)

// Result represents the outcome of a single Target execution.
type Result interface {
	fmt.Stringer
}

// Target is a function that performs the work to be tested.
// It receives a context for cancellation and must return a Result and an error.
type Target func(ctx context.Context) (Result, error)

// Offer is a functional option for configuring a Worker.
// This pattern allows for flexible and extensible Worker initialization.
type Offer func(w *Worker)

// Gossiper defines the interface for listeners that process Gossip events.
// This allows plugging in various metric collectors, loggers, or reporters.
type Gossiper interface {
	// Listen runs in its own goroutine and processes events from the gossips channel.
	Listen(ctx context.Context, worker *Worker, gossips <-chan *Gossip)
	// Stop is called to gracefully shut down the listener and flush any buffered data.
	Stop()
}

// Worker is the core entity that generates load by repeatedly calling the Target
// function at a specified rate (RPS) and concurrency limit.
type Worker struct {
	target Target
	// wlb (work-life balance) is a channel used as a semaphore to limit the
	// number of concurrent Target calls.
	wlb    chan struct{}
	ident  string
	config config
	wait   sync.WaitGroup
	busy   atomic.Bool
}

// Work starts the load generation loop. It's a blocking method that runs until
// the provided context is canceled. It generates requests at the specified RPS,
// respecting the concurrency limit.
func (w *Worker) Work(ctx context.Context, rps int) error {
	tick, err := w.isReady(rps)
	if err != nil {
		return err
	}

	tracks := w.runListeners(ctx, rps)

	defer w.complete(tracks)

	timeless := time.NewTicker(tick)

	defer timeless.Stop()

	for {
		select {
		case <-ctx.Done():
			return ex.Cast(ctx.Err())

		case <-timeless.C:
			// This inner select attempts to acquire a semaphore slot.
			// If all slots are busy, it emits a Cancelled event and skips the tick.
			// It also checks for context cancellation for an immediate exit.
			select {
			case w.wlb <- struct{}{}:
			case <-ctx.Done():
				return ex.Cast(ctx.Err())
			default:
				w.whisp(tracks, &Gossip{When: Cancelled, Result: nil, Error: nil})

				continue // move to the next tick
			}

			w.wait.Go(func() {
				defer func() { <-w.wlb }()

				w.shout(tracks, &Gossip{When: BeforeTarget, Result: nil, Error: nil})
				res, err := w.target(ctx)
				w.shout(tracks, &Gossip{When: AfterTarget, Result: res, Error: err})
			})
		}
	}
}

func (w *Worker) String() string {
	return w.ident
}

// isReady performs pre-flight checks before starting the main loop.
// It ensures the worker is not already busy and validates the RPS value.
func (w *Worker) isReady(rps int) (time.Duration, error) {
	if !w.busy.CompareAndSwap(false, true) {
		return 0, ErrWorkerIsBusy.Reason("try again later")
	}

	if rps < 1 {
		w.busy.Store(false)

		return 0, ErrInvalidRPS.Reason("rps must be positive")
	}

	tick := time.Second / time.Duration(rps)

	if tick < time.Nanosecond {
		return 0, ErrInvalidRPS.Reason("rps too large, resulting tick < 1ns")
	}

	return tick, nil
}

// runListeners starts a goroutine for each configured Gossiper,
// creating a channel for each to receive events.
func (w *Worker) runListeners(ctx context.Context, rps int) []chan *Gossip {
	tracks := make([]chan *Gossip, 0)

	for _, gossiper := range w.config.listeners {
		tracks = append(tracks, make(chan *Gossip, double*rps))

		go gossiper.Listen(ctx, w, tracks[len(tracks)-1])
	}

	return tracks
}

// complete handles the graceful shutdown of the worker. It waits for all active
// tasks to finish, then stops and closes all associated listeners and channels.
func (w *Worker) complete(tracks []chan *Gossip) {
	w.wait.Wait()

	for id, track := range tracks {
		close(track)
		w.config.listeners[id].Stop()
	}

	w.busy.Store(false)
}

// whisp performs a non-blocking, best-effort send of an event.
// It's used for events like 'Cancelled', where losing some telemetry under high
// load is acceptable to avoid blocking the worker.
func (w *Worker) whisp(tracks []chan *Gossip, gossip *Gossip) {
	for _, track := range tracks {
		select {
		case track <- gossip:
		default:
		}
	}
}

// shout performs a blocking send of an event, ensuring its delivery.
// It's used for critical events (task results) where data loss is unacceptable.
// This can create backpressure if a listener is slow.
func (w *Worker) shout(tracks []chan *Gossip, gossip *Gossip) {
	for _, track := range tracks {
		track <- gossip
	}
}
