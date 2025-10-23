package pusher_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/therenotomorrow/ex"

	"github.com/therenotomorrow/pusher"
)

var errDummy = errors.New("dummy")

func noop(context.Context) (pusher.Result, error) { return nil, errDummy }

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
