package sat

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/trace"

	"github.com/suborbital/e2core/foundation/bus/bus"
	"github.com/suborbital/e2core/foundation/bus/discovery/local"
	"github.com/suborbital/e2core/foundation/bus/transport/websocket"
	"github.com/suborbital/e2core/sat/engine2"
	"github.com/suborbital/e2core/sat/engine2/api"
	"github.com/suborbital/e2core/sat/sat/metrics"
	"github.com/suborbital/e2core/sat/sat/process"
	"github.com/suborbital/systemspec/tenant"
)

// Sat is a sat server with annoyingly terse field names (because it's smol)
type Sat struct {
	config    *Config
	logger    zerolog.Logger
	server    *echo.Echo
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
func New(config *Config, logger zerolog.Logger, traceProvider trace.TracerProvider, mtx metrics.Metrics) (*Sat, error) {
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

	engineAPI, err := api.NewWithConfig(logger, config.CapConfig)
	if err != nil {
		return nil, errors.Wrap(err, "failed to NewWithConfig")
	}

	engine := engine2.New(config.JobType, module, engineAPI)

	if traceProvider == nil {
		traceProvider = trace.NewNoopTracerProvider()
	}

	sat := &Sat{
		config:  config,
		logger:  logger,
		engine:  engine,
		tracer:  traceProvider.Tracer("sat"),
		metrics: mtx,
	}

	sat.server = echo.New()

	// if a "transport" is configured, enable bus and metrics endpoints, otherwise enable server mode
	if config.ControlPlaneUrl != "" {
		sat.transport = websocket.New()

		sat.server.GET("/meta/message", echo.WrapHandler(sat.transport.HTTPHandlerFunc()))
		sat.server.GET("/meta/metrics", sat.workerMetricsHandler())
	} else {
		// allow any HTTP method
		sat.server.Any("*", sat.handler(engine))
	}

	return sat, nil
}

// Start starts Sat's Vektor server and Grav discovery
func (s *Sat) Start() error {
	serverError := make(chan error, 1)

	// start Vektor first so that the server is started up before Bus starts discovery
	go func() {
		if err := s.server.Start(fmt.Sprintf(":%d", s.config.Port)); err != nil {
			serverError <- err
		}
	}()

	go func() {
		if s.transport != nil {
			s.setupBus()
		}
	}()

	select {
	case err := <-serverError:
		if !errors.Is(err, http.ErrServerClosed) {
			return errors.Wrap(err, "failed to start server")
		}
	}

	return nil
}

func (s *Sat) Shutdown() error {
	s.logger.Info().Msg("sat shutting down")

	// stop Bus with a 3s delay between Withdraw and Stop (to allow in-flight requests to drain)
	// s.vektor.Stop isn't called until all connections are ready to close (after said delay)
	// this is needed to ensure a safe withdraw from the constellation/mesh
	if s.transport != nil {
		if err := s.bus.Withdraw(); err != nil {
			s.logger.Err(err).Msg("encountered error during bus.Withdraw, will proceed")
		}

		time.Sleep(time.Second * 3)

		if err := s.bus.Stop(); err != nil {
			s.logger.Err(err).Msg("encountered error during bus.Stop, will proceed")
		}
	}

	if err := process.Delete(s.config.ProcUUID); err != nil {
		s.logger.Err(err).Msg("encountered error during process.Delete, will proceed")
	}

	stopCtx, cxl := context.WithTimeout(context.Background(), time.Second)
	defer cxl()

	if err := s.server.Shutdown(stopCtx); err != nil {
		return errors.Wrap(err, "failed to echo.Shutdown()")
	}

	return nil
}

func (s *Sat) setupBus() {
	// configure Bus to join the mesh for its appropriate application
	// and broadcast its "interest" (i.e. the loaded function)
	opts := []bus.OptionsModifier{
		bus.UseBelongsTo(s.config.Tenant),
		bus.UseInterests(s.config.JobType),
		bus.UseLogger(s.logger),
		bus.UseMeshTransport(s.transport),
		bus.UseDiscovery(local.New()),
		bus.UseEndpoint(fmt.Sprintf("%d", s.config.Port), "/meta/message"),
	}

	s.bus = bus.New(opts...)
	s.pod = s.bus.Connect()

	s.engine.ListenAndRun(s.bus.Connect(), s.config.JobType, s.handleFnResult)
}

// testStart returns Sat's internal server for testing purposes
func (s *Sat) testServer() *echo.Echo {
	return s.server
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
