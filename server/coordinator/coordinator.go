package coordinator

import (
	"fmt"
	"net/http"

	"github.com/pkg/errors"

	"github.com/suborbital/appspec/capabilities"
	"github.com/suborbital/appspec/tenant"
	"github.com/suborbital/appspec/tenant/executable"
	"github.com/suborbital/e2core/bus/bus"
	"github.com/suborbital/e2core/bus/transport/kafka"
	"github.com/suborbital/e2core/bus/transport/nats"
	"github.com/suborbital/e2core/bus/transport/websocket"
	"github.com/suborbital/e2core/options"
	"github.com/suborbital/e2core/server/coordinator/executor"
	"github.com/suborbital/e2core/syncer"
	"github.com/suborbital/vektor/vk"
	"github.com/suborbital/vektor/vlog"
)

const (
	e2coreMethodSchedule = "SCHED"
	e2coreMessageURI     = "/meta/message"
	E2CoreMetricsURI     = "/meta/metrics"
	E2CoreHealthURI      = "/health"
	connectionKeyFormat  = "%s.%d.%s.%s.%s" // ident.version.namespace.connType.connName
)

// Coordinator is a type that is responsible for converting the directive into
// usable Vektor handles by coordinating Reactr jobs and meshing when needed.
type Coordinator struct {
	syncer *syncer.Syncer
	opts   *options.Options

	log *vlog.Logger

	exec executor.Executor

	transport *websocket.Transport

	connections map[string]*bus.Bus
	handlerPods map[string]*bus.Pod
}

type requestScope struct {
	RequestID string `json:"request_id"`
}

// New creates a coordinator.
func New(syncer *syncer.Syncer, options *options.Options) *Coordinator {
	transport := websocket.New()

	exec := executor.New(options.Logger(), transport)

	c := &Coordinator{
		syncer:      syncer,
		opts:        options,
		log:         options.Logger(),
		exec:        exec,
		connections: map[string]*bus.Bus{},
		handlerPods: map[string]*bus.Pod{},
		transport:   transport,
	}

	return c
}

// Start allows the Coordinator to bootstrap.
func (c *Coordinator) Start() error {
	if err := c.syncer.Start(); err != nil {
		return errors.Wrap(err, "failed to syncer.Start")
	}

	// establish connections defined by the app.
	c.createConnections()

	return nil
}

// SetupHandlers configures all of the app's handlers and generates a Vektor Router for the app.
func (c *Coordinator) SetupHandlers() (*vk.Router, error) {
	router := vk.NewRouter(c.log, "")

	// start by adding the otel handler to the stack.
	router.Before(c.openTelemetryHandler())

	// set a middleware on the root RouteGroup.
	router.Before(scopeMiddleware)

	router.POST("/name/:ident/:namespace/:name", c.vkHandlerForModuleByName())
	router.POST("/ref/:ref", c.vkHandlerForModuleByRef())

	// TODO: implement triggers
	// switch h.Input.Type {
	// case tenant.InputTypeRequest:
	// 	router.Handle(h.Input.Method, h.Input.Resource, c.vkHandlerForDirectiveHandler(h))
	// case tenant.InputTypeStream:
	// 	if h.Input.Source == "" || h.Input.Source == tenant.InputSourceServer {
	// 		router.HandleHTTP(http.MethodGet, h.Input.Resource, c.websocketHandlerForDirectiveHandler(h))
	// 	} else {
	// 		c.streamConnectionForDirectiveHandler(h, application)
	// 	}
	// }

	router.GET(E2CoreMetricsURI, c.metricsHandler())

	router.GET(E2CoreHealthURI, c.health())

	if c.transport != nil {
		router.HandleHTTP(http.MethodGet, e2coreMessageURI, c.transport.HTTPHandlerFunc())
	}

	return router, nil
}

