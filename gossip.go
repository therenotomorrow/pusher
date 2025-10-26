package pusher

const (
	// BeforeTarget is the moment just before the Target function is called.
	BeforeTarget When = "before-target"

	// AfterTarget is the moment just after the Target function returns.
	AfterTarget When = "after-target"

	// Cancelled indicates that a scheduled task was skipped because the concurrency
	// limit was reached.
	Cancelled When = "cancelled"
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
)

// Cancelled returns true if the Gossip event represents a cancelled task.
func (g *Gossip) Cancelled() bool {
	return g.When == Cancelled
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
