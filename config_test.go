package pusher_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/therenotomorrow/pusher"
)

func TestWithGossips(t *testing.T) {
	t.Parallel()

	t.Run("single", func(t *testing.T) {
		t.Parallel()

		var (
			worker = new(pusher.Worker)
			obs    = newObserver()
		)

		pusher.WithGossips(obs)(worker)

		got := worker.Config().Listeners
		want := []pusher.Gossiper{obs}

		assert.Equal(t, want, got)
	})

	t.Run("multiple", func(t *testing.T) {
		t.Parallel()

		var (
			worker = new(pusher.Worker)
			obs1   = newObserver()
			obs2   = newObserver()
			obs3   = newObserver()
		)

		pusher.WithGossips(obs1, obs2, obs3)(worker)

		got := worker.Config().Listeners
		want := []pusher.Gossiper{obs1, obs2, obs3}

		assert.Equal(t, want, got)
	})

	t.Run("empty", func(t *testing.T) {
		t.Parallel()

		worker := new(pusher.Worker)

		pusher.WithGossips()(worker)

		got := worker.Config().Listeners

		assert.Empty(t, got)
	})
}

func TestWithOvertime(t *testing.T) {
	t.Parallel()

	var (
		limit  = 42
		worker = new(pusher.Worker)
	)

	pusher.WithOvertime(limit)(worker)

	got := worker.Config().Overtime
	want := limit

	assert.Equal(t, want, got)
}

func TestWorkerConfig(t *testing.T) {
	t.Parallel()

	var (
		ident     = "memes"
		limit     = 100
		gossipers = []pusher.Gossiper{newObserver(), newSentry()}
		worker    = pusher.Hire(ident, noop(), pusher.WithOvertime(limit), pusher.WithGossips(gossipers...))
	)

	got := worker.Config()
	want := pusher.Config{
		Ident:       ident,
		Listeners:   gossipers,
		Overtime:    limit,
		Busy:        false,
		WLBCapacity: limit,
	}

	assert.Equal(t, want, got)
}
