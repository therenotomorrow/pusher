package pusher

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/therenotomorrow/ex"
	"golang.org/x/sync/errgroup"
)

const (
	ErrInvalidRPS    = ex.Error("rps must be positive value")
	ErrMissingTarget = ex.Error("target is missing")
	ErrWorkerIsBusy  = ex.Error("worker is busy")
)

type Result interface {
	fmt.Stringer
}

type Target func(ctx context.Context) (Result, error)

type Offer func(w *Worker)

type When int

const (
	BeforeTarget = iota
	AfterTarget
	Cancelled
)

type Gossip struct {
	Result Result
	Error  error
	When   When
}

func (g *Gossip) String() string {
	if g == nil {
		return "<nil>"
	}

	if g.Result == nil {
		return "<empty>"
	}

	return g.Result.String()
}

func (g *Gossip) Cancelled() bool {
	return g.When == Cancelled
}

func (g *Gossip) BeforeTarget() bool {
	return g.When == BeforeTarget
}

func (g *Gossip) AfterTarget() bool {
	return g.When == AfterTarget
}

type Gossiper interface {
	Listen(ctx context.Context, id string, gossips <-chan *Gossip)
	Stop()
}

type config struct {
	listeners []Gossiper
	overtime  int
}

func WithGossips(listeners ...Gossiper) Offer {
	return func(w *Worker) {
		w.config.listeners = listeners
	}
}

func WithOvertime(limit int) Offer {
	return func(w *Worker) {
		w.config.overtime = limit
	}
}

type Worker struct {
	target Target
	wlb    chan struct{}
	ident  string
	config config
	wait   sync.WaitGroup
	busy   atomic.Bool
}

const (
	double          = 2
	defaultOvertime = 1_000_000
)

func Hire(ident string, target Target, offers ...Offer) (*Worker, error) {
	if target == nil {
		return nil, ErrMissingTarget
	}

	worker := &Worker{
		ident:  ident,
		target: target,
		config: config{
			overtime:  defaultOvertime,
			listeners: make([]Gossiper, 0),
		},
		wlb:  nil, // will be ignored when Worker.Do calls
		wait: sync.WaitGroup{},
		busy: atomic.Bool{},
	}

	for _, offer := range offers {
		offer(worker)
	}

	worker.wlb = make(chan struct{}, worker.config.overtime)

	return worker, nil
}

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

func (w *Worker) runListeners(ctx context.Context, rps int) []chan *Gossip {
	tracks := make([]chan *Gossip, 0)

	for _, gossiper := range w.config.listeners {
		tracks = append(tracks, make(chan *Gossip, double*rps))

		go gossiper.Listen(ctx, w.ident, tracks[len(tracks)-1])
	}

	return tracks
}

func (w *Worker) complete(tracks []chan *Gossip) {
	w.wait.Wait()

	for id, track := range tracks {
		close(track)
		w.config.listeners[id].Stop()
	}

	w.busy.Store(false)
}

func (w *Worker) whisp(tracks []chan *Gossip, gossip *Gossip) {
	for _, track := range tracks {
		select {
		case track <- gossip:
		default:
		}
	}
}

func (w *Worker) shout(tracks []chan *Gossip, gossip *Gossip) {
	for _, track := range tracks {
		track <- gossip
	}
}

func Work(target Target, rps int, duration time.Duration, offers ...Offer) error {
	worker, err := Hire("judas", target, offers...)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	return worker.Work(ctx, rps)
}

func Farm(workers []*Worker, rps int, duration time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	group, gtx := errgroup.WithContext(ctx)

	for _, worker := range workers {
		group.Go(func() error {
			return worker.Work(gtx, rps)
		})
	}

	return ex.Cast(group.Wait())
}

func Force(target Target, rps int, duration time.Duration, amount int, offers ...Offer) error {
	workers := make([]*Worker, amount)
	for ident := range workers {
		worker, err := Hire(fmt.Sprintf("force #%d", ident), target, offers...)
		if err != nil {
			return err
		}

		workers[ident] = worker
	}

	return Farm(workers, rps, duration)
}
