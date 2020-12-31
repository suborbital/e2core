package coordinator

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/suborbital/grav/grav"
	"github.com/suborbital/hive-wasm/bundle"
	"github.com/suborbital/hive-wasm/directive"
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

// CoordinatedRequest represents a request being coordinated
type CoordinatedRequest struct {
	URL   string                 `json:"url"`
	Body  string                 `json:"body"`
	State map[string]interface{} `json:"state"`
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
		reqBody, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return nil, vk.E(http.StatusInternalServerError, "failed to read request body")
		}

		defer r.Body.Close()

		req := CoordinatedRequest{
			URL:   r.URL.String(),
			Body:  string(reqBody),
			State: map[string]interface{}{},
		}

		for _, step := range handler.Steps {
			stateJSON, err := req.Marshal()
			if err != nil {
				return nil, vk.Wrap(http.StatusInternalServerError, errors.Wrap(err, "failed to Marshal Request State"))
			}

			if step.IsFn() {
				entry, err := c.runSingleFn(step.CallableFn, stateJSON, ctx)
				if err != nil {
					return nil, err
				}

				if entry != nil {
					key := step.Fn
					if step.As != "" {
						key = step.As
					}

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

func (c *Coordinator) runSingleFn(fn directive.CallableFn, body []byte, ctx *vk.Ctx) (interface{}, error) {
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

	podErr := pod.Send(jobMsg).WaitUntil(grav.Timeout(3), func(msg grav.Message) error {
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

	return stringOrMap(jobResult), nil
}

type fnResult struct {
	name   string
	result interface{}
	err    error
}

// runGroup runs a group of functions
// this is all more complicated than it needs to be, Grav should be doing more of the work for us here
func (c *Coordinator) runGroup(fns []directive.CallableFn, body []byte, ctx *vk.Ctx) (map[string]interface{}, error) {
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

		key := fn.Fn
		if fn.As != "" {
			key = fn.As
		}

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

	entry := map[string]interface{}{}
	respCount := 0
	timeoutChan := time.After(5 * time.Second)

	for respCount < len(fns) {
		select {
		case resp := <-resultChan:
			if resp.err != nil {
				return nil, errors.Wrapf(resp.err, "%s produced error", resp.name)
			}

			if resp.result != nil {
				entry[resp.name] = resp.result
			}
		case <-timeoutChan:
			return nil, errors.New("function group timed out")
		}

		respCount++
	}

	return entry, nil
}

// Marshal marshals a CoordinatedRequest
func (c *CoordinatedRequest) Marshal() ([]byte, error) {
	return json.Marshal(c)
}

func scopeMiddleware(r *http.Request, ctx *vk.Ctx) error {
	scope := requestScope{
		RequestID: ctx.RequestID(),
	}

	ctx.UseScope(scope)

	return nil
}

// resultFromState returns the state value for the last single function that ran in a handler
func resultFromState(handler directive.Handler, state map[string]interface{}) interface{} {
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

// stringOrMap converts bytes to a map if they are JSON, or a string if not
func stringOrMap(result []byte) interface{} {
	resMap := map[string]interface{}{}
	if err := json.Unmarshal(result, &resMap); err != nil {
		return string(result)
	}

	return resMap
}
