package pusher

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"os"
	"reflect"
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

type Step string

const (
	StepBefore Step = "before"
	StepAfter  Step = "after"
)

type Gossip struct {
	Result Result
	Error  error
	Step   Step
}

func (g *Gossip) Before() bool {
	return g.Step == StepBefore
}

func (g *Gossip) After() bool {
	return g.Step == StepAfter
}

type Gossiper interface {
	Listen(ctx context.Context, id string, gossips <-chan *Gossip)
	Finish(ctx context.Context, id string)
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
	id      string
	target  Target
	config  config
	// internals
	wlb     chan struct{} // work-life balance
	wait    sync.WaitGroup
	running atomic.Bool
	runWg   sync.WaitGroup
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
		wait: sync.WaitGroup{},
	}

	for _, offer := range offers {
		offer(worker)
	}

	return worker, nil
}

func (w *Worker) Work(ctx context.Context, rps int) error {
	if !w.running.CompareAndSwap(false, true) {
		return ErrWorkerIsBusy
	}

	if rps < 1 {
		w.running.Store(false)
		return ErrInvalidRPS
	}

	w.runWg.Add(1)

	if !w.config.overtime {
		w.wlb = make(chan struct{}, rps)
	}

	tracker := make([]chan *Gossip, 0)
	for _, gossiper := range w.config.listeners {
		tracker = append(tracker, make(chan *Gossip, 2*rps))

		go gossiper.Listen(ctx, w.id, tracker[len(tracker)-1])
	}

	go func() {
		defer func() {
			w.wait.Wait()

			for id := range tracker {
				close(tracker[id])
				w.config.listeners[id].Finish(ctx, w.id)
			}

			if !w.config.overtime {
				close(w.wlb)
			}

			w.running.Store(false)
			w.runWg.Done()
		}()

		timeless := time.NewTicker(time.Second / time.Duration(rps))
		defer timeless.Stop()

		for {
			select {
			case <-ctx.Done():
				return

			case <-timeless.C:
				for _, track := range tracker {
					select {
					case track <- &Gossip{Step: StepBefore, Result: nil, Error: nil}:
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

				w.wait.Add(1)
				go func() {
					defer w.wait.Done()
					defer func() {
						if !w.config.overtime {
							<-w.wlb
						}
					}()

					res, err := w.target(ctx)

					for _, track := range tracker {
						select {
						case track <- &Gossip{Step: StepAfter, Result: res, Error: err}:
						default:
						}
					}
				}()
			}
		}
	}()

	return nil
}
func (w *Worker) Wait() {
	w.runWg.Wait()
}

func Work(target Target, rps int, duration time.Duration, offers ...Offer) error {
	w, err := Hire("judas", target, offers...)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	if err := w.Work(ctx, rps); err != nil {
		return err
	}

	w.Wait()

	return nil
}

func TestMe(_ context.Context) (int, error) {
	val := rand.Int32() % 100

	switch {
	case val < 50:
		//time.Sleep(3 * time.Second)
		time.Sleep(time.Second / 2)

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

type Notebook struct {
	log *slog.Logger
}

func (n *Notebook) Listen(_ context.Context, id string, gossips <-chan *Gossip) {
	for gossip := range gossips {
		if gossip.Before() {
			continue
		}

		var result string
		if gossip.Result != nil {
			// Use reflection to safely handle potentially nil pointers inside the interface
			v := reflect.ValueOf(gossip.Result)
			if v.Kind() == reflect.Ptr && v.IsNil() {
				result = "<nil>"
			} else {
				result = gossip.Result.String()
			}
		}

		if gossip.Error != nil {
			n.log.Error("failure", "worker", id, "result", result, "error", gossip.Error)
		} else {
			n.log.Info("success", "worker", id, "result", result)
		}
	}
}

func (n *Notebook) Finish(_ context.Context, _ string) {}

const maxBucketSize = 600 // 10 minutes of per-second data

type Tracker struct {
	receive atomic.Int64
	planned atomic.Int64
	success atomic.Int64
	failure atomic.Int64
	buckets map[string][]int64
	perSec  atomic.Int64
	mutex   sync.Mutex
	wait    sync.WaitGroup
}

func (t *Tracker) Listen(ctx context.Context, id string, gossips <-chan *Gossip) {
	t.wait.Add(1)
	defer t.wait.Done()

	t.mutex.Lock()
	if _, ok := t.buckets[id]; !ok {
		t.buckets[id] = make([]int64, 0)
	}
	t.mutex.Unlock()

	t.wait.Add(1)
	go func() {
		defer t.wait.Done()
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				t.mutex.Lock()
				t.buckets[id] = append(t.buckets[id], t.perSec.Swap(0))
				if len(t.buckets[id]) > maxBucketSize {
					t.buckets[id] = t.buckets[id][1:]
				}
				t.mutex.Unlock()
			case <-ctx.Done():
				if perSec := t.perSec.Load(); perSec > 0 {
					t.mutex.Lock()
					t.buckets[id] = append(t.buckets[id], perSec)
					t.mutex.Unlock()
				}
				return
			}
		}
	}()

	for gossip := range gossips {
		if gossip.Before() {
			t.planned.Add(1)
			continue
		}

		t.receive.Add(1)
		t.perSec.Add(1)

		if gossip.Error != nil {
			t.failure.Add(1)
		} else {
			t.success.Add(1)
		}
	}
}

func (t *Tracker) Finish(_ context.Context, _ string) {
	t.wait.Wait()
}

func Main() {
	target := func(ctx context.Context) (Result, error) {
		res, err := TestMe(ctx)

		return IntResult(res), err
	}
	notes := &Notebook{
		log: slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			AddSource:   false,
			Level:       slog.LevelInfo,
			ReplaceAttr: nil,
		})),
	}
	track := &Tracker{
		receive: atomic.Int64{},
		planned: atomic.Int64{},
		success: atomic.Int64{},
		failure: atomic.Int64{},
		perSec:  atomic.Int64{},
		buckets: make(map[string][]int64),
		mutex:   sync.Mutex{},
		wait:    sync.WaitGroup{},
	}

	start := time.Now()

		if err := Work(target, 20, 2*time.Second,
		WithGossips(notes, track),
		WithOvertime(),
	); err != nil {
		notes.log.Error("work failed", "error", err)
		return
	}

	fmt.Println("took", time.Since(start))
	fmt.Printf("%+v", track)
}
