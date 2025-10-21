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
		worker, err := pusher.Hire(fmt.Sprintf("somebody #%d", i), target)
		if err != nil {
			log.Fatalln(err)
		}

		workers = append(workers, worker)
	}

	// Run all workers in parallel
	log.Println(pusher.Farm(workers, 100, 2*time.Minute))
}
