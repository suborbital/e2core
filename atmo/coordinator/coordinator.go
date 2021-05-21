package coordinator

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/suborbital/atmo/atmo/appsource"
	"github.com/suborbital/atmo/atmo/options"
	"github.com/suborbital/atmo/bundle/load"
	"github.com/suborbital/atmo/directive"
	"github.com/suborbital/grav/grav"
	"github.com/suborbital/reactr/request"
	"github.com/suborbital/reactr/rt"
	"github.com/suborbital/vektor/vk"
	"github.com/suborbital/vektor/vlog"
)

const (
	atmoMethodSchedule = "SCHED"
)

type rtFunc func(rt.Job, *rt.Ctx) (interface{}, error)

// Coordinator is a type that is responsible for covnerting the directive into
// usable Vektor handles by coordinating Reactr jobs and meshing when needed.
type Coordinator struct {
	App  appsource.AppSource
	opts *options.Options

	log *vlog.Logger

	reactr *rt.Reactr
	grav   *grav.Grav

	lock sync.Mutex
}

type requestScope struct {
	RequestID string `json:"request_id"`
}

// New creates a coordinator
func New(appSource appsource.AppSource, options *options.Options) *Coordinator {
	reactr := rt.New()
	grav := grav.New(
		grav.UseLogger(options.Logger),
	)

	c := &Coordinator{
		App:    appSource,
		opts:   options,
		log:    options.Logger,
		reactr: reactr,
		grav:   grav,
		lock:   sync.Mutex{},
	}

	return c
}

// GenerateRouter generates a Vektor routeGroup for the app
func (c *Coordinator) GenerateRouter() *vk.RouteGroup {
	c.lock.Lock()
	defer c.lock.Unlock()

	// mount all of the Wasm modules into the Reactr instance
	load.ModuleRefsIntoInstance(c.reactr, c.App.Refs(), c.App.File)

	group := vk.Group("").Before(scopeMiddleware)

	// connect a Grav pod to each function
	for _, fn := range c.App.Runnables() {
		if fn.FQFN == "" {
			c.log.ErrorString("fn", fn.Name, "missing calculated FQFN, will not be available")
			continue
		}

		c.reactr.Listen(c.grav.Connect(), fn.FQFN)
	}

	// mount each handler into the VK group
	for _, h := range c.App.Handlers() {
		if h.Input.Type != directive.InputTypeRequest {
			continue
		}

		handler := c.vkHandlerForDirectiveHandler(h)

		group.Handle(h.Input.Method, h.Input.Resource, handler)
	}

	// mount each schedule into Reactr
	for _, s := range c.App.Schedules() {
		rtFunc := c.rtFuncForDirectiveSchedule(s)

		// create basically an fqfn for this schedule (com.suborbital.appname#schedule.dojob@v0.1.0)
		jobName := fmt.Sprintf("%s#schedule.%s@%s", c.App.Meta().Identifier, s.Name, c.App.Meta().AppVersion)
		c.log.Debug("adding schedule", jobName)

		c.reactr.Register(jobName, &scheduledRunner{rtFunc})

		seconds := s.NumberOfSeconds()

		// only actually schedule the job if the env var isn't set (or is set but not 'false')
		// the job stays mounted on reactr because we could get a request to run it from grav
		if c.opts.RunSchedules == "true" {
			c.reactr.Schedule(rt.Every(seconds, func() rt.Job {
				return rt.NewJob(jobName, nil)
			}))
		}
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

		// a sequence executes the handler's steps and manages its state
		seq := newSequence(handler.Steps, c.grav.Connect, ctx.Log)

		seqState, err := seq.exec(req)
		if err != nil {
			if errors.Is(err, ErrSequenceRunErr) && seqState.err != nil {
				if seqState.err.Code < 200 || seqState.err.Code > 599 {
					// if the Runnable returned an invalid code for HTTP, default to 500
					return nil, vk.Err(http.StatusInternalServerError, seqState.err.Message)
				}

				return nil, vk.Err(seqState.err.Code, seqState.err.Message)
			}

			return nil, vk.Wrap(http.StatusInternalServerError, err)
		}

		// handle any response headers that were set by the Runnables
		if req.RespHeaders != nil {
			for head, val := range req.RespHeaders {
				ctx.RespHeaders.Set(head, val)
			}
		}

		return resultFromState(handler, seqState.state), nil
	}
}

// scheduledRunner is a runner that will run a schedule on a.... schedule
type scheduledRunner struct {
	RunFunc rtFunc
}

func (s *scheduledRunner) Run(job rt.Job, ctx *rt.Ctx) (interface{}, error) {
	return s.RunFunc(job, ctx)
}

func (s *scheduledRunner) OnChange(_ rt.ChangeEvent) error { return nil }

func (c *Coordinator) rtFuncForDirectiveSchedule(sched directive.Schedule) rtFunc {
	return func(job rt.Job, ctx *rt.Ctx) (interface{}, error) {
		c.log.Info("executing schedule", sched.Name)

		// read the "initial" state from the Directive
		state := map[string][]byte{}
		for k, v := range sched.State {
			state[k] = []byte(v)
		}

		req := &request.CoordinatedRequest{
			Method:  atmoMethodSchedule,
			URL:     sched.Name,
			ID:      uuid.New().String(),
			Body:    []byte{},
			Headers: map[string]string{},
			Params:  map[string]string{},
			State:   state,
		}

		// a sequence executes the handler's steps and manages its state
		seq := newSequence(sched.Steps, c.grav.Connect, c.log)

		if seqState, err := seq.exec(req); err != nil {
			if errors.Is(err, ErrSequenceRunErr) && seqState.err != nil {
				c.log.Error(errors.Wrapf(seqState.err, "schedule %s returned an error", sched.Name))
			} else {
				c.log.Error(errors.Wrapf(err, "schedule %s failed", sched.Name))
			}
		}

		return nil, nil
	}
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
	if step.IsGroup() {
		return nil
	}

	// determine what the state key is
	key := step.Fn
	if step.IsForEach() {
		key = step.ForEach.As
	}

	val, exists := state[key]
	if exists {
		return val
	}

	return nil
}

func scopeMiddleware(r *http.Request, ctx *vk.Ctx) error {
	scope := requestScope{
		RequestID: ctx.RequestID(),
	}

	ctx.UseScope(scope)

	return nil
}
