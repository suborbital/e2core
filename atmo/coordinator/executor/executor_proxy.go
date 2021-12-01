//go:build proxy

package executor

import (
	"github.com/pkg/errors"

	"github.com/suborbital/atmo/directive"
	"github.com/suborbital/atmo/directive/executable"
	"github.com/suborbital/grav/discovery/local"
	"github.com/suborbital/grav/grav"
	"github.com/suborbital/grav/transport/websocket"
	"github.com/suborbital/reactr/rcap"
	"github.com/suborbital/reactr/rt"
	"github.com/suborbital/vektor/vk"
	"github.com/suborbital/vektor/vlog"
)

const (
	MsgTypeAtmoFnResult = "atmo.fnresult"
)

var (
	ErrExecutorNotConfigured = errors.New("executor not fully configured")
	ErrCannotHandle          = errors.New("cannot handle job")
)

// Executor is a facade over Grav and Reactr that allows executing local OR remote
// functions with a single call, ensuring there is no difference between them to the caller
type Executor struct {
	grav     *grav.Grav
	pod      *grav.Pod
	log      *vlog.Logger
	callback grav.MsgFunc
}

// New creates a new Executor
func New(log *vlog.Logger, transport *websocket.Transport) *Executor {
	gravOpts := []grav.OptionsModifier{
		grav.UseLogger(log),
	}

	if transport != nil {
		d := local.New()

		gravOpts = append(gravOpts, grav.UseTransport(transport))
		gravOpts = append(gravOpts, grav.UseDiscovery(d))
	}

	g := grav.New(gravOpts...)

	// Reactr is configured in UseCapabiltyConfig
	e := &Executor{
		grav: g,
		pod:  g.Connect(),
		log:  log,
	}

	return e
}

// Do executes a remote job
func (e *Executor) Do(jobType string, data interface{}, ctx *vk.Ctx) (interface{}, error) {
	var runErr *rt.RunErr
	var cbErr error

	pod := e.grav.Connect()
	defer pod.Disconnect()

	e.log.Info("proxying function", jobType)

	completed := make(chan bool)

	// start listening to the messages produced by peers
	// on the network, and don't stop until there's an error
	// or the Sequence we're connected to deems that its complete
	pod.On(func(msg grav.Message) error {
		if msg.ParentID() != ctx.RequestID() {
			return nil
		} else if msg.Type() != MsgTypeAtmoFnResult {
			return nil
		}

		// if a callback is registered, send it up the chain
		// (probably to the Sequence object that called Do)
		if e.callback != nil {
			// the Sequence callback will return an error under two main conditions:
			// - The sequence has ended: yay!
			// - The result we just got caused an error: boo :(
			// either way, show's over and we send on `completed`

			if err := e.callback(msg); err != nil {
				if err == executable.ErrSequenceCompleted {
					// do nothing! that's great!
				} else if cbRunErr, isRunErr := err.(*rt.RunErr); isRunErr {
					// handle the runErr
					runErr = cbRunErr
				} else {
					// nothing we really can do here, but let's propogate it
					cbErr = err
				}

				completed <- true
			}
		}

		return nil
	})

	pod.Send(grav.NewMsgWithParentID(jobType, ctx.RequestID(), data.([]byte)))

	// wait until the sequence completes or errors
	<-completed

	// checking this explicitly because somehow Go interprets an
	// un-instantiated literal pointer as a non-nil error interface
	if cbErr != nil {
		return nil, cbErr
	} else if runErr != nil {
		return nil, runErr
	}

	e.log.Info("proxied execution", jobType, "fulfilled by peer")

	// getting the JobResult was done by the callback, return nothing
	return nil, nil
}

// UseCapabilityConfig sets up the executor's Reactr instance using the provided capability configuration
func (e *Executor) UseCapabilityConfig(config rcap.CapabilityConfig) error {
	// nothing to do in proxy mode

	return nil
}

// Register registers a Runnable
func (e *Executor) Register(jobType string, runner rt.Runnable, opts ...rt.Option) error {
	// nothing to do in proxy mode

	return nil
}

// SetSchedule adds a Schedule to the executor's Reactr instance
func (e *Executor) SetSchedule(sched rt.Schedule) error {
	// nothing to do in proxy mode

	return nil
}

// Load loads Runnables into the executor's Reactr instance
// And connects them to the Grav instance (currently unused)
func (e *Executor) Load(runnables []directive.Runnable) error {
	// nothing to do in proxy mode

	return nil
}

// UseCallback sets a function to be called on receipt of a message
func (e *Executor) UseCallback(callback grav.MsgFunc) {
	e.callback = callback
}

// Metrics returns the executor's Reactr isntance's internal metrics
func (e *Executor) Metrics() (*rt.ScalerMetrics, error) {
	// nothing to do in proxy mode

	return nil, nil
}