package main

import (
	"fmt"
	"log"
	"time"

	"github.com/therenotomorrow/pusher"
	"github.com/therenotomorrow/pusher/examples"
)

func main() {
	// Create multiple workers
	workers := make([]*pusher.Worker, 0)
	for i, target := range examples.Targets() {
		worker := pusher.Hire(fmt.Sprintf("somebody #%d", i), target)
		workers = append(workers, worker)
	}

	rps := 100
	duration := time.Minute

	// Run all workers in parallel
	log.Println(pusher.Farm(rps, duration, workers))
}
