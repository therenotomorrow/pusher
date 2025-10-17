package pusher

import (
	"context"
	"log/slog"
	"sync/atomic"

	"github.com/therenotomorrow/ex"
)

const (
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

type Target func(ctx context.Context) error

type Pusher struct {
	target  Target
	stats   *stats
	logger  *slog.Logger
	workers []worker
	config  config
}

func (p *Pusher) SetStats(err error) error {
	p.stats.total.Add(1)

	if err != nil {
		p.stats.failure.Add(1)
	} else {
		p.stats.success.Add(1)
	}

	return err
}

func (p *Pusher) LogError(err error, message string, args ...any) {
	if err == nil {
		return
	}

	p.logger.Info(message, append([]any{"error", err}, args...)...)
}

func (p *Pusher) DebugTarget(target Target) Target {
	return func(ctx context.Context) error {
		p.logger.Info("push")
		defer p.logger.Info("pull")

		return target(ctx)
	}
}

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

	if pusher.config.debug {
		pusher.target = pusher.DebugTarget(target)
	}

	pusher.setupWorkers()

	return pusher, nil
}

func Force(target Target, options ...Option) *Pusher {
	pusher, err := Push(target, options...)
	if err != nil {
		panic(err)
	}

	return pusher
}

func (p *Pusher) Run(ctx context.Context) {
	p.startupLog()

	ctx, cancel := context.WithTimeout(ctx, p.config.duration)
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
		base      = p.config.rps / p.config.workers
		remainder = p.config.rps % p.config.workers
	)

	for id := range p.config.workers {
		rps := base
		if remainder > 0 {
			rps++
			remainder--
		}

		p.workers = append(p.workers, worker{rps: rps, worker: Hire(id+1, p.target).WithObserver(p)})
	}
}

func (p *Pusher) startupLog() {
	distribution := make([]int, 0)
	for _, work := range p.workers {
		distribution = append(distribution, work.rps)
	}

	p.logger.Info("Test started.",
		"rps", p.config.rps,
		"duration", p.config.duration,
		"workers", len(p.workers),
		"distribution", distribution,
		"expected requests", p.config.rps*int(p.config.duration.Seconds()),
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
