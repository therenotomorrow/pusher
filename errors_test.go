package pusher_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/therenotomorrow/pusher"
)

func TestErrInvalidRPS(t *testing.T) {
	t.Parallel()

	assert.EqualError(t, pusher.ErrInvalidRPS, "invalid rps value")
}

func TestErrMissingTarget(t *testing.T) {
	t.Parallel()

	assert.EqualError(t, pusher.ErrMissingTarget, "target is missing")
}

func TestErrWorkerIsBusy(t *testing.T) {
	t.Parallel()

	assert.EqualError(t, pusher.ErrWorkerIsBusy, "worker is busy")
}

func TestErrInvalidOvertime(t *testing.T) {
	t.Parallel()

	assert.EqualError(t, pusher.ErrInvalidOvertime, "invalid overtime value")
}
