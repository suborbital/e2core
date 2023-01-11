package sat

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/trace"

	"github.com/suborbital/e2core/foundation/bus/bus"
	"github.com/suborbital/e2core/foundation/bus/discovery/local"
	"github.com/suborbital/e2core/foundation/bus/transport/websocket"
	"github.com/suborbital/e2core/sat/engine2"
	"github.com/suborbital/e2core/sat/engine2/api"
	"github.com/suborbital/e2core/sat/sat/metrics"
	"github.com/suborbital/e2core/sat/sat/process"
	"github.com/suborbital/systemspec/tenant"
	"github.com/suborbital/vektor/vk"
)

// Sat is a sat server with annoyingly terse field names (because it's smol)
type Sat struct {
	config    *Config
	vektor    *vk.Server
	bus       *bus.Bus
	pod       *bus.Pod
	transport *websocket.Transport
	engine    *engine2.Engine
	tracer    trace.Tracer
	metrics   metrics.Metrics
}

type loggerScope struct {
	RequestID string `json:"request_id"`
}

// New initializes a Sat instance
// if traceProvider is nil, the default NoopTraceProvider will be used
func New(config *Config, traceProvider trace.TracerProvider, mtx metrics.Metrics) (*Sat, error) {
	var module *tenant.WasmModuleRef

	if config.Module != nil && config.Module.WasmRef != nil && len(config.Module.WasmRef.Data) > 0 {
		module = config.Module.WasmRef
	} else {
		ref, err := refFromFilename("", "", config.ModuleArg)
		if err != nil {
			return nil, errors.Wrap(err, "faild to refFromFilename")
		}

		module = ref
	}

	api, err := api.NewWithConfig(config.Logger, config.CapConfig)
	if err != nil {
		return nil, errors.Wrap(err, "failed to NewWithConfig")
	}

	engine := engine2.New(config.JobType, module, api)

	if traceProvider == nil {
		traceProvider = trace.NewNoopTracerProvider()
	}

	sat := &Sat{
		config:  config,
		engine:  engine,
		tracer:  traceProvider.Tracer("sat"),
		metrics: mtx,
	}

	// Grav and Vektor will be started on call to s.Start()
	sat.vektor = vk.New(
		vk.UseLogger(config.Logger),
		vk.UseAppName(config.PrettyName),
		vk.UseHTTPPort(config.Port),
		vk.UseEnvPrefix("SAT"),
		vk.UseQuietRoutes("/meta/metrics"),
	)

	// if a transport is configured, enable bus and metrics endpoints, otherwise enable server mode
	if config.ControlPlaneUrl != "" {
		sat.transport = websocket.New()

		sat.vektor.HandleHTTP(http.MethodGet, "/meta/message", sat.transport.HTTPHandlerFunc())
		sat.vektor.GET("/meta/metrics", sat.workerMetricsHandler())
	} else {
		// allow any HTTP method
		sat.vektor.GET("/*any", sat.handler(engine))
		sat.vektor.POST("/*any", sat.handler(engine))
		sat.vektor.PATCH("/*any", sat.handler(engine))
		sat.vektor.DELETE("/*any", sat.handler(engine))
		sat.vektor.HEAD("/*any", sat.handler(engine))
		sat.vektor.OPTIONS("/*any", sat.handler(engine))
	}

	return sat, nil
}

// Start starts Sat's Vektor server and Grav discovery
func (s *Sat) Start(ctx context.Context) error {
	vektorError := make(chan error, 1)

	// start Vektor first so that the server is started up before Bus starts discovery
	go func() {
		if err := s.vektor.Start(); err != nil {
			vektorError <- err
		}
	}()

	go func() {
		if s.transport != nil {
			s.setupBus()
		}
	}()

	select {
	case <-ctx.Done():
		if err := s.Shutdown(); err != nil {
			return errors.Wrap(err, "failed to Shutdown")
		}
	case err := <-vektorError:
		if !errors.Is(err, http.ErrServerClosed) {
			return errors.Wrap(err, "failed to start server")
		}
	}

	return nil
}

func (s *Sat) Shutdown() error {
	s.config.Logger.Info("sat shutting down")

	// stop Bus with a 3s delay between Withdraw and Stop (to allow in-flight requests to drain)
	// s.vektor.Stop isn't called until all connections are ready to close (after said delay)
	// this is needed to ensure a safe withdraw from the constellation/mesh
	if s.transport != nil {
		if err := s.bus.Withdraw(); err != nil {
			s.config.Logger.Warn("encountered error during Withdraw, will proceed:", err.Error())
		}

		time.Sleep(time.Second * 3)

		if err := s.bus.Stop(); err != nil {
			s.config.Logger.Warn("encountered error during Stop, will proceed:", err.Error())
		}
	}

	if err := process.Delete(s.config.ProcUUID); err != nil {
		s.config.Logger.Debug("encountered error during process.Delete, will proceed:", err.Error())
	}

	stopCtx, _ := context.WithTimeout(context.Background(), time.Second)

	if err := s.vektor.StopCtx(stopCtx); err != nil {
		return errors.Wrap(err, "failed to StopCtx")
	}

	return nil
}

func (s *Sat) setupBus() {
	// configure Bus to join the mesh for its appropriate application
	// and broadcast its "interest" (i.e. the loaded function)
	opts := []bus.OptionsModifier{
		bus.UseBelongsTo(s.config.Tenant),
		bus.UseInterests(s.config.JobType),
		bus.UseLogger(s.config.Logger),
		bus.UseMeshTransport(s.transport),
		bus.UseDiscovery(local.New()),
		bus.UseEndpoint(fmt.Sprintf("%d", s.config.Port), "/meta/message"),
	}

	s.bus = bus.New(opts...)
	s.pod = s.bus.Connect()

	s.engine.ListenAndRun(s.bus.Connect(), s.config.JobType, s.handleFnResult)
}

// testStart returns Sat's internal server for testing purposes
func (s *Sat) testServer() *vk.Server {
	return s.vektor
}

func refFromFilename(name, fqmn, filename string) (*tenant.WasmModuleRef, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, errors.Wrap(err, "failed to os.Open")
	}

	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, errors.Wrap(err, "failed to ReadAll")
	}

	ref := tenant.NewWasmModuleRef(name, fqmn, data)

	return ref, nil
}
