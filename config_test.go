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

		got := worker.Unsafe()["listeners"]
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

		got := worker.Unsafe()["listeners"]
		want := []pusher.Gossiper{gossiper1, gossiper2, gossiper3}

		assert.Equal(t, want, got)
	})

	t.Run("empty", func(t *testing.T) {
		t.Parallel()

		worker := new(pusher.Worker)

		pusher.WithGossips()(worker)

		got := worker.Unsafe()["listeners"]

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

	got := worker.Unsafe()["overtime"]
	want := limit

	assert.Equal(t, want, got)
}
