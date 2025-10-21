package examples

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strconv"

	"github.com/therenotomorrow/pusher"
)

type Int int

func (r Int) String() string {
	return strconv.Itoa(int(r))
}

func Target(_ context.Context) (pusher.Result, error) {
	return Int(42), nil
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

func Random(_ context.Context) (pusher.Result, error) {
	flip := rand.Uint32() % 100

	if flip < 50 {
		return nil, errors.New("oops")
	}

	return Struct{Name: "lumen", Age: 42}, nil
}
