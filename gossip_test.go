package pusher_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/therenotomorrow/pusher"
)

func TestGossipStates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		when pusher.When
		want []bool
	}{
		{name: "canceled", when: pusher.Canceled, want: []bool{true, false, false}},
		{name: "before target", when: pusher.BeforeTarget, want: []bool{false, true, false}},
		{name: "after target", when: pusher.AfterTarget, want: []bool{false, false, true}},
		{name: "unsupported", when: pusher.When("unsupported"), want: []bool{false, false, false}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			gossip := pusher.Gossip{When: test.when, Result: nil, Error: nil}

			got := []bool{gossip.Canceled(), gossip.BeforeTarget(), gossip.AfterTarget()}

			assert.Equal(t, test.want, got)
		})
	}
}

func TestGossipString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		gossip *pusher.Gossip
		want   string
	}{
		{name: "nil", gossip: nil, want: "<nil>"},
		{name: "empty", gossip: &pusher.Gossip{Result: nil, Error: nil, When: pusher.BeforeTarget}, want: "<empty>"},
		{
			name:   "smoke",
			gossip: &pusher.Gossip{Result: result("useful"), Error: nil, When: pusher.AfterTarget},
			want:   "useful",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := test.gossip.String()

			assert.Equal(t, test.want, got)
		})
	}
}
