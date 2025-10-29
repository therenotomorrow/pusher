package pusher_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/therenotomorrow/pusher"
)

func TestWorkerValidateTarget(t *testing.T) {
	t.Parallel()

	var (
		target pusher.Target
		worker = pusher.Hire("", target)
	)

	err := worker.Work(t.Context(), 1)

	require.ErrorIs(t, err, pusher.ErrMissingTarget)
	require.EqualError(t, err, "target is missing: not provided")

	got := worker.Config().Busy

	assert.False(t, got)
}

func TestWorkerValidateRPS(t *testing.T) {
	t.Parallel()

	worker := pusher.Hire("", noop())

	type args struct {
		rps int
	}

	type want struct {
		err string
	}

	tests := []struct {
		name string
		want want
		args args
	}{
		{name: "zero", args: args{rps: 0}, want: want{err: "invalid rps: must be positive"}},
		{name: "negative", args: args{rps: -42}, want: want{err: "invalid rps: must be positive"}},
		{name: "large", args: args{rps: 1e10}, want: want{err: "invalid rps: too large, resulting tick < 1ns"}},
	}

	for _, test := range tests {
		err := worker.Work(t.Context(), test.args.rps)

		require.ErrorIs(t, err, pusher.ErrInvalidRPS)
		require.EqualError(t, err, test.want.err)

		got := worker.Config().Busy

		assert.False(t, got)
	}
}

func TestWorkerValidateOvertime(t *testing.T) {
	t.Parallel()

	var (
		limit  = -42
		worker = pusher.Hire("", noop(), pusher.WithOvertime(limit))
	)

	err := worker.Work(t.Context(), 1)

	require.ErrorIs(t, err, pusher.ErrInvalidOvertime)
	require.EqualError(t, err, "invalid overtime: must be more or equal zero")

	got := worker.Config().Busy

	assert.False(t, got)
}

func TestWorkerValidateBusy(t *testing.T) {
	t.Parallel()

	var (
		runs  = 5
		wait  = sync.WaitGroup{}
		mutex = sync.Mutex{}
		errs  = make([]error, 0)
	)

	worker, run := runner(awaitable())

	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()

	for range runs {
		wait.Go(func() {
			err := run(ctx, 1)
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

	assert.Len(t, errs, runs-1)

	got := worker.Config().Busy

	assert.False(t, got)
}

func TestWorkerString(t *testing.T) {
	t.Parallel()

	type args struct {
		ident string
	}

	tests := []struct {
		name string
		args args
		want string
	}{
		{name: "no name", args: args{ident: ""}, want: "judas"},
		{name: "somebody", args: args{ident: "somebody"}, want: "somebody"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := pusher.Hire(test.args.ident, nil).String()

			assert.Equal(t, test.want, got)
		})
	}
}

func TestWorkerWorkFast(t *testing.T) {
	t.Parallel()

	var (
		rps      = 100
		limit    = 5
		duration = 5 * time.Second
		wait     = sync.WaitGroup{}
		obs      = newObserver()
	)

	worker, run := runner(fuzzBuzz(), pusher.WithGossips(obs), pusher.WithOvertime(limit))

	ctx, cancel := context.WithTimeout(t.Context(), duration)
	defer cancel()

	wait.Go(func() {
		err := run(ctx, rps)

		require.NoError(t, err)
	})

	// wait for start goroutine and check business
	time.Sleep(duration / 5)

	assert.True(t, worker.Config().Busy)
	wait.Wait()
	assert.False(t, worker.Config().Busy)

	var (
		canceled = int(obs.canceled.Load())
		received = int(obs.received.Load())
		success  = int(obs.success.Load())
		failure  = int(obs.failure.Load())
	)

	assert.Greater(t, canceled, 400)
	assert.Greater(t, received, 30)
	assert.Greater(t, success, 20)
	assert.Less(t, failure, 20)
	// because the shout method waits the context also
	assert.GreaterOrEqual(t, received, success+failure)
}

func TestWorkerWorkSlow(t *testing.T) {
	t.Parallel()

	var (
		rps      = 1
		limit    = 1
		duration = 5 * time.Second
		sent     = newSentry() // can handle only a few signals
	)

	_, run := runner(slow(), pusher.WithGossips(sent), pusher.WithOvertime(limit))

	ctx, cancel := context.WithTimeout(t.Context(), duration)
	defer cancel()

	err := run(ctx, rps)

	require.NoError(t, err)
}
