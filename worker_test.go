package pusher_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/therenotomorrow/ex"

	"github.com/therenotomorrow/pusher"
)

var errDummy = errors.New("dummy")

func noop(context.Context) (pusher.Result, error) { return nil, errDummy }

type observer struct {
	worker    *pusher.Worker
	done      chan struct{}
	cancelled atomic.Int64
	received  atomic.Int64
	success   atomic.Int64
	failure   atomic.Int64
}

func (o *observer) Listen(_ context.Context, worker *pusher.Worker, gossips <-chan *pusher.Gossip) {
	o.worker = worker
	o.done = make(chan struct{})

	defer close(o.done)

	for gossip := range gossips {
		if gossip.Cancelled() {
			o.cancelled.Add(1)

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
	worker *pusher.Worker
	done   chan struct{}
}

func (s *sentry) Listen(_ context.Context, worker *pusher.Worker, gossips <-chan *pusher.Gossip) {
	s.worker = worker
	s.done = make(chan struct{})

	defer close(s.done)

	<-gossips
	<-gossips
}

func (s *sentry) Stop() {
	<-s.done
}

func safe(target pusher.Target, offers ...pusher.Offer) (*pusher.Worker, func(ctx context.Context, rps int) error) {
	worker := pusher.Hire("", target, offers...)

	return worker, func(ctx context.Context, rps int) error {
		err := worker.Work(ctx, rps)

		switch {
		case errors.Is(err, context.Canceled):
		case errors.Is(err, context.DeadlineExceeded):
			return nil
		}

		return ex.Cast(err)
	}
}

func TestWorkerValidateTarget(t *testing.T) {
	t.Parallel()

	worker, call := safe(nil)
	err := call(t.Context(), 1)

	require.ErrorIs(t, err, pusher.ErrMissingTarget)
	require.EqualError(t, err, "target is missing: not provided")

	assert.False(t, worker.Config().Busy)
}

func TestWorkerValidateRPS(t *testing.T) {
	t.Parallel()

	type args struct {
		rps int
	}

	type want struct {
		err string
	}

	worker, call := safe(noop)

	for _, test := range []struct {
		name string
		want want
		args args
	}{
		{name: "zero", args: args{rps: 0}, want: want{err: "invalid rps: must be positive"}},
		{name: "negative", args: args{rps: -42}, want: want{err: "invalid rps: must be positive"}},
		{name: "large", args: args{rps: 1e10}, want: want{err: "invalid rps: too large, resulting tick < 1ns"}},
	} {
		err := call(t.Context(), test.args.rps)

		require.ErrorIs(t, err, pusher.ErrInvalidRPS)
		require.EqualError(t, err, test.want.err)

		assert.False(t, worker.Config().Busy)
	}
}

func TestWorkerValidateOvertime(t *testing.T) {
	t.Parallel()

	worker, call := safe(noop, pusher.WithOvertime(-42))
	err := call(t.Context(), 1)

	require.ErrorIs(t, err, pusher.ErrInvalidOvertime)
	require.EqualError(t, err, "invalid overtime: must be more or equal zero")

	assert.False(t, worker.Config().Busy)
}

func TestWorkerValidateBusy(t *testing.T) {
	t.Parallel()

	var (
		size  = 5
		wait  = sync.WaitGroup{}
		mutex = sync.Mutex{}
		errs  = make([]error, 0)
	)

	worker, call := safe(func(ctx context.Context) (pusher.Result, error) {
		<-ctx.Done()

		return nil, errDummy
	})

	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()

	for range size {
		wait.Go(func() {
			err := call(ctx, 1)
			if err == nil {
				return
			}

			mutex.Lock()
			defer mutex.Unlock()

			errs = append(errs, err)
		})
	}

	wait.Wait()

	for _, err := range errs {
		require.ErrorIs(t, err, pusher.ErrWorkerIsBusy)
		require.Error(t, err, "worker is busy: try again later")
	}

	assert.Len(t, errs, size-1)
	assert.False(t, worker.Config().Busy)
}

func TestWorkerString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		ident string
		want  string
	}{
		{
			name:  "no name",
			ident: "",
			want:  "judas",
		},
		{
			name:  "somebody",
			ident: "somebody",
			want:  "somebody",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			worker := pusher.Hire(test.ident, nil)
			got := worker.String()

			assert.Equal(t, test.want, got)
		})
	}
}

func TestWorkerWorkFast(t *testing.T) {
	t.Parallel()

	var (
		num      = 0
		rps      = 100
		limit    = 5
		duration = 5 * time.Second
		wait     = sync.WaitGroup{}
		mutex    = sync.Mutex{}
		obs      = new(observer)
	)

	worker, call := safe(func(_ context.Context) (pusher.Result, error) {
		mutex.Lock()
		defer mutex.Unlock()

		num++

		switch {
		case num%3 == 0:
			return nil, errDummy
		case num%5 == 0:
			time.Sleep(time.Second)

			return result("busy"), nil
		default:
			return result("done"), nil
		}
	}, pusher.WithGossips(obs), pusher.WithOvertime(limit))

	ctx, cancel := context.WithTimeout(t.Context(), duration)
	defer cancel()

	wait.Go(func() {
		err := call(ctx, rps)

		require.NoError(t, err)
	})

	// wait for start goroutine and check business
	time.Sleep(duration / 5)

	assert.True(t, worker.Config().Busy)
	wait.Wait()
	assert.False(t, worker.Config().Busy)

	var (
		cancelled = int(obs.cancelled.Load())
		received  = int(obs.received.Load())
		success   = int(obs.success.Load())
		failure   = int(obs.failure.Load())
	)

	assert.Greater(t, cancelled, 400)
	assert.Greater(t, received, 30)
	assert.Greater(t, success, 20)
	assert.Less(t, failure, 20)
	assert.Greater(t, received, success+failure)
}

func TestWorkerWorkSlow(t *testing.T) {
	t.Parallel()

	var (
		num      = 0
		rps      = 1
		limit    = 1
		duration = 5 * time.Second
		mutex    = sync.Mutex{}
		sent     = new(sentry)
	)

	_, call := safe(func(_ context.Context) (pusher.Result, error) {
		mutex.Lock()
		defer mutex.Unlock()

		num++

		if num%2 == 0 {
			time.Sleep(time.Second)
		}

		return result("done"), nil
	}, pusher.WithGossips(sent), pusher.WithOvertime(limit))

	ctx, cancel := context.WithTimeout(t.Context(), duration)
	defer cancel()

	err := call(ctx, rps)

	require.NoError(t, err)
}
