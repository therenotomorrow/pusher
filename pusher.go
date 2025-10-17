package pusher

import (
	"context"
	"log/slog"
	"sync/atomic"

	"github.com/therenotomorrow/ex"
)

const (
	// ErrMissingTarget is returned when a nil target function is provided to Push.
	ErrMissingTarget = ex.Error("target is missing")
)

type stats struct {
	total   atomic.Int64
	success atomic.Int64
	failure atomic.Int64
}

type worker struct {
	worker *Worker
	rps    int
}

// Target represents a function that will be executed during load testing.
// It receives a context and should return an error if the operation fails.
type Target func(ctx context.Context) error

// Pusher represents a load testing pusher that can execute
// a target function at a specified RPS using multiple workers.
type Pusher struct {
	target  Target
	stats   *stats
	logger  *slog.Logger
	workers []worker
	config  Config
}

// SetStats updates the internal statistics with the result of a target execution.
// It increments the total counter and either success or failure counter based on the error.
func (p *Pusher) SetStats(err error) error {
	p.stats.total.Add(1)

	if err != nil {
		p.stats.failure.Add(1)
	} else {
		p.stats.success.Add(1)
	}

	return err
}

// LogError logs an error message with additional context if the error is not nil.
// It uses the Error log level for proper error categorization.
func (p *Pusher) LogError(err error, message string, args ...any) {
	if err == nil {
		return
	}

	p.logger.Error(message, append([]any{"error", err}, args...)...)
}

// GetConfig returns the current configuration of the Pusher.
func (p *Pusher) GetConfig() Config {
	return p.config
}

// Push creates a new Pusher instance with the given target function
// and configuration options. Returns an error if the target is nil or
// if the configuration is invalid.
func Push(target Target, options ...Option) (*Pusher, error) {
	if target == nil {
		return nil, ErrMissingTarget
	}

	pusher := &Pusher{
		target:  target,
		config:  defaults(),
		stats:   new(stats),
		workers: make([]worker, 0),
		logger:  slog.New(slog.DiscardHandler),
	}

	for _, option := range options {
		option(pusher)
	}

	err := pusher.config.Validate()
	if err != nil {
		return nil, err
	}

	pusher.setupWorkers()

	return pusher, nil
}

// Force creates a new Pusher instance and panics if there's an error.
// This is a convenience function for cases where you want to fail fast
// instead of handling the error explicitly.
func Force(target Target, options ...Option) *Pusher {
	pusher, err := Push(target, options...)
	if err != nil {
		panic(err)
	}

	return pusher
}

// Run starts the load testing process. It creates workers, distributes RPS
// among them, and executes the target function according to the configured rate.
// The process runs until the context is cancelled or the duration expires.
func (p *Pusher) Run(ctx context.Context) {
	p.startupLog()

	ctx, cancel := context.WithTimeout(ctx, p.config.Duration)
	defer cancel()

	for _, work := range p.workers {
		go work.worker.Do(ctx, work.rps)
	}

	p.logger.Info("Processing...")

	// Ожидаем завершения всех воркеров
	for _, work := range p.workers {
		work.worker.Wait()
	}

	p.shutdownLog()
}

func (p *Pusher) setupWorkers() {
	var (
		base      = p.config.RPS / p.config.Workers
		remainder = p.config.RPS % p.config.Workers
	)

	for id := range p.config.Workers {
		rps := base
		if remainder > 0 {
			rps++
			remainder--
		}

		p.workers = append(p.workers, worker{rps: rps, worker: Hire(id, p.target).WithObserver(p)})
	}
}

func (p *Pusher) startupLog() {
	distribution := make([]int, 0)
	for _, work := range p.workers {
		distribution = append(distribution, work.rps)
	}

	p.logger.Info("Test started.",
		"rps", p.config.RPS,
		"duration", p.config.Duration,
		"workers", len(p.workers),
		"distribution", distribution,
		"expected requests", p.config.RPS*int(p.config.Duration.Seconds()),
	)
	p.logger.Info("------------------------------------")
}

func (p *Pusher) shutdownLog() {
	p.logger.Info("------------------------------------")
	p.logger.Info("Test finished.",
		"total", p.stats.total.Load(),
		"success", p.stats.success.Load(),
		"failure", p.stats.failure.Load(),
	)
}
