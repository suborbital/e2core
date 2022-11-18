package executor

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/suborbital/e2core/foundation/bus/bus"
	"github.com/suborbital/e2core/foundation/bus/discovery/local"
	"github.com/suborbital/e2core/foundation/bus/transport/websocket"
	"github.com/suborbital/e2core/foundation/scheduler"
	"github.com/suborbital/systemspec/capabilities"
	"github.com/suborbital/systemspec/request"
	"github.com/suborbital/systemspec/system"
	"github.com/suborbital/systemspec/tenant/executable"
	"github.com/suborbital/vektor/vk"
	"github.com/suborbital/vektor/vlog"
)

const (
	MsgTypeSuborbitalResult = "suborbital.result"
)

var (
	ErrDesiredStateNotGenerated = errors.New("desired state was not generated")
	ErrExecutorNotConfigured    = errors.New("executor not fully configured")
	ErrExecutorTimeout          = errors.New("execution did not complete before the timeout")
	ErrCannotHandle             = errors.New("cannot handle job")
)

type Executor interface {
	Do(jobType string, req *request.CoordinatedRequest, ctx *vk.Ctx, cb bus.MsgFunc) (interface{}, error)
	DesiredStepState(step executable.Executable, req *request.CoordinatedRequest) (map[string][]byte, error)
	SetSchedule(sched scheduler.Schedule) error
	Metrics() (*scheduler.ScalerMetrics, error)
}

// meshExecutor is a facade over Grav that allows executing remote
// functions with a single call, ensuring there is no difference between them to the caller
type meshExecutor struct {
	bus       *bus.Bus
	pod       *bus.Pod
	log       *vlog.Logger
	callbacks map[string]bus.MsgFunc
	cbLock    sync.RWMutex
}

// New creates a new Executor
func New(log *vlog.Logger, transport *websocket.Transport) *meshExecutor {
	gravOpts := []bus.OptionsModifier{
		bus.UseLogger(log),
	}

	if transport != nil {
		d := local.New()

		gravOpts = append(gravOpts, bus.UseMeshTransport(transport))
		gravOpts = append(gravOpts, bus.UseDiscovery(d))
	}

	b := bus.New(gravOpts...)

	e := &meshExecutor{
		bus:       b,
		pod:       b.Connect(),
		log:       log,
		callbacks: make(map[string]bus.MsgFunc),
		cbLock:    sync.RWMutex{},
	}

	// funnel all result messages to their respective sequence callbacks
	e.pod.OnType(MsgTypeSuborbitalResult, func(msg bus.Message) error {
		e.cbLock.RLock()
		defer e.cbLock.RUnlock()

		cb, exists := e.callbacks[msg.ParentID()]
		if !exists {
			log.ErrorString("encountered nil callback:", msg.Type(), msg.UUID(), msg.ParentID())
			return nil
		}

		cb(msg)

		return nil
	})

	return e
}

// Do executes a remote job
func (e *meshExecutor) Do(jobType string, req *request.CoordinatedRequest, ctx *vk.Ctx, cb bus.MsgFunc) (interface{}, error) {
	var runErr error
	var cbErr error

	pod := e.bus.Connect()
	defer pod.Disconnect()

	ctx.Log.Debug("proxying execution for", jobType)

	completed := make(chan bool)

	// start listening to the messages produced by peers
	// on the network, and don't stop until there's an error
	// or the Sequence we're connected to deems that it's complete
	e.addCallback(ctx.RequestID(), func(msg bus.Message) error {
		// the Sequence callback will return an error under two main conditions:
		// - The sequence has ended: yay!
		// - The result we just got caused an error: boo :(
		// either way, show's over and we send on `completed`

		if err := cb(msg); err != nil {
			if err == executable.ErrSequenceCompleted {
				// do nothing! that's great!
			} else if cbRunErr, isRunErr := err.(scheduler.RunErr); isRunErr {
				// handle the runErr
				runErr = cbRunErr
			} else {
				// nothing we really can do here, but let's propogate it
				cbErr = err
			}

			completed <- true
		}

		return nil
	})

	defer e.removeCallback(ctx.RequestID())

	// defer func() {
	// 	if e := recover(); e != nil {
	// 		fmt.Println("RECOVERED:", e)
	// 	}
	// }()

	data, err := json.Marshal(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to req.toJSON")
	}

	msg := bus.NewMsgWithParentID(jobType, ctx.RequestID(), data)

	// find an appropriate peer and tunnel the first excution to them
	if err := e.bus.Tunnel(jobType, msg); err != nil {
		return nil, errors.Wrap(err, "failed to Tunnel, will retry")
	}

	ctx.Log.Debug("proxied execution for", ctx.RequestID(), "to peer with message", msg.UUID())

	// wait until the sequence completes or errors
	select {
	case <-completed:
		// awesome, do nothing
	case <-time.After(time.Second * 10):
		return nil, ErrExecutorTimeout
	}

	if cbErr != nil {
		return nil, cbErr
	} else if runErr != nil {
		return nil, runErr
	}

	// getting the JobResult was done by the callback, return nothing
	return nil, nil
}

func (e *meshExecutor) addCallback(parentID string, cb bus.MsgFunc) {
	e.cbLock.Lock()
	defer e.cbLock.Unlock()

	e.callbacks[parentID] = cb
}

func (e *meshExecutor) removeCallback(parentID string) {
	e.cbLock.Lock()
	defer e.cbLock.Unlock()

	delete(e.callbacks, parentID)
}

// UseCapabilityConfig sets up the executor's Reactr instance using the provided capability configuration
func (e *meshExecutor) UseCapabilityConfig(config capabilities.CapabilityConfig) error {
	// nothing to do in proxy mode

	return nil
}

// DesiredStepState generates the desired state for the step from the 'real' state
func (e *meshExecutor) DesiredStepState(step executable.Executable, req *request.CoordinatedRequest) (map[string][]byte, error) {
	// in proxy mode, we don't want to handle desired state ourselves, we want each peer to handle it themselves
	return nil, ErrDesiredStateNotGenerated
}

// this does nothing in proxy mode
func (e *meshExecutor) ListenAndRun(msgType string, run func(bus.Message, interface{}, error)) error {
	return nil
}

// SetSchedule adds a Schedule to the executor's Reactr instance
func (e *meshExecutor) SetSchedule(sched scheduler.Schedule) error {
	// nothing to do in proxy mode

	return nil
}

// Load loads Runnables into the executor's Reactr instance
// And connects them to the Grav instance (currently unused)
func (e *meshExecutor) Load(source system.Source) error {
	// nothing to do in proxy mode

	return nil
}

// Metrics returns the executor's Reactr isntance's internal metrics
func (e *meshExecutor) Metrics() (*scheduler.ScalerMetrics, error) {
	// nothing to do in proxy mode

	return &scheduler.ScalerMetrics{}, nil
}
