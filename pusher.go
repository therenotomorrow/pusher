// Package pusher provides tools for load testing by repeatedly calling a given
// target function.
package pusher

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/therenotomorrow/ex"
	"golang.org/x/sync/errgroup"
)

// Hire creates and configures a new Worker instance using functional options.
func Hire(ident string, target Target, offers ...Offer) (*Worker, error) {
	if target == nil {
		return nil, ErrMissingTarget
	}

	worker := &Worker{
		ident:  ident,
		target: target,
		config: config{
			overtime:  defaultOvertime,
			listeners: make([]Gossiper, 0),
		},
		wlb:  nil, // initialized after all options are applied
		wait: sync.WaitGroup{},
		busy: atomic.Bool{},
	}

	for _, offer := range offers {
		offer(worker)
	}

	worker.wlb = make(chan struct{}, worker.config.overtime)

	return worker, nil
}

// Work is a convenience wrapper that creates and runs a single Worker
// for a specified duration.
func Work(target Target, rps int, duration time.Duration, offers ...Offer) error {
	worker, err := Hire("judas", target, offers...)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	return worker.Work(ctx, rps)
}

// Farm runs a set of pre-configured workers in parallel.
// It uses an errgroup to manage their lifecycle, ensuring that if one worker
// fails, the context is canceled for all.
func Farm(workers []*Worker, rps int, duration time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	group, gtx := errgroup.WithContext(ctx)

	for _, worker := range workers {
		group.Go(func() error {
			return worker.Work(gtx, rps)
		})
	}

	return ex.Cast(group.Wait())
}

// Force is a high-level wrapper that creates a specified number of workers
// with the same configuration and runs them as a Farm.
func Force(target Target, rps int, duration time.Duration, amount int, offers ...Offer) error {
	workers := make([]*Worker, amount)
	for ident := range workers {
		worker, err := Hire(fmt.Sprintf("force #%d", ident), target, offers...)
		if err != nil {
			return err
		}

		workers[ident] = worker
	}

	return Farm(workers, rps, duration)
}
