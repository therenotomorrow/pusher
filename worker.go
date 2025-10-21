package pusher

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
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
	Planned When = iota
	BeforeTarget
	AfterTarget
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

func (g *Gossip) Planned() bool {
	return g.When == Planned
}

func (g *Gossip) BeforeTarget() bool {
	return g.When < AfterTarget
}

func (g *Gossip) AfterTarget() bool {
	return g.When == AfterTarget
}

type Gossiper interface {
	Listen(ctx context.Context, id string, gossips <-chan *Gossip)
	io.Closer
}

type config struct {
	overtime  bool
	listeners []Gossiper
}

func WithGossips(listeners ...Gossiper) Offer {
	return func(w *Worker) {
		w.config.listeners = listeners
	}
}

func WithOvertime() Offer {
	return func(w *Worker) {
		w.config.overtime = true
	}
}

type Worker struct {
	id     string
	target Target
	config config
	// internals
	wlb  chan struct{} // work-life balance
	busy atomic.Bool
}

func Hire(id string, target Target, offers ...Offer) (*Worker, error) {
	if target == nil {
		return nil, ErrMissingTarget
	}

	worker := &Worker{
		id:     id,
		target: target,
		config: config{
			overtime:  false,
			listeners: make([]Gossiper, 0),
		},
		wlb:  make(chan struct{}), // will be ignored when Worker.Do calls
		busy: atomic.Bool{},
	}

	for _, offer := range offers {
		offer(worker)
	}

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

	if !w.config.overtime {
		w.wlb = make(chan struct{}, rps)
	}

	wait := sync.WaitGroup{}

	tracker := make([]chan *Gossip, 0)
	for _, gossiper := range w.config.listeners {
		tracker = append(tracker, make(chan *Gossip, 2*rps))

		go gossiper.Listen(ctx, w.id, tracker[len(tracker)-1])
	}

	defer func() {
		wait.Wait()

		if !w.config.overtime {
			close(w.wlb)
		}

		for id, track := range tracker {
			close(track)
			_ = w.config.listeners[id].Close()
		}

		w.busy.Store(false)
	}()

	timeless := time.NewTicker(tick)
	defer timeless.Stop()

	for {
		select {
		case <-ctx.Done():
			wait.Wait()
			return nil

		case <-timeless.C:
			for _, track := range tracker {
				select {
				case track <- &Gossip{When: Planned, Result: nil, Error: nil}:
				default:
				}
			}

			if !w.config.overtime {
				select {
				case w.wlb <- struct{}{}:
				default:
					continue
				}
			}

			wait.Add(1)
			go func() {
				defer wait.Done()
				defer func() {
					if !w.config.overtime {
						<-w.wlb
					}
				}()

				for _, track := range tracker {
					select {
					case track <- &Gossip{When: BeforeTarget, Result: nil, Error: nil}:
					default:
					}
				}

				res, err := w.target(ctx)

				for _, track := range tracker {
					select {
					case track <- &Gossip{When: AfterTarget, Result: res, Error: err}:
					default:
					}
				}
			}()
		}
	}
}

func Work(target Target, rps int, duration time.Duration, offers ...Offer) error {
	w, err := Hire("judas", target, offers...)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	return w.Work(ctx, rps)
}

func Farm(workers []*Worker, rps int, duration time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	var wait sync.WaitGroup

	for _, worker := range workers {
		wait.Go(func() {
			_ = worker.Work(ctx, rps)
		})
	}

	wait.Wait()

	return
}

func TestMe(_ context.Context) (int, error) {
	val := rand.Int32() % 100

	switch {
	case val < 50:
		//time.Sleep(3 * time.Second)
		//time.Sleep(time.Second / 2)

		return 1, nil
	case val < 75:
		return 2, nil
	default:
		return 0, errors.New("test error")
	}
}

type IntResult int

func (r IntResult) String() string {
	return strconv.Itoa(int(r))
}

func Main() {
	target := func(ctx context.Context) (Result, error) {
		res, err := TestMe(ctx)

		return IntResult(res), err
	}

	start := time.Now()

	if err := Work(target, 20, 2*time.Second, WithOvertime()); err != nil {
		return
	}

	fmt.Println("took", time.Since(start))
}
