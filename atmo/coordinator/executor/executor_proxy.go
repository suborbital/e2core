//go:build proxy

package executor

import (
	"encoding/json"
	"sync"

	"github.com/pkg/errors"

	"github.com/suborbital/atmo/directive"
	"github.com/suborbital/grav/discovery/local"
	"github.com/suborbital/grav/grav"
	"github.com/suborbital/grav/transport/websocket"
	"github.com/suborbital/reactr/rcap"
	"github.com/suborbital/reactr/rt"
	"github.com/suborbital/vektor/vk"
	"github.com/suborbital/vektor/vlog"
)

var (
	ErrExecutorNotConfigured = errors.New("executor not fully configured")
	ErrCannotHandle          = errors.New("cannot handle job")
)

// Executor is a facade over Grav and Reactr that allows executing local OR remote
// functions with a single call, ensuring there is no difference between them to the caller
type Executor struct {
	grav *grav.Grav

	pod *grav.Pod

	log *vlog.Logger

	listening sync.Map
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
		grav:      g,
		pod:       g.Connect(),
		log:       log,
		listening: sync.Map{},
	}

	return e
}

// Do executes a remote job
func (e *Executor) Do(jobType string, data interface{}, ctx *vk.Ctx) (interface{}, error) {
	var jobResult []byte
	var runErr *rt.RunErr

	pod := e.grav.Connect()
	defer pod.Disconnect()

	podErr := pod.Send(grav.NewMsgWithParentID(jobType, ctx.RequestID(), nil)).WaitUntil(grav.Timeout(30), func(msg grav.Message) error {
		switch msg.Type() {
		case rt.MsgTypeReactrResult:
			// if the Runnable returned a result
			jobResult = msg.Data()
		case rt.MsgTypeReactrRunErr:
			// if the Runnable itself returned an error
			runErr = &rt.RunErr{}
			if err := json.Unmarshal(msg.Data(), runErr); err != nil {
				return errors.Wrap(err, "failed to Unmarshal RunErr")
			}
		case rt.MsgTypeReactrJobErr:
			// if something else caused an error while running this fn
			return errors.New(string(msg.Data()))
		case rt.MsgTypeReactrNilResult:
			// if the Runnable returned nil, do nothing
		}

		return nil
	})

	if podErr != nil {
		if podErr == grav.ErrWaitTimeout {
			return nil, errors.Wrapf(podErr, "fn %s timed out", jobType)
		}

		return nil, errors.Wrapf(podErr, "failed to execute fn %s", jobType)
	}

	return jobResult, runErr
}

// UseCapabilityConfig sets up the executor's Reactr instance using the provided capability configuration
func (e *Executor) UseCapabilityConfig(config rcap.CapabilityConfig) {
	// nothing to do in proxy mode
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

// Metrics returns the executor's Reactr isntance's internal metrics
func (e *Executor) Metrics() (*rt.ScalerMetrics, error) {
	// nothing to do in proxy mode

	return nil, nil
}
