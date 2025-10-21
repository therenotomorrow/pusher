package main

import (
	"context"
	"fmt"
	"log"
	"sync/atomic"
	"time"

	"github.com/therenotomorrow/pusher"
	"github.com/therenotomorrow/pusher/examples"
)

// MetricsCollector is a custom event listener.
type MetricsCollector struct {
	done     chan struct{}
	name     string
	canceled atomic.Int64
	received atomic.Int64
	success  atomic.Int64
	failure  atomic.Int64
}

func (m *MetricsCollector) Listen(_ context.Context, worker *pusher.Worker, gossips <-chan *pusher.Gossip) {
	m.name = worker.String()
	m.done = make(chan struct{})

	defer close(m.done)

	for gossip := range gossips {
		if gossip.Cancelled() {
			m.canceled.Add(1)

			continue
		}

		if gossip.BeforeTarget() {
			m.received.Add(1)

			continue
		}

		if gossip.AfterTarget() {
			if gossip.Error != nil {
				m.failure.Add(1)
			} else {
				m.success.Add(1)
			}
		}
	}
}

func (m *MetricsCollector) Stop() {
	<-m.done
}

func (m *MetricsCollector) Stats() string {
	return fmt.Sprintf("Received: %d, Canceled: %d, Success: %d, Errors: %d",
		m.received.Load(),
		m.canceled.Load(),
		m.success.Load(),
		m.failure.Load(),
	)
}

func main() {
	collector := new(MetricsCollector)

	// Usage with metrics and overtime (max concurrent requests)
	worker, err := pusher.Hire("monitor", examples.RandomTime,
		pusher.WithGossips(collector),
		pusher.WithOvertime(10),
	)
	if err != nil {
		log.Fatalln(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	log.Println(worker.Work(ctx, 50))

	collector.Stop()

	log.Println(collector.Stats())
}
