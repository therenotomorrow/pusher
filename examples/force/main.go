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

	// Create 10 workers with the same configuration
	log.Println(pusher.Force(examples.Target, rps, duration, amount))
}
