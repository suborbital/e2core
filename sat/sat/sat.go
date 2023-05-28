package sat

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"

	"github.com/suborbital/e2core/foundation/bus/bus"
	"github.com/suborbital/e2core/foundation/bus/discovery/local"
	"github.com/suborbital/e2core/foundation/bus/transport/websocket"
	"github.com/suborbital/e2core/nuexecutor/handlers"
	"github.com/suborbital/e2core/nuexecutor/worker"
	"github.com/suborbital/e2core/sat/engine2"
	"github.com/suborbital/e2core/sat/engine2/api"
	"github.com/suborbital/e2core/sat/sat/metrics"
	kitError "github.com/suborbital/go-kit/web/error"
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
	metrics   metrics.Metrics
}

// New initializes a Sat instance
// if traceProvider is nil, the default NoopTraceProvider will be used
func New(config *Config, logger zerolog.Logger, mtx metrics.Metrics) (*Sat, error) {
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

	engine := engine2.New(config.JobType, module, engineAPI, logger)

	sat := &Sat{
		config:  config,
		logger:  logger,
		engine:  engine,
		metrics: mtx,
	}

	w, err := worker.New(worker.Config{}, logger, module.Data)
	if err != nil {
		return nil, errors.Wrap(err, "worker.New")
	}

	wc := w.Start()

	sat.server = echo.New()
	sat.server.Use(
		otelecho.Middleware("e2core-bebby"),
		middleware.Recover(),
	)
	sat.server.HTTPErrorHandler = kitError.Handler(logger)
	sat.server.HideBanner = true
	sat.server.HidePort = true

	// if a "transport" is configured, enable bus and metrics endpoints, otherwise enable server mode
	if config.ControlPlaneUrl != "" {
		logger.Info().Msg("controlplane url is present, creating the websocket for transport, and the meta/message and meta/metrics endpoints")
		sat.transport = websocket.New()

		sat.server.POST("/meta/sync", handlers.Sync(wc))

		sat.server.GET("/meta/message", echo.WrapHandler(sat.transport.HTTPHandlerFunc()))
		sat.server.GET("/meta/metrics", sat.workerMetricsHandler())
	} else {
		logger.Info().Msg("controlplane url is not present, pass anything to sat.handler")
		// allow any HTTP method
		sat.server.Any("*", sat.handler(engine))
	}

	return sat, nil
}

// Start starts Sat's echo server and Bus discovery.
func (s *Sat) Start() error {
	serverError := make(chan error, 1)

	// start Echo first so that the server is started up before Bus starts discovery.
	go func() {
		if err := s.server.Start(fmt.Sprintf(":%d", s.config.Port)); err != nil {
			serverError <- err
		}
	}()
	//
	// go func() {
	// 	if s.transport != nil {
	// 		s.setupBus()
	// 	}
	// }()

	select {
	case err := <-serverError:
		if !errors.Is(err, http.ErrServerClosed) {
			return errors.Wrap(err, "failed to start server")
		}
	}

	return nil
}

func (s *Sat) Shutdown() error {
	ll := s.logger.With().Str("func", "Sat.Shutdown").Logger()

	ll.Info().Msg("sat shutting down")

	// stop Bus with a 3s delay between Withdraw and Stop (to allow in-flight requests to drain)
	// s.server.Shutdown isn't called until all connections are ready to close (after said delay)
	// this is needed to ensure a safe withdraw from the constellation/mesh
	// if s.transport != nil {
	// 	ll.Info().Msg("shutting down transport")
	// 	if err := s.bus.Withdraw(); err != nil {
	// 		ll.Err(err).Msg("encountered error during bus.Withdraw, will proceed")
	// 	}
	//
	// 	time.Sleep(time.Second * 3)
	//
	// 	if err := s.bus.Stop(); err != nil {
	// 		s.logger.Err(err).Msg("encountered error during bus.Stop, will proceed")
	// 	}
	//
	// 	ll.Info().Msg("transport shutdown finished")
	// }
	//
	// ll.Info().Str("proc_uuid", s.config.ProcUUID).Msg("trying to process delete this process uuid")
	// if err := process.Delete(s.config.ProcUUID); err != nil {
	// 	s.logger.Err(err).Msg("encountered error during process.Delete, will proceed")
	// }

	// ll.Info().Str("proc_uuid", s.config.ProcUUID).Msg("process delete finished")

	stopCtx, cxl := context.WithTimeout(context.Background(), time.Second)
	defer cxl()

	ll.Info().Msg("shutting down echo server")
	if err := s.server.Shutdown(stopCtx); err != nil {
		return errors.Wrap(err, "failed to echo.Shutdown()")
	}

	ll.Info().Msg("echo server shutdown completed, everything shut down. Good night!")

	return nil
}

func (s *Sat) setupBus() {
	// configure Bus to join the mesh for its appropriate application
	// and broadcast its "interest" (i.e. the loaded function)
	opts := []bus.OptionsModifier{
		bus.UseBelongsTo(s.config.Tenant),
		bus.UseInterests(s.config.JobType),
		bus.UseLogger(s.logger.With().Str("source", "sat.setupBus").Logger()),
		bus.UseMeshTransport(s.transport),
		bus.UseDiscovery(local.New()),
		bus.UseEndpoint(fmt.Sprintf("%d", s.config.Port), "/meta/message"),
	}

	s.bus = bus.New(opts...)
	s.pod = s.bus.Connect()

	s.engine.ListenAndRun(s.bus.Connect(), s.config.JobType, s.handleFnResult)
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
