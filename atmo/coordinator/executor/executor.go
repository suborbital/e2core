//go:build !proxy

package executor

import (
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"

	"github.com/suborbital/atmo/atmo/appsource"
	"github.com/suborbital/atmo/atmo/coordinator/capabilities"
	"github.com/suborbital/atmo/directive/executable"
	"github.com/suborbital/grav/discovery/local"
	"github.com/suborbital/grav/grav"
	"github.com/suborbital/grav/transport/websocket"
	"github.com/suborbital/reactr/request"
	"github.com/suborbital/reactr/rt"
	"github.com/suborbital/reactr/rwasm"
	"github.com/suborbital/vektor/vk"
	"github.com/suborbital/vektor/vlog"
)

var (
	ErrExecutorNotConfigured    = errors.New("executor not fully configured")
	ErrDesiredStateNotGenerated = errors.New("desired state was not generated")
	ErrCannotHandle             = errors.New("cannot handle job")
)

// Executor is a facade over Grav and Reactr that allows executing local OR remote
// functions with a single call, ensuring there is no difference between them to the caller.
type Executor struct {
	reactr   *rt.Reactr
	grav     *grav.Grav
	capCache map[string]*rt.Capabilities

	pod *grav.Pod

	log *vlog.Logger
}

// New creates a new Executor with the default Grav configuration.
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

// NewWithGrav creates an Executor with a custom Grav instance.
func NewWithGrav(log *vlog.Logger, g *grav.Grav) *Executor {
	var pod *grav.Pod

	if g != nil {
		pod = g.Connect()
	}

	e := &Executor{
		grav:     g,
		pod:      pod,
		log:      log,
		reactr:   rt.New(),
		capCache: make(map[string]*rt.Capabilities),
	}

	return e
}

// Do executes a local or remote job.
func (e *Executor) Do(jobType string, req *request.CoordinatedRequest, ctx *vk.Ctx, cb grav.MsgFunc) (interface{}, error) {
	if e.reactr == nil {
		return nil, ErrExecutorNotConfigured
	}

	if !e.reactr.IsRegistered(jobType) {
		// TODO: handle with a remote call.

		return nil, ErrCannotHandle
	}

	res := e.reactr.Do(rt.NewJob(jobType, req))

	e.Send(grav.NewMsgWithParentID(fmt.Sprintf("local/%s", jobType), ctx.RequestID(), nil))

	result, err := res.Then()
	if err != nil {
		e.Send(grav.NewMsgWithParentID(rt.MsgTypeReactrRunErr, ctx.RequestID(), []byte(err.Error())))
	} else {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			e.log.Error(errors.Wrap(err, "failed to Marshal executor result"))
		}

		e.Send(grav.NewMsgWithParentID(rt.MsgTypeReactrResult, ctx.RequestID(), resultJSON))
	}

	return result, err
}

// UseGrav sets a Grav instance to use (in case one was not provided initially)
// this will NOT set the internal pod, and so internal messages will not be sent.
func (e *Executor) UseGrav(g *grav.Grav) {
	e.grav = g
}

// Register registers a Runnable.
func (e *Executor) Register(jobType string, runner rt.Runnable, opts ...rt.Option) error {
	if e.reactr == nil {
		return ErrExecutorNotConfigured
	}

	e.reactr.Register(jobType, runner, opts...)

	return nil
}

// DesiredStepState calculates the state as it should be for a particular step's 'with' clause.
func (e *Executor) DesiredStepState(step executable.Executable, req *request.CoordinatedRequest) (map[string][]byte, error) {
	if len(step.With) == 0 {
		return nil, ErrDesiredStateNotGenerated
	}

	desiredState := map[string][]byte{}
	aliased := map[string]bool{}

	// first go through the 'with' clause and load all of the appropriate aliased values.
	for alias, key := range step.With {
		val, exists := req.State[key]
		if !exists {
			return nil, fmt.Errorf("failed to build desired state, %s does not exists in handler state", key)
		}

		desiredState[alias] = val
		aliased[key] = true
	}

	// next, go through the rest of the original state and load the non-aliased values.
	for key, val := range req.State {
		_, skip := aliased[key]
		if !skip {
			desiredState[key] = val
		}
	}

	return desiredState, nil
}

// ListenAndRun sets up the executor's Reactr instance to listen for messages and execute the associated job.
func (e *Executor) ListenAndRun(msgType string, run func(grav.Message, interface{}, error)) error {
	if e.reactr == nil {
		return ErrExecutorNotConfigured
	}

	e.reactr.ListenAndRun(e.grav.Connect(), msgType, run)

	return nil
}

// Send sends a message on the configured Pod.
func (e *Executor) Send(msg grav.Message) {
	if e.pod == nil {
		return
	}

	e.pod.Send(msg)
}

// SetSchedule adds a Schedule to the executor's Reactr instance.
func (e *Executor) SetSchedule(sched rt.Schedule) error {
	if e.reactr == nil {
		return ErrExecutorNotConfigured
	}

	e.reactr.Schedule(sched)

	return nil
}

// @todo pass in the whole appsource
//
// Load loads Runnables into the executor's Reactr instance
// And connects them to the Grav instance (currently unused).
//func (e *Executor) Load(runnables []directive.Runnable) error {

func (e *Executor) Load(source appsource.AppSource) error {
	if e.reactr == nil {
		return ErrExecutorNotConfigured
	}

	for _, app := range source.Applications() {
		for _, fn := range source.Runnables(app.Identifier, app.AppVersion) {
			if fn.FQFN == "" {
				e.log.ErrorString("fn", fn.Name, "missing calculated FQFN, will not be available")
				continue
			}

			capObject, err := e.resolveCap(app.Identifier, fn.Namespace, app.AppVersion, source, e.log)
			if err != nil {
				e.log.ErrorString("e.resolveCap", err.Error(), app.Identifier, fn.Namespace, app.AppVersion)
				continue
			}

			e.reactr.RegisterWithCaps(fn.FQFN, rwasm.NewRunnerWithRef(fn.ModuleRef), *capObject)

			e.log.Debug("adding listener for", fn.FQFN)
			e.reactr.Listen(e.grav.Connect(), fn.FQFN)
		}
	}

	return nil
}

// resolveCap stores the cap if it doesn't exist yet, or returns it if it does.
func (e *Executor) resolveCap(ident, namespace, version string, source appsource.AppSource, log *vlog.Logger) (*rt.Capabilities, error) {
	cacheKey := fmt.Sprintf("%s/%s/%s", ident, namespace, version)

	foundCap, ok := e.capCache[cacheKey]
	if ok {
		return foundCap, nil
	}

	renderedCap, err := capabilities.ResolveFromSource(source, ident, namespace, version, e.log)
	if err != nil {
		return nil, errors.Wrap(err, "capabilities.ResolveFromSource")
	}

	capObject, err := rt.CapabilitiesFromConfig(renderedCap)
	if err != nil {
		return nil, errors.Wrap(err, "rt.CapabilitiesFromConfig")
	}

	e.capCache[cacheKey] = capObject

	return capObject, nil
}

// Metrics returns the executor's Reactr isntance's internal metrics.
func (e *Executor) Metrics() (*rt.ScalerMetrics, error) {
	if e.reactr == nil {
		return nil, ErrExecutorNotConfigured
	}

	metrics := e.reactr.Metrics()

	return &metrics, nil
}
