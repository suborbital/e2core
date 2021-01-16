package coordinator

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/suborbital/grav/grav"
	"github.com/suborbital/hive-wasm/bundle"
	"github.com/suborbital/hive-wasm/directive"
	"github.com/suborbital/hive-wasm/request"
	"github.com/suborbital/hive-wasm/wasm"
	"github.com/suborbital/hive/hive"
	"github.com/suborbital/vektor/vk"
	"github.com/suborbital/vektor/vlog"
)

// Coordinator is a type that is responsible for covnerting the directive into
// usable Vektor handles by coordinating Hive jobs and meshing when needed.
type Coordinator struct {
	directive *directive.Directive
	bundle    *bundle.Bundle

	log *vlog.Logger

	hive *hive.Hive
	bus  *grav.Grav

	lock sync.Mutex
}

type requestScope struct {
	RequestID string `json:"request_id"`
}

// New creates a coordinator
func New(logger *vlog.Logger) *Coordinator {
	hive := hive.New()
	bus := grav.New(
		grav.UseLogger(logger),
	)

	c := &Coordinator{
		log:  logger,
		hive: hive,
		bus:  bus,
		lock: sync.Mutex{},
	}

	return c
}

// UseBundle sets a bundle to be used
func (c *Coordinator) UseBundle(bundle *bundle.Bundle) *vk.RouteGroup {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.directive = bundle.Directive

	// mount all of the Wasm modules into the Hive instance
	wasm.HandleBundle(c.hive, bundle)

	group := vk.Group("").Before(scopeMiddleware)

	// connect a Grav pod to each function
	for _, fn := range bundle.Directive.Runnables {
		fqfn, err := bundle.Directive.FQFN(fn.Name)
		if err != nil {
			c.log.Error(errors.Wrapf(err, "failed to derive FQFN for Directive function %s, function will not be available", fn.Name))
			continue
		}

		c.hive.Listen(c.bus.Connect(), fqfn)
	}

	// mount each handler into the VK group
	for _, h := range bundle.Directive.Handlers {
		if h.Input.Type != directive.InputTypeRequest {
			continue
		}

		handler := c.vkHandlerForDirectiveHandler(h)

		group.Handle(h.Input.Method, h.Input.Resource, handler)
	}

	return group
}

func (c *Coordinator) vkHandlerForDirectiveHandler(handler directive.Handler) vk.HandlerFunc {
	return func(r *http.Request, ctx *vk.Ctx) (interface{}, error) {
		req, err := request.FromVKRequest(r, ctx)
		if err != nil {
			ctx.Log.Error(errors.Wrap(err, "failed to request.FromVKRequest"))
			return nil, vk.E(http.StatusInternalServerError, "failed to handle request")
		}

		// run through the handler's steps, updating the coordinated state after each
		for _, step := range handler.Steps {
			stateJSON, err := stateJSONForStep(req, step)
			if err != nil {
				ctx.Log.Error(errors.Wrap(err, "failed to stateJSONForStep"))
				return nil, err
			}

			if step.IsFn() {
				entry, err := c.runSingleFn(step.CallableFn, stateJSON, ctx)
				if err != nil {
					return nil, err
				}

				if entry != nil {
					// hive-wasm issue #45
					key := key(step.CallableFn)

					req.State[key] = entry
				}
			} else {
				// if the step is a group, run them all concurrently and collect the results
				entries, err := c.runGroup(step.Group, stateJSON, ctx)
				if err != nil {
					return nil, err
				}

				for k, v := range entries {
					req.State[k] = v
				}
			}
		}

		return resultFromState(handler, req.State), nil
	}
}

func (c *Coordinator) runSingleFn(fn directive.CallableFn, body []byte, ctx *vk.Ctx) ([]byte, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		ctx.Log.Debug("fn", fn.Fn, fmt.Sprintf("executed in %d ms", duration.Milliseconds()))
	}()

	// calculate the FQFN
	fqfn, err := c.directive.FQFN(fn.Fn)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to FQFN for group fn %s", fn.Fn)
	}

	// compose a message containing the serialized request state, and send it via Grav
	// for the appropriate meshed Hive to handle. It may be handled by self if appropriate.
	jobMsg := grav.NewMsg(fqfn, body)

	var jobResult []byte
	var jobErr error

	pod := c.bus.Connect()
	defer pod.Disconnect()

	podErr := pod.Send(jobMsg).WaitUntil(grav.Timeout(30), func(msg grav.Message) error {
		switch msg.Type() {
		case hive.MsgTypeHiveResult:
			jobResult = msg.Data()
		case hive.MsgTypeHiveJobErr:
			jobErr = errors.New(string(msg.Data()))
		case hive.MsgTypeHiveNilResult:
			// do nothing
		}

		return nil
	})

	// check for errors and results, convert to something useful, and return
	// this should probably be refactored as it looks pretty goofy

	if podErr != nil {
		if podErr == grav.ErrWaitTimeout {
			// do nothing
		} else {
			jobErr = errors.Wrap(podErr, "message reply timed out")
		}
	}

	if jobErr != nil {
		return nil, vk.Wrap(http.StatusInternalServerError, errors.Wrapf(jobErr, "group fn %s failed", fn.Fn))
	}

	if jobResult == nil {
		ctx.Log.Debug("fn", fn.Fn, "returned a nil result")
		return nil, nil
	}

	return jobResult, nil
}

