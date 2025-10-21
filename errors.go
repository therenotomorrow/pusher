package pusher

import (
	"github.com/therenotomorrow/ex"
)

const (
	// ErrInvalidRPS is returned when the provided requests-per-second (RPS) value is not positive.
	ErrInvalidRPS = ex.Error("rps must be positive value")

	// ErrMissingTarget is returned when a Worker is hired without a Target function.
	ErrMissingTarget = ex.Error("target is missing")

	// ErrWorkerIsBusy is returned when Work is called on a Worker that is already running.
	ErrWorkerIsBusy = ex.Error("worker is busy")
)