// CreateConnections establishes all of the connections described in the tenant.
func (c *Coordinator) createConnections() {
	tenants := c.syncer.ListTenants()

	// mount each handler into the VK group.
	for ident := range tenants {
		tnt := c.syncer.TenantOverview(ident)
		if tnt == nil {
			continue
		}

		namespaces := []tenant.NamespaceConfig{tnt.Config.DefaultNamespace}
		namespaces = append(namespaces, tnt.Config.Namespaces...)

		for i := range namespaces {
			ns := namespaces[i]

			for j := range ns.Connections {
				conn := ns.Connections[j]

				if conn.Type == tenant.ConnectionTypeNATS {
					natsKey := fmt.Sprintf(connectionKeyFormat, tnt.Identifier, tnt.Version, ns.Name, tenant.InputSourceNATS, conn.Name)
					config := conn.Config.(*tenant.NATSConnection)

					address := capabilities.AugmentedValFromEnv(config.ServerAddress)

					gnats, err := nats.New(address)
					if err != nil {
						c.log.Error(errors.Wrap(err, "failed to nats.New for NATS connection"))
					} else {
						b := bus.New(
							bus.UseLogger(c.log),
							bus.UseBridgeTransport(gnats),
						)

						c.connections[natsKey] = b
					}
				} else if conn.Type == tenant.ConnectionTypeKafka {
					kafkaKey := fmt.Sprintf(connectionKeyFormat, tnt.Identifier, tnt.Version, ns.Name, tenant.InputSourceKafka, conn.Name)
					config := conn.Config.(*tenant.KafkaConnection)

					address := capabilities.AugmentedValFromEnv(config.BrokerAddress)

					gkafka, err := kafka.New(address)
					if err != nil {
						c.log.Error(errors.Wrap(err, "failed to kafka.New for Kafka connection"))
					} else {
						g := bus.New(
							bus.UseLogger(c.log),
							bus.UseBridgeTransport(gkafka),
						)

						c.connections[kafkaKey] = g
					}
				}
			}
		}
	}
}

// TODO: Workflows are not fully implemented, need to add scheduled execution
func (c *Coordinator) SetupWorkflows(router *vk.Router) error {
	tenants := c.syncer.ListTenants()

	// mount each handler into the VK group.
	for ident, _ := range tenants {
		tnt := c.syncer.TenantOverview(ident)
		if tnt == nil {
			continue
		}

		namespaces := []tenant.NamespaceConfig{tnt.Config.DefaultNamespace}
		namespaces = append(namespaces, tnt.Config.Namespaces...)

		for i := range namespaces {
			ns := namespaces[i]

			for j := range ns.Workflows {
				wfl := ns.Workflows[j]

				// mount the workflow's handler to /workflow/{ident}/{namespace}{workflowname}, i.e. /workflow/com.suborbital.appname/default/dosomething
				path := fmt.Sprintf("/workflow/%s/%s/%s", tnt.Config.Identifier, ns.Name, wfl.Name)

				router.POST(path, c.vkHandlerForWorkflow(wfl))

				// TODO: set up scheduled workflows (will need to add a scheduler instance to the coordinator and register them)
				// seconds := wfl.Schedule.NumberOfSeconds()

				// // only actually schedule the job if the env var isn't set (or is set but not 'false')
				// // the job stays mounted on reactr because we could get a request to run it from grav.
				// if *c.opts.RunSchedules && wfl.Schedule != nil {
				// 	c.log.Debug("adding schedule", jobName)

				// 	c.exec.SetSchedule(scheduler.Every(seconds, func() scheduler.Job {
				// 		return scheduler.NewJob(jobName, nil)
				// 	}))
				// }
			}
		}

	}

	return nil
}

// resultFromState returns the state value for the last single function that ran in a handler.
func resultFromState(steps []executable.Executable, state map[string][]byte) []byte {
	// if not, use the last step. If last step is a group, return nil.
	step := steps[len(steps)-1]
	if step.IsGroup() {
		return nil
	}

	// determine what the state traceKey is.
	key := step.FQMN
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
