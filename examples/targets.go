package examples

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"time"

	"github.com/therenotomorrow/pusher"
)

const (
	answer  = 42
	quarter = 25
	half    = 50
	full    = 100
)

type Int int

func (r Int) String() string {
	return strconv.Itoa(int(r))
}

func Target(_ context.Context) (pusher.Result, error) {
	return Int(answer), nil
}

type Str string

func (r Str) String() string {
	return string(r)
}

func Targets() []pusher.Target {
	template := func(name string) pusher.Target {
		return func(ctx context.Context) (pusher.Result, error) {
			return Str(name), nil
		}
	}

	return []pusher.Target{
		template("judas"),
		template("alejandro"),
		template("wayne"),
	}
}

type Struct struct {
	Name string
	Age  int
}

func (r Struct) String() string {
	return fmt.Sprintf("Somebody %q at age %d", r.Name, r.Age)
}

var errOops = errors.New("oops")

func RandomTime(_ context.Context) (pusher.Result, error) {
	rnd, _ := rand.Int(rand.Reader, big.NewInt(full))
	flip := rnd.Uint64()

	if flip < quarter {
		time.Sleep(time.Second)
	}

	if flip < half {
		return nil, errOops
	}

	return Struct{Name: "lumen", Age: answer}, nil
}
