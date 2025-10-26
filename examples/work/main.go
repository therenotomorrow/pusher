package main

import (
	"log"
	"time"

	"github.com/therenotomorrow/pusher"
	"github.com/therenotomorrow/pusher/examples"
)

func main() {
	rps := 50
	duration := time.Minute

	// Run with 50 RPS for one minute
	log.Println(pusher.Work(examples.Target, rps, duration))
}
