package pusher_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/therenotomorrow/pusher"
)

func TestHireWithDefaults(t *testing.T) {
	t.Parallel()

	worker := pusher.Hire("", nil)

	got := worker.Config()
	want := pusher.Config{
		Ident:       "judas",
		Listeners:   make([]pusher.Gossiper, 0),
		Overtime:    1_000_000,
		WLBCapacity: 1_000_000,
		Busy:        false,
	}

	assert.Equal(t, want, got)
}

func TestHireWithOffers(t *testing.T) {
	t.Parallel()

	var (
		ident     = "cozy"
		limit     = 100
		gossipers = []pusher.Gossiper{newSentry(), newObserver()}
		offers    = []pusher.Offer{pusher.WithGossips(gossipers...), pusher.WithOvertime(limit)}
		worker    = pusher.Hire(ident, fuzzBuzz(), offers...)
	)

	got := worker.Config()
	want := pusher.Config{
		Ident:       ident,
		Listeners:   gossipers,
		Overtime:    limit,
		WLBCapacity: limit,
		Busy:        false,
	}

	assert.Equal(t, want, got)
}

func TestHireNegativeOvertime(t *testing.T) {
	t.Parallel()

	var (
		ident  = "negative one"
		limit  = -42
		offers = []pusher.Offer{pusher.WithOvertime(limit)}
		worker = pusher.Hire(ident, nil, offers...)
	)

	got := worker.Config()
	want := pusher.Config{
		Ident:       ident,
		Listeners:   make([]pusher.Gossiper, 0),
		Overtime:    -42,
		WLBCapacity: 0,
		Busy:        false,
	}

	assert.Equal(t, want, got)
}

func TestWork(t *testing.T) {
	t.Parallel()

	type args struct {
		rps int
	}

	type want struct {
		err error
	}

	tests := []struct {
		want want
		name string
		args args
	}{
		{name: "success", args: args{rps: 1}, want: want{err: context.DeadlineExceeded}},
		{name: "failure", args: args{rps: -1}, want: want{err: pusher.ErrInvalidRPS}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			err := pusher.Work(test.args.rps, time.Second, noop())

			require.ErrorIs(t, err, test.want.err)
		})
	}
}

func TestFarm(t *testing.T) {
	t.Parallel()

	type args struct {
		worker *pusher.Worker
	}

	type want struct {
		err    error
		calls  int64
		strict bool
	}

	tests := []struct {
		args args
		name string
		want want
	}{
		{
			name: "success",
			args: args{worker: pusher.Hire("", awaitable())},
			want: want{calls: 22, strict: false, err: context.DeadlineExceeded},
		},
		{
			name: "failure",
			args: args{worker: pusher.Hire("", nil)},
			want: want{calls: 0, strict: true, err: pusher.ErrMissingTarget},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			var (
				rps      = 10
				limit    = 10
				duration = time.Second
				obs      = newObserver()
				obs1     = newObserver()
				obs2     = newObserver()
				obs3     = newObserver()
				workers  = []*pusher.Worker{
					pusher.Hire("#1", slow(), pusher.WithGossips(obs1, obs), pusher.WithOvertime(limit)),
					pusher.Hire("#2", fuzzBuzz(), pusher.WithGossips(obs2, obs), pusher.WithOvertime(limit)),
					pusher.Hire("#3", noop(), pusher.WithGossips(obs3, obs)),
				}
			)

			workers = append(workers, test.args.worker)
			err := pusher.Farm(rps, duration, workers)

			require.ErrorIs(t, err, test.want.err)

			if test.want.strict {
				assert.Equal(t, obs.received.Load(), test.want.calls)
			} else {
				assert.Greater(t, obs.received.Load(), test.want.calls)
			}
		})
	}
}

func TestForce(t *testing.T) {
	t.Parallel()

	type args struct {
		rps int
	}

	type want struct {
		err    error
		calls  int64
		strict bool
	}

	tests := []struct {
		name string
		want want
		args args
	}{
		{name: "success", args: args{rps: 10}, want: want{calls: 22, strict: false, err: context.DeadlineExceeded}},
		{name: "failure", args: args{rps: -42}, want: want{calls: 0, strict: true, err: pusher.ErrInvalidRPS}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			var (
				limit    = 10
				amount   = 3
				duration = time.Second
				obs      = newObserver()
				obs1     = newObserver()
				obs2     = newObserver()
				obs3     = newObserver()
			)

			run := pusher.Force(
				test.args.rps,
				duration,
				noop(),
				pusher.WithGossips(obs1, obs2, obs3, obs),
				pusher.WithOvertime(limit),
			)
			err := run(amount)

			require.ErrorIs(t, err, test.want.err)

			if test.want.strict {
				assert.Equal(t, obs.received.Load(), test.want.calls)
			} else {
				assert.Greater(t, obs.received.Load(), test.want.calls)
			}
		})
	}
}
