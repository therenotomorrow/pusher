package pusher

import (
	"log/slog"
	"time"

	"github.com/therenotomorrow/ex"
)

const (
	// ErrInvalidWorkers is returned when the number of workers is less than 1.
	ErrInvalidWorkers = ex.Error("must specify at least one Worker")

	// ErrInvalidDuration is returned when the duration is less than 1 second.
	ErrInvalidDuration = ex.Error("duration should be greater or equal second")

	// ErrInvalidRPS is returned when the RPS is less than 1.
	ErrInvalidRPS = ex.Error("must specify at least one rps")
)

// Option represents a configuration option for the Pusher.
type Option func(*Pusher)

// Config represents the configuration for a Pusher instance.
type Config struct {
	RPS      int
	Workers  int
	Duration time.Duration
}

func (c Config) Validate() error {
	if c.Workers < 1 {
		return ErrInvalidWorkers
	}

	if c.RPS < 1 {
		return ErrInvalidRPS
	}

	if c.Duration < time.Second {
		return ErrInvalidDuration
	}

	return nil
}

func defaults() Config {
	return Config{
		RPS:      1,
		Workers:  1,
		Duration: time.Second,
	}
}

// WithWorkers sets the number of workers for the Pusher.
// Each worker will handle a portion of the total RPS.
func WithWorkers(workers int) Option {
	return func(p *Pusher) {
		p.config.Workers = workers
	}
}

// WithRPS sets the requests per second rate for the Pusher.
// This rate will be distributed among all workers.
func WithRPS(rps int) Option {
	return func(p *Pusher) {
		p.config.RPS = rps
	}
}

// WithDuration sets the duration for which the load test will run.
// The test will stop after this duration or when the context is cancelled.
func WithDuration(duration time.Duration) Option {
	return func(p *Pusher) {
		p.config.Duration = duration
	}
}

// WithLogger sets a custom logger for the Pusher.
// If not provided, a no-op logger will be used.
func WithLogger(logger *slog.Logger) Option {
	return func(p *Pusher) {
		p.logger = logger
	}
}
