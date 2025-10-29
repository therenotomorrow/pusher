package main

import (
	"log"
	"time"

	"github.com/therenotomorrow/pusher"
	"github.com/therenotomorrow/pusher/examples"
)

func main() {
	rps := 200
	duration := time.Minute
	amount := 5

	// Create a runner with the set pre-requests
	runner := pusher.Force(rps, duration, examples.Target)

	// Create 10 workers with the same configuration
	log.Println(runner(amount))
}
