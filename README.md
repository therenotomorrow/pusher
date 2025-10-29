# pusher

<div>
  <a href="https://github.com/therenotomorrow/pusher/releases" target="_blank">
    <img src="https://img.shields.io/github/v/release/therenotomorrow/pusher?color=FBC02D" alt="GitHub releases">
  </a>
  <a href="https://go.dev/doc/go1.25" target="_blank">
    <img src="https://img.shields.io/badge/Go-%3E%3D%201.25-blue.svg" alt="Go 1.21">
  </a>
  <a href="https://pkg.go.dev/github.com/therenotomorrow/pusher" target="_blank">
    <img src="https://godoc.org/github.com/therenotomorrow/pusher?status.svg" alt="Go reference">
  </a>
  <a href="https://github.com/therenotomorrow/pusher/blob/master/LICENSE" target="_blank">
    <img src="https://img.shields.io/github/license/therenotomorrow/pusher?color=388E3C" alt="License">
  </a>
  <a href="https://github.com/therenotomorrow/pusher/actions/workflows/ci.yml" target="_blank">
    <img src="https://github.com/therenotomorrow/pusher/actions/workflows/ci.yml/badge.svg" alt="ci status">
  </a>
  <a href="https://goreportcard.com/report/github.com/therenotomorrow/pusher" target="_blank">
    <img src="https://goreportcard.com/badge/github.com/therenotomorrow/pusher" alt="Go report">
  </a>
  <a href="https://codecov.io/gh/therenotomorrow/pusher" target="_blank">
    <img src="https://img.shields.io/codecov/c/github/therenotomorrow/pusher?color=546E7A" alt="Codecov">
  </a>
</div>

**pusher** is a Go library for simple and quick load testing, see more in the [examples](./examples) folder.

The library is built around the **Worker** concept — the entity who works! Key elements:

- **Worker** — the main character
- **Target** — what we want to test
- **Gossiper** — interface for listening to something interesting
- **Gossip** — task lifecycle events
- **Offer** — options for hire the **Worker**

## Quick Start

```go
package main

import (
	"context"
	"time"

	"github.com/therenotomorrow/pusher"
)

func main() {
	target := func(_ context.Context) (pusher.Result, error) {
		return time.Now(), nil // time.Time support pusher.Result interface
	}

	// run target with 50 RPS for 1 minute
	_ = pusher.Work(50, time.Minute, target)
}
```

## Slow Start

```go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/therenotomorrow/pusher"
)

func main() {
	// Your function to test
	target := func(ctx context.Context) (pusher.Result, error) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		now := time.Now().UTC()
		if now.Second()%2 == 0 {
			time.Sleep(100 * time.Millisecond)
		}

		return Result(now.String()), nil
	}

	// Some of your gossips listeners
	observers := make([]*Observer, 0)

	gossipers := make([]pusher.Gossiper, 0)
	for _, when := range []pusher.When{pusher.Canceled, pusher.BeforeTarget, pusher.AfterTarget} {
		observer := &Observer{done: make(chan struct{}), when: when, count: 0}

		observers = append(observers, observer)
		gossipers = append(gossipers, observer)
	}

	// run target with 100 RPS for 1 minute with max 10 requests concurrent
	// and add 3 listeners for collect statistics
	err := pusher.Work(
		100,                              // rps
		time.Minute,                      // duration
		target,                           // target
		pusher.WithOvertime(10),          // concurrent limit
		pusher.WithGossips(gossipers...), // our gossipers
	)

	// check what was done and collect
	fmt.Println("We're done with error:", err)
	fmt.Println("Canceled:", observers[0].count)
	fmt.Println("Received:", observers[1].count)
	fmt.Println("Processed:", observers[2].count)

	// Output:
	// We're done with error: context deadline exceeded
	// Canceled: 256
	// Received: 5744
	// Processed: 5744
}

type Result string

func (r Result) String() string {
	return string(r)
}

type Observer struct {
	done  chan struct{}
	when  pusher.When
	count int
}

func (o *Observer) Listen(_ context.Context, _ *pusher.Worker, gossips <-chan *pusher.Gossip) {
	defer close(o.done)

	for gossip := range gossips {
		if gossip.When == o.when {
			o.count++
		}
	}
}

func (o *Observer) Stop() {
	<-o.done
}
```

## Development

### System Requirements

```shell
go version
# go version go1.25.2 or higher

just --version
# just 1.42.4 or higher (https://just.systems/)
```

### Download sources

```shell
PROJECT_ROOT=pusher
git clone https://github.com/therenotomorrow/pusher.git "$PROJECT_ROOT"
cd "$PROJECT_ROOT"
```

### Setup dependencies

```shell
# install dependencies
go mod tidy

# check code integrity
just code test # see other recipes by calling `just`

# setup safe development (optional)
git config --local core.hooksPath .githooks
```

### Project Structure

```
pusher/
├── config.go   # Configuration and functional options
├── errors.go   # Error definitions
├── gossip.go   # Event system and telemetry
├── pusher.go   # Main API and high-level functions
└── worker.go   # Worker implementation and execution logic
```

### Testing

```shell
# run quick checks
just test smoke # or just test

# run with coverage
just test cover
```

## License

MIT License. See the [LICENSE](./LICENSE) file for details.
