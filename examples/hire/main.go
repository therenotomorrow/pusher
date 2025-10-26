package main

import (
	"context"
	"log"
	"time"

	"github.com/therenotomorrow/pusher"
	"github.com/therenotomorrow/pusher/examples"
)

func main() {
	// Create a worker
	worker := pusher.Hire("body", examples.Target)

	rps := 59
	duration := time.Minute

	// Run with 50 RPS for 1 minute
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	log.Println(worker.Work(ctx, rps))
}
