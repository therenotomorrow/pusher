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

var ErrOops = errors.New("oops")

func RandomTime(_ context.Context) (pusher.Result, error) {
	rnd, _ := rand.Int(rand.Reader, big.NewInt(100))
	flip := rnd.Uint64()

	if flip < 25 {
		time.Sleep(time.Second)
	}

	if flip < 50 {
		return nil, ErrOops
	}

	return Struct{Name: "lumen", Age: 42}, nil
}
