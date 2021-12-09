//go:build !proxy

package executor

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/suborbital/atmo/bundle/load"
	"github.com/suborbital/atmo/directive"
	"github.com/suborbital/atmo/directive/executable"
	"github.com/suborbital/grav/discovery/local"
	"github.com/suborbital/grav/grav"
	"github.com/suborbital/grav/transport/websocket"
	"github.com/suborbital/reactr/rcap"
	"github.com/suborbital/reactr/request"
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
	reactr *rt.Reactr
	grav   *grav.Grav

	pod *grav.Pod

	log *vlog.Logger
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

	return NewWithGrav(log, g)
}

func NewWithGrav(log *vlog.Logger, g *grav.Grav) *Executor {
	// Reactr is configured in UseCapabiltyConfig
	e := &Executor{
		grav: g,
		pod:  g.Connect(),
		log:  log,
	}

	return e
}

// Do executes a local or remote job
func (e *Executor) Do(jobType string, req *request.CoordinatedRequest, ctx *vk.Ctx) (interface{}, error) {
	if e.reactr == nil {
		return nil, ErrExecutorNotConfigured
	}

	if !e.reactr.IsRegistered(jobType) {
		// TODO: handle with a remote call

		return nil, ErrCannotHandle
	}

	res := e.reactr.Do(rt.NewJob(jobType, req))

	e.pod.Send(grav.NewMsgWithParentID(fmt.Sprintf("local/%s", jobType), ctx.RequestID(), nil))

	result, err := res.Then()
	if err != nil {
		e.pod.Send(grav.NewMsgWithParentID(rt.MsgTypeReactrRunErr, ctx.RequestID(), []byte(err.Error())))
	} else {
		e.pod.Send(grav.NewMsgWithParentID(rt.MsgTypeReactrResult, ctx.RequestID(), result.([]byte)))
	}

	return result, err
}

// UseCapabilityConfig sets up the executor's Reactr instance using the provided capability configuration
func (e *Executor) UseCapabilityConfig(config rcap.CapabilityConfig) error {
	r, err := rt.NewWithConfig(config)
	if err != nil {
		return errors.Wrap(err, "failed to rt.NewWithConfig")
	}

	e.reactr = r

	return nil
}

// Register registers a Runnable
func (e *Executor) Register(jobType string, runner rt.Runnable, opts ...rt.Option) error {
	if e.reactr == nil {
		return ErrExecutorNotConfigured
	}

	e.reactr.Register(jobType, runner, opts...)

	return nil
}

// DesiredStepState calculates the state as it should be for a particular step's 'with' clause
func (e *Executor) DesiredStepState(step *executable.Executable, req *request.CoordinatedRequest) (map[string][]byte, error) {
	if len(step.With) == 0 {
		return req.State, nil
	}

	desiredState := map[string][]byte{}

	for alias, key := range step.With {
		val, exists := req.State[key]
		if !exists {
			return nil, fmt.Errorf("failed to build desired state, %s does not exists in handler state", key)
		}

		desiredState[alias] = val
	}

	return desiredState, nil
}

// ListenAndRun sets up the executor's Reactr instance to listen for messages and execute the associated job
func (e *Executor) ListenAndRun(msgType string, run func(grav.Message, interface{}, error)) error {
	if e.reactr == nil {
		return ErrExecutorNotConfigured
	}

	e.reactr.ListenAndRun(e.grav.Connect(), msgType, run)

	return nil
}

// SetSchedule adds a Schedule to the executor's Reactr instance
func (e *Executor) SetSchedule(sched rt.Schedule) error {
	if e.reactr == nil {
		return ErrExecutorNotConfigured
	}

	e.reactr.Schedule(sched)

	return nil
}

// Load loads Runnables into the executor's Reactr instance
// And connects them to the Grav instance (currently unused)
func (e *Executor) Load(runnables []directive.Runnable) error {
	if e.reactr == nil {
		return ErrExecutorNotConfigured
	}

	for _, fn := range runnables {
		if fn.FQFN == "" {
			e.log.ErrorString("fn", fn.Name, "missing calculated FQFN, will not be available")
			continue
		}

		e.log.Debug("adding listener for", fn.FQFN)
		e.reactr.Listen(e.grav.Connect(), fn.FQFN)
	}

	return load.Runnables(e.reactr, runnables, false)
}

// UseCallback does nothing in normal executor mode
func (e *Executor) UseCallback(callback grav.MsgFunc) {
	// nothing to do
}

// Metrics returns the executor's Reactr isntance's internal metrics
func (e *Executor) Metrics() (*rt.ScalerMetrics, error) {
	if e.reactr == nil {
		return nil, ErrExecutorNotConfigured
	}

	metrics := e.reactr.Metrics()

	return &metrics, nil
}
