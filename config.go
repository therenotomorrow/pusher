package pusher

import (
	"log/slog"
	"time"

	"github.com/therenotomorrow/ex"
)

const (
	ErrInvalidWorkers  = ex.Error("must specify at least one Worker")
	ErrInvalidDuration = ex.Error("duration should be greater or equal second")
	ErrInvalidRPS      = ex.Error("must specify at least one rps")
)

type Option func(*Pusher)

type config struct {
	rps      int
	workers  int
	duration time.Duration
}

func (c config) Validate() error {
	if c.workers < 1 {
		return ErrInvalidWorkers
	}

	if c.rps < 1 {
		return ErrInvalidRPS
	}

	if c.duration < time.Second {
		return ErrInvalidDuration
	}

	return nil
}

func defaults() config {
	return config{
		rps:      1,
		workers:  1,
		duration: time.Second,
	}
}

func WithWorkers(workers int) Option {
	return func(p *Pusher) {
		p.config.workers = workers
	}
}

func WithRPS(rps int) Option {
	return func(p *Pusher) {
		p.config.rps = rps
	}
}

func WithDuration(duration time.Duration) Option {
	return func(p *Pusher) {
		p.config.duration = duration
	}
}

func WithLogger(logger *slog.Logger) Option {
	return func(p *Pusher) {
		p.logger = logger
	}
}
