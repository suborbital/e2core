package coordinator

import (
	"fmt"
	"net/http"

	"github.com/pkg/errors"
	"github.com/suborbital/atmo/atmo/appsource"
	"github.com/suborbital/atmo/atmo/coordinator/capabilities"
	"github.com/suborbital/atmo/atmo/coordinator/executor"
	"github.com/suborbital/atmo/atmo/options"
	"github.com/suborbital/atmo/directive"
	"github.com/suborbital/grav/grav"
	"github.com/suborbital/grav/transport/kafka"
	"github.com/suborbital/grav/transport/nats"
	"github.com/suborbital/grav/transport/websocket"
	"github.com/suborbital/reactr/rcap"
	"github.com/suborbital/reactr/rt"
	"github.com/suborbital/vektor/vk"
	"github.com/suborbital/vektor/vlog"
)

const (
	atmoMethodSchedule       = "SCHED"
	atmoMethodStream         = "STREAM"
	atmoHeadlessStateHeader  = "X-Atmo-State"
	atmoHeadlessParamsHeader = "X-Atmo-Params"
	atmoRequestIDHeader      = "X-Atmo-RequestID"
	atmoMessageURI           = "/meta/message"
	AtmoMetricsURI           = "/meta/metrics"
	AtmoHealthURI            = "/health"
)

type rtFunc func(rt.Job, *rt.Ctx) (interface{}, error)

// Coordinator is a type that is responsible for covnerting the directive into
// usable Vektor handles by coordinating Reactr jobs and meshing when needed.
type Coordinator struct {
	App  appsource.AppSource
	opts *options.Options

	log *vlog.Logger

	exec *executor.Executor

	transport *websocket.Transport

	connections map[string]*grav.Grav
	handlerPods map[string]*grav.Pod
}

type requestScope struct {
	RequestID string `json:"request_id"`
}

// New creates a coordinator
func New(appSource appsource.AppSource, options *options.Options) *Coordinator {
	var transport *websocket.Transport

	if options.ControlPlane != "" || options.Proxy {
		transport = websocket.New()
	}

	exec := executor.New(options.Logger, transport)

	c := &Coordinator{
		App:         appSource,
		opts:        options,
		log:         options.Logger,
		exec:        exec,
		connections: map[string]*grav.Grav{},
		handlerPods: map[string]*grav.Pod{},
		transport:   transport,
	}

	return c
}

// Start allows the Coordinator to bootstrap
func (c *Coordinator) Start() error {
	if c.opts.Proxy {
		c.log.Info("running in proxy mode")
	}

	if err := c.App.Start(*c.opts); err != nil {
		return errors.Wrap(err, "failed to App.Start")
	}

	// establish connections defined by the app
	c.createConnections()

	caps, err := capabilities.Render(rcap.DefaultCapabilityConfig(), c.App, c.log)
	if err != nil {
		return errors.Wrap(err, "failed to renderCapabilities")
	}

	// we have to wait until here to initialize Reactr
	// since the appsource needs to be fully initialized
	if err := c.exec.UseCapabilityConfig(caps); err != nil {
		return errors.Wrap(err, "failed to UseCapabilityConfig")
	}

	// do an initial sync of Runnables
	// from the AppSource into RVG
	c.SyncAppState()

	return nil
}

// SetupHandlers configures all of the app's handlers and generates a Vektor Router for the app
func (c *Coordinator) SetupHandlers() *vk.Router {
	router := vk.NewRouter(c.log)

	// set a middleware on the root RouteGroup
	router.Before(scopeMiddleware)

	// if in headless mode, enable runnable authentication
	if *c.opts.Headless {
		router.Before(c.headlessAuthMiddleware())
	}

	// mount each handler into the VK group
	for _, h := range c.App.Handlers() {
		switch h.Input.Type {
		case directive.InputTypeRequest:
			router.Handle(h.Input.Method, h.Input.Resource, c.vkHandlerForDirectiveHandler(h))
		case directive.InputTypeStream:
			if h.Input.Source == "" || h.Input.Source == directive.InputSourceServer {
				router.HandleHTTP(http.MethodGet, h.Input.Resource, c.websocketHandlerForDirectiveHandler(h))
			} else {
				c.streamConnectionForDirectiveHandler(h)
			}
		}
	}

	router.GET(AtmoMetricsURI, c.metricsHandler())

	router.GET(AtmoHealthURI, c.health())

	if c.transport != nil {
		router.HandleHTTP(http.MethodGet, atmoMessageURI, c.transport.HTTPHandlerFunc())
	}

	return router
}

// CreateConnections establishes all of the connections described in the directive
func (c *Coordinator) createConnections() {
	connections := c.App.Connections()

	if connections.NATS != nil {
		address := rcap.AugmentedValFromEnv(connections.NATS.ServerAddress)

		gnats, err := nats.New(address)
		if err != nil {
			c.log.Error(errors.Wrap(err, "failed to nats.New for NATS connection"))
		} else {
			g := grav.New(
				grav.UseLogger(c.log),
				grav.UseTransport(gnats),
			)

			c.connections[directive.InputSourceNATS] = g
		}
	}

	if connections.Kafka != nil {
		address := rcap.AugmentedValFromEnv(connections.Kafka.BrokerAddress)

		gkafka, err := kafka.New(address)
		if err != nil {
			c.log.Error(errors.Wrap(err, "failed to kafka.New for Kafka connection"))
		} else {
			g := grav.New(
				grav.UseLogger(c.log),
				grav.UseTransport(gkafka),
			)

			c.connections[directive.InputSourceKafka] = g
		}
	}
}

func (c *Coordinator) SetSchedules() {
	// mount each schedule into Reactr
	for _, s := range c.App.Schedules() {
		rtFunc := c.rtFuncForDirectiveSchedule(s)

		// create basically an fqfn for this schedule (com.suborbital.appname#schedule.dojob@v0.1.0)
		jobName := fmt.Sprintf("%s#schedule.%s@%s", c.App.Meta().Identifier, s.Name, c.App.Meta().AppVersion)

		c.exec.Register(jobName, &scheduledRunner{rtFunc})

		seconds := s.NumberOfSeconds()

		// only actually schedule the job if the env var isn't set (or is set but not 'false')
		// the job stays mounted on reactr because we could get a request to run it from grav
		if *c.opts.RunSchedules {
			c.log.Debug("adding schedule", jobName)

			c.exec.SetSchedule(rt.Every(seconds, func() rt.Job {
				return rt.NewJob(jobName, nil)
			}))
		}
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
	if step.As != "" {
		key = step.As
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
