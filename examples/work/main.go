package main

import (
	"log"
	"time"

	"github.com/therenotomorrow/pusher"
	"github.com/therenotomorrow/pusher/examples"
)

func main() {
	// Run with 50 RPS for one minute
	log.Println(pusher.Work(examples.Target, 50, time.Minute))
}
