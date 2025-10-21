package pusher

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/therenotomorrow/ex"
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
		busy: atomic.Bool{},
	}

	for _, offer := range offers {
		offer(worker)
	}

	worker.wlb = make(chan struct{}, worker.config.overtime)

	return worker, nil
}

func (w *Worker) Work(ctx context.Context, rps int) error {
	if !w.busy.CompareAndSwap(false, true) {
		return ErrWorkerIsBusy.Reason("try again later")
	}

	if rps < 1 {
		w.busy.Store(false)

		return ErrInvalidRPS.Reason("rps must be positive")
	}

	tick := time.Second / time.Duration(rps)

	if tick < time.Nanosecond {
		return ErrInvalidRPS.Reason("rps too large, resulting tick < 1ns")
	}

	wait := sync.WaitGroup{}
	tracker := make([]chan *Gossip, 0)

	for _, gossiper := range w.config.listeners {
		tracker = append(tracker, make(chan *Gossip, double*rps))

		go gossiper.Listen(ctx, w.ident, tracker[len(tracker)-1])
	}

	defer func() {
		wait.Wait()

		for id, track := range tracker {
			close(track)
			w.config.listeners[id].Stop()
		}

		w.busy.Store(false)
	}()

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
				continue
			default:
				for _, track := range tracker {
					select {
					case track <- &Gossip{When: Cancelled, Result: nil, Error: nil}:
					default:
					}
				}

				continue // move to the next tick
			}

			wait.Go(func() {
				defer func() {
					<-w.wlb
				}()

				for _, track := range tracker {
					track <- &Gossip{When: BeforeTarget, Result: nil, Error: nil}
				}

				res, err := w.target(ctx)

				for _, track := range tracker {
					track <- &Gossip{When: AfterTarget, Result: res, Error: err}
				}
			})
		}
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

func Farm(workers []*Worker, rps int, duration time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	var wait sync.WaitGroup

	for _, worker := range workers {
		wait.Go(func() {
			_ = worker.Work(ctx, rps) // we ignore errors from workers
		})
	}

	wait.Wait()
}

func TestMe(_ context.Context) (int, error) {
	return 0, nil
}

type IntResult int

func (r IntResult) String() string {
	return strconv.Itoa(int(r))
}

func Main() error {
	rps := 20
	dur := time.Second
	lim := 100

	target := func(ctx context.Context) (Result, error) {
		res, err := TestMe(ctx)

		return IntResult(res), err
	}

	return Work(target, rps, dur, WithOvertime(lim))
}
