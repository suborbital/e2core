//go:build !proxy

package executor

import (
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"

	"github.com/suborbital/e2core/foundation/bus/bus"
	"github.com/suborbital/e2core/foundation/scheduler"
	"github.com/suborbital/e2core/sat/engine"
	"github.com/suborbital/e2core/sat/engine/runtime/api"
	"github.com/suborbital/systemspec/capabilities"
	"github.com/suborbital/systemspec/request"
	"github.com/suborbital/systemspec/tenant"
	"github.com/suborbital/systemspec/tenant/executable"
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
	engine   *engine.Engine
	bus      *bus.Bus
	capCache map[string]*capabilities.Capabilities

	pod *bus.Pod

	log *vlog.Logger
}

// New creates an Executor
func New(log *vlog.Logger, config capabilities.CapabilityConfig) (*Executor, error) {
	api, err := api.NewWithConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, "failed to NewWithConfig")
	}

	e := &Executor{
		log:      log,
		engine:   engine.NewWithAPI(api),
		capCache: make(map[string]*capabilities.Capabilities),
	}

	return e, nil
}

// Do executes a local or remote job.
func (e *Executor) Do(jobType string, req *request.CoordinatedRequest, ctx *vk.Ctx, cb bus.MsgFunc) (interface{}, error) {
	if e.engine == nil {
		return nil, ErrExecutorNotConfigured
	}

	if !e.engine.IsRegistered(jobType) {
		// TODO: handle with a remote call.

		return nil, ErrCannotHandle
	}

	res := e.engine.Do(scheduler.NewJob(jobType, req))

	e.Send(bus.NewMsgWithParentID(fmt.Sprintf("local/%s", jobType), ctx.RequestID(), nil))

	result, err := res.Then()
	if err != nil {
		e.Send(bus.NewMsgWithParentID(scheduler.MsgTypeReactrRunErr, ctx.RequestID(), []byte(err.Error())))
	} else {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			e.log.Error(errors.Wrap(err, "failed to Marshal executor result"))
		}

		e.Send(bus.NewMsgWithParentID(scheduler.MsgTypeReactrResult, ctx.RequestID(), resultJSON))
	}

	return result, err
}

// UseBus sets a Bus instance to use (in case one was not provided initially)
func (e *Executor) UseBus(b *bus.Bus) {
	e.bus = b
	e.pod = b.Connect()
}

// Register registers a Runnable.
func (e *Executor) Register(jobType string, ref *tenant.WasmModuleRef, opts ...scheduler.Option) error {
	if e.engine == nil {
		return ErrExecutorNotConfigured
	}

	e.engine.Register(jobType, ref, opts...)

	return nil
}

// DesiredStepState calculates the state as it should be for a particular step's 'with' clause.
func (e *Executor) DesiredStepState(step executable.Executable, req *request.CoordinatedRequest) (map[string][]byte, error) {
	// this is no longer needed in the Executor, will be removed from e2core in the future
	return nil, ErrDesiredStateNotGenerated
}

// ListenAndRun sets up the executor's Reactr instance to listen for messages and execute the associated job.
func (e *Executor) ListenAndRun(msgType string, run func(bus.Message, interface{}, error)) error {
	if e.engine == nil {
		return ErrExecutorNotConfigured
	}

	e.engine.ListenAndRun(e.bus.Connect(), msgType, run)

	return nil
}

// Send sends a message on the configured Pod.
func (e *Executor) Send(msg bus.Message) *bus.MsgReceipt {
	if e.pod == nil {
		return nil
	}

	return e.pod.Send(msg)
}

// SetSchedule adds a Schedule to the executor's Reactr instance.
func (e *Executor) SetSchedule(sched scheduler.Schedule) error {
	if e.engine == nil {
		return ErrExecutorNotConfigured
	}

	e.engine.Schedule(sched)

	return nil
}

// Metrics returns the executor's Reactr instance's internal metrics.
func (e *Executor) Metrics() (*scheduler.ScalerMetrics, error) {
	if e.engine == nil {
		return nil, ErrExecutorNotConfigured
	}

	metrics := e.engine.Metrics()

	return &metrics, nil
}
