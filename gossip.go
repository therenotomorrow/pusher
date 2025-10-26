package pusher

import "context"

const (
	// BeforeTarget is the moment just before the Target function is called.
	BeforeTarget When = "before-target"

	// AfterTarget is the moment just after the Target function returns.
	AfterTarget When = "after-target"

	// Canceled indicates that a scheduled task was skipped because the concurrency
	// limit was reached.
	Canceled When = "canceled"
)

type (
	// When defines the stage of a task's lifecycle at which a Gossip event is generated.
	When string

	// Gossip represents a telemetry event generated during a Worker's operation.
	// It contains the result, an error, and the task lifecycle stage.
	Gossip struct {
		Result Result
		Error  error
		When   When
	}

	// Gossiper defines the interface for listeners that process Gossip events.
	// This allows plugging in various metric collectors, loggers, or reporters.
	Gossiper interface {
		// Listen runs in its own goroutine and processes events from the gossip channel.
		Listen(ctx context.Context, worker *Worker, gossips <-chan *Gossip)

		// Stop is called to gracefully shut down the listener and flush any buffered data.
		Stop()
	}
)

// Canceled returns true if the Gossip event represents a canceled task.
func (g *Gossip) Canceled() bool {
	return g.When == Canceled
}

// BeforeTarget returns true if the Gossip event occurred before the target execution.
func (g *Gossip) BeforeTarget() bool {
	return g.When == BeforeTarget
}

// AfterTarget returns true if the Gossip event occurred after the target execution.
func (g *Gossip) AfterTarget() bool {
	return g.When == AfterTarget
}

func (g *Gossip) String() string {
	if g == nil {
		return "<nil>"
	}

	if g.Result == nil {
		return "<empty>"
	}

	return g.Result.String()
}
