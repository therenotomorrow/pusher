package pusher

import (
	"github.com/therenotomorrow/ex"
)

const (
	// ErrInvalidRPS is returned when the provided requests-per-second (RPS) value is not positive.
	ErrInvalidRPS = ex.Error("invalid rps value")

	// ErrMissingTarget is returned when a Worker is hired without a Target function.
	ErrMissingTarget = ex.Error("target is missing")

	// ErrWorkerIsBusy is returned when Work is called on a Worker that is already running.
	ErrWorkerIsBusy = ex.Error("worker is busy")

	// ErrInvalidOvertime is returned when Work is tried to run with negative WithOvertime option.
	ErrInvalidOvertime = ex.Error("invalid overtime value")
)
