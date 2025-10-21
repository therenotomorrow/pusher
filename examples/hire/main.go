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
	worker, err := pusher.Hire("body", examples.Target)
	if err != nil {
		log.Fatalln(err)
	}

	// Run with 50 RPS for 1 minute
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	log.Println(worker.Work(ctx, 50))
}
