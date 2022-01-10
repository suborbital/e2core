//go:build proxy

package executor

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"sync"
	"time"

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

const (
	MsgTypeAtmoFnResult = "atmo.fnresult"
)

var (
	ErrDesiredStateNotGenerated = errors.New("desired state was not generated")
	ErrExecutorNotConfigured    = errors.New("executor not fully configured")
	ErrExecutorTimeout          = errors.New("execution did not complete before the timeout")
	ErrCannotHandle             = errors.New("cannot handle job")
)

// Executor is a facade over Grav and Reactr that allows executing local OR remote
// functions with a single call, ensuring there is no difference between them to the caller
type Executor struct {
	grav      *grav.Grav
	pod       *grav.Pod
	log       *vlog.Logger
	callbacks map[string]grav.MsgFunc
	cbLock    sync.RWMutex
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

	e := &Executor{
		grav:      g,
		pod:       g.Connect(),
		log:       log,
		callbacks: map[string]grav.MsgFunc{},
		cbLock:    sync.RWMutex{},
	}

	// funnel all result messages to their respective sequence callbacks
	e.pod.OnType(MsgTypeAtmoFnResult, func(msg grav.Message) error {
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
func (e *Executor) Do(jobType string, req *request.CoordinatedRequest, ctx *vk.Ctx, cb grav.MsgFunc) (interface{}, error) {
	var runErr error
	var cbErr error

	pod := e.grav.Connect()
	defer pod.Disconnect()

	ctx.Log.Debug("proxying execution for", jobType)

	completed := make(chan bool)

	// start listening to the messages produced by peers
	// on the network, and don't stop until there's an error
	// or the Sequence we're connected to deems that it's complete
	defer e.removeCallback(ctx.RequestID())

	e.addCallback(ctx.RequestID(), func(msg grav.Message) error {
		// the Sequence callback will return an error under two main conditions:
		// - The sequence has ended: yay!
		// - The result we just got caused an error: boo :(
		// either way, show's over and we send on `completed`

		if err := cb(msg); err != nil {
			if err == executable.ErrSequenceCompleted {
				// do nothing! that's great!
			} else if cbRunErr, isRunErr := err.(rt.RunErr); isRunErr {
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

	defer func() {
		if e := recover(); e != nil {
			fmt.Println("RECOVERED:", e)
		}
	}()

	data, err := json.Marshal(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to req.toJSON")
	}

	msg := grav.NewMsgWithParentID(jobType, ctx.RequestID(), data)

	// find an appropriate peer and tunnel the first excution to them
	if err := e.grav.Tunnel(jobType, msg); err != nil {
		return nil, errors.Wrap(err, "failed to Tunnel, will retry")
	}

	ctx.Log.Info("proxied execution for", ctx.RequestID(), "to peer")

	// wait until the sequence completes or errors
	select {
	case <-completed:
		// awesome, do nothing
	case <-time.After(time.Second * 10):
		return nil, ErrExecutorTimeout
	}

	if cbErr != nil {
		return nil, cbErr
	}

	if runErr != nil {
		return nil, runErr
	}

	// getting the JobResult was done by the callback, return nothing
	return nil, nil
}

func (e *Executor) addCallback(parentID string, cb grav.MsgFunc) {
	e.cbLock.Lock()
	defer e.cbLock.Unlock()

	e.callbacks[parentID] = cb
}

func (e *Executor) removeCallback(parentID string) {
	e.cbLock.Lock()
	defer e.cbLock.Unlock()

	delete(e.callbacks, parentID)
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

// DesiredStepState generates the desired state for the step from the 'real' state
func (e *Executor) DesiredStepState(step executable.Executable, req *request.CoordinatedRequest) (map[string][]byte, error) {
	// in proxy mode, we don't want to handle desired state ourselves, we want each peer to handle it themselves
	return nil, ErrDesiredStateNotGenerated
}

// this does nothing in proxy mode
func (e *Executor) ListenAndRun(msgType string, run func(grav.Message, interface{}, error)) error {
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

	return &rt.ScalerMetrics{}, nil
}