type fnResult struct {
	name   string
	result []byte
	err    error
}

// runGroup runs a group of functions
// this is all more complicated than it needs to be, Grav should be doing more of the work for us here
func (c *Coordinator) runGroup(fns []directive.CallableFn, body []byte, ctx *vk.Ctx) (map[string][]byte, error) {
	start := time.Now()
	defer func() {
		ctx.Log.Debug("group", fmt.Sprintf("executed in %d ms", time.Since(start).Milliseconds()))
	}()

	resultChan := make(chan fnResult, len(fns))

	// for now we'll use a bit of a kludgy means of running all of the group fns concurrently
	// in the future, we should send out all of the messages first, then have some new Grav
	// functionality to collect all the responses, probably using the parent ID.
	for i := range fns {
		fn := fns[i]
		ctx.Log.Debug("running fn", fn.Fn, "from group")

		key := key(fn)

		go func() {
			res, err := c.runSingleFn(fn, body, ctx)

			result := fnResult{
				name:   key,
				result: res,
				err:    err,
			}

			resultChan <- result
		}()
	}

	entries := map[string][]byte{}
	respCount := 0
	timeoutChan := time.After(5 * time.Second)

	for respCount < len(fns) {
		select {
		case resp := <-resultChan:
			if resp.err != nil {
				return nil, errors.Wrapf(resp.err, "%s produced error", resp.name)
			}

			if resp.result != nil {
				entries[resp.name] = resp.result
			}
		case <-timeoutChan:
			return nil, errors.New("function group timed out")
		}

		respCount++
	}

	return entries, nil
}

func scopeMiddleware(r *http.Request, ctx *vk.Ctx) error {
	scope := requestScope{
		RequestID: ctx.RequestID(),
	}

	ctx.UseScope(scope)

	return nil
}

func stateJSONForStep(req *request.CoordinatedRequest, step directive.Executable) ([]byte, error) {
	// the desired state is cached, so after the first call this is very efficient
	desired, err := step.ParseWith()
	if err != nil {
		return nil, vk.Wrap(http.StatusInternalServerError, errors.Wrap(err, "failed to ParseWith"))
	}

	// based on the step's `with` clause, build the state to pass into the function
	stepState, err := desiredState(desired, req.State)
	if err != nil {
		return nil, vk.Wrap(http.StatusInternalServerError, errors.Wrap(err, "failed to build desiredState"))
	}

	stepReq := request.CoordinatedRequest{
		Method:  req.Method,
		URL:     req.URL,
		ID:      req.ID,
		Body:    req.Body,
		Headers: req.Headers,
		Params:  req.Params,
		State:   stepState,
	}

	stateJSON, err := stepReq.ToJSON()
	if err != nil {
		return nil, vk.Wrap(http.StatusInternalServerError, errors.Wrap(err, "failed to ToJSON Request State"))
	}

	return stateJSON, nil
}

// resultFromState returns the state value for the last single function that ran in a handler
func resultFromState(handler directive.Handler, state map[string][]byte) []byte {
	// if the handler defines a response explicitly, use it (return nil if there is nothing in state)
	if handler.Response != "" {
		resp, exists := state[handler.Response]
		if exists {
			return resp
		}

		return nil
	}

	// if not, use the last step. If last step is a group, return nil
	step := handler.Steps[len(handler.Steps)-1]
	if step.Fn == "" {
		return nil
	}

	val, exists := state[step.Fn]
	if exists {
		return val
	}

	return nil
}

func desiredState(desired []directive.Alias, state map[string][]byte) (map[string][]byte, error) {
	if desired == nil || len(desired) == 0 {
		return state, nil
	}

	desiredState := map[string][]byte{}

	for _, a := range desired {
		val, exists := state[a.Key]
		if !exists {
			return nil, fmt.Errorf("failed to build desired state, %s does not exists in handler state", a.Key)
		}

		desiredState[a.Alias] = val
	}

	return desiredState, nil
}

func key(fn directive.CallableFn) string {
	key := fn.Fn

	if fn.As != "" {
		key = fn.As
	}

	return key
}
