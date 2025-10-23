package pusher_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/therenotomorrow/pusher"
)

type gossiper struct{}

func (g *gossiper) Listen(_ context.Context, _ *pusher.Worker, _ <-chan *pusher.Gossip) {}

func (g *gossiper) Stop() {}

func TestWithGossips(t *testing.T) {
	t.Parallel()

	t.Run("single", func(t *testing.T) {
		t.Parallel()

		var (
			worker    = new(pusher.Worker)
			gossiper1 = new(gossiper)
		)

		pusher.WithGossips(gossiper1)(worker)

		got := worker.Config().Listeners
		want := []pusher.Gossiper{gossiper1}

		assert.Equal(t, want, got)
	})

	t.Run("multiple", func(t *testing.T) {
		t.Parallel()

		var (
			worker    = new(pusher.Worker)
			gossiper1 = new(gossiper)
			gossiper2 = new(gossiper)
			gossiper3 = new(gossiper)
		)

		pusher.WithGossips(gossiper1, gossiper2, gossiper3)(worker)

		got := worker.Config().Listeners
		want := []pusher.Gossiper{gossiper1, gossiper2, gossiper3}

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
		gossipers = []pusher.Gossiper{new(gossiper), new(gossiper)}
		worker    = pusher.Hire(
			"memes", noop,
			pusher.WithOvertime(100),
			pusher.WithGossips(gossipers...),
		)
	)

	got := worker.Config()
	want := pusher.Config{
		Ident:     "memes",
		Listeners: gossipers,
		Overtime:  100,
		Busy:      false,
	}

	assert.Equal(t, want, got)
}
