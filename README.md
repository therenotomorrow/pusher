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

**pusher** is a Go library for simple and quick load testing, see more in [examples](./examples) folder.

The library is built around the **Worker** concept — the entity who works! Key components:

- **Worker** — the main character
- **Target** — what we want to test
- **Gossiper** — interface for listening something interesting
- **Gossip** — task lifecycle events
- **Offer** — options for hire the **Worker**

## Quick Start

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/therenotomorrow/pusher"
)

type Result string

func (r Result) String() string {
	return string(r)
}

func main() {
	// Your function to test
	target := func(ctx context.Context) (pusher.Result, error) {
		return Result(fmt.Sprintf("result at %v", time.Now())), nil
	}

	// Create a worker
	worker, err := pusher.Hire("body", target, pusher.WithOvertime(10))
	if err != nil {
		log.Fatalln(err)
	}

	// Run with 50 RPS for 1 minute
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	log.Println(worker.Work(ctx, 50))
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
just code # see other recipes by calling just without arguments

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

## License

MIT License. See [LICENSE](./LICENSE) file for details.
