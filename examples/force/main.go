package main

import (
	"log"
	"time"

	"github.com/therenotomorrow/pusher"
	"github.com/therenotomorrow/pusher/examples"
)

func main() {
	// Create 10 workers with the same configuration
	log.Println(pusher.Force(examples.Target, 200, 5*time.Minute, 10))
}
