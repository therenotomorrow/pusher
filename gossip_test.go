package pusher_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/therenotomorrow/pusher"
)

type result string

func (m result) String() string { return string(m) }

func TestGossipWhen(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		when pusher.When
		want []bool
	}{
		{
			name: "Cancelled",
			when: pusher.Cancelled,
			want: []bool{true, false, false},
		},
		{
			name: "BeforeTarget",
			when: pusher.BeforeTarget,
			want: []bool{false, true, false},
		},
		{
			name: "AfterTarget",
			when: pusher.AfterTarget,
			want: []bool{false, false, true},
		},
		{
			name: "Unsupported",
			when: pusher.When("unsupported"),
			want: []bool{false, false, false},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			var (
				result = new(result)
				gossip = pusher.Gossip{When: test.when, Result: result, Error: nil}
			)

			got := []bool{gossip.Cancelled(), gossip.BeforeTarget(), gossip.AfterTarget()}

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
		{
			name:   "nil",
			gossip: nil,
			want:   "<nil>",
		},
		{
			name:   "empty",
			gossip: &pusher.Gossip{Result: nil, Error: nil, When: pusher.BeforeTarget},
			want:   "<empty>",
		},
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
