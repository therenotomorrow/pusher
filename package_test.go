package pusher_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/therenotomorrow/ex"

	"github.com/therenotomorrow/pusher"
)

type result string

func (m result) String() string {
	return string(m)
}

type observer struct {
	done     chan struct{}
	canceled atomic.Int64
	received atomic.Int64
	success  atomic.Int64
	failure  atomic.Int64
	once     sync.Once
}

func newObserver() *observer {
	obs := new(observer)
	obs.done = make(chan struct{})

	return obs
}

func (o *observer) Listen(_ context.Context, _ *pusher.Worker, gossips <-chan *pusher.Gossip) {
	defer o.once.Do(func() {
		close(o.done)
	})

	for gossip := range gossips {
		if gossip.Canceled() {
			o.canceled.Add(1)

			continue
		}

		if gossip.BeforeTarget() {
			o.received.Add(1)

			continue
		}

		if gossip.Error != nil {
			o.failure.Add(1)
		} else {
			o.success.Add(1)
		}
	}
}

func (o *observer) Stop() {
	<-o.done
}

type sentry struct {
	done chan struct{}
}

func newSentry() *sentry {
	sent := new(sentry)
	sent.done = make(chan struct{})

	return sent
}

func (s *sentry) Listen(_ context.Context, _ *pusher.Worker, gossips <-chan *pusher.Gossip) {
	s.done = make(chan struct{})
	defer close(s.done)

	<-gossips
	<-gossips
}

func (s *sentry) Stop() {
	<-s.done
}

func runner(target pusher.Target, offers ...pusher.Offer) (*pusher.Worker, func(ctx context.Context, rps int) error) {
	worker := pusher.Hire("", target, offers...)
	run := func(ctx context.Context, rps int) error {
		err := worker.Work(ctx, rps)

		switch {
		case errors.Is(err, context.Canceled):
		case errors.Is(err, context.DeadlineExceeded):
			return nil
		}

		return ex.Cast(err)
	}

	return worker, run
}

func noop() pusher.Target {
	return func(_ context.Context) (pusher.Result, error) {
		return result("done"), nil
	}
}

func awaitable() pusher.Target {
	return func(ctx context.Context) (pusher.Result, error) {
		<-ctx.Done()

		return result("done"), nil
	}
}

func slow() pusher.Target {
	var (
		num   int
		mutex sync.Mutex
	)

	return func(_ context.Context) (pusher.Result, error) {
		mutex.Lock()
		defer mutex.Unlock()

		num++

		if num%2 == 0 {
			time.Sleep(time.Second)
		}

		return result("done"), nil
	}
}

func fuzzBuzz() pusher.Target {
	var (
		num   int
		mutex sync.Mutex
	)

	return func(_ context.Context) (pusher.Result, error) {
		mutex.Lock()
		defer mutex.Unlock()

		num++

		switch {
		case num%3 == 0:
			return nil, ex.ErrUnexpected
		case num%5 == 0:
			time.Sleep(time.Second)

			return result("busy"), nil
		default:
			return result("done"), nil
		}
	}
}
