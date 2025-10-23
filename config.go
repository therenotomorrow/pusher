package pusher

const (
	// double is a multiplier for the listener channel's buffer size.
	// A size of 2*rps provides a sufficient buffer to handle bursts
	// of BeforeTarget and AfterTarget events without blocking.
	double          = 2
	defaultOvertime = 1_000_000
)

type (
	config struct {
		listeners []Gossiper
		overtime  int
	}

	// Config is a public copy of the Worker internals.
	Config struct {
		Ident     string
		Listeners []Gossiper
		Overtime  int
		Busy      bool
	}

	// Offer is a functional option for configuring a Worker.
	// This pattern allows for flexible and extensible Worker initialization.
	Offer func(w *Worker)
)

// WithGossips configures a Worker with a set of event listeners.
func WithGossips(listeners ...Gossiper) Offer {
	return func(w *Worker) {
		w.config.listeners = listeners
	}
}

// WithOvertime sets the maximum number of concurrently executing tasks that a
// Worker can run. It acts as a concurrency limiter.
func WithOvertime(limit int) Offer {
	return func(w *Worker) {
		w.config.overtime = limit
	}
}

// Config returns the public copy of Worker internals.
func (w *Worker) Config() Config {
	return Config{
		Busy:      w.busy.Load(),
		Ident:     w.ident,
		Listeners: w.config.listeners,
		Overtime:  w.config.overtime,
	}
}
