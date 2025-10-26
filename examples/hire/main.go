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

	// Run with 50 RPS for 1 minute
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	log.Println(worker.Work(ctx, 50))
}
