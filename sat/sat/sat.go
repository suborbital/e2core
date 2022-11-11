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

	"github.com/suborbital/appspec/tenant"
	"github.com/suborbital/e2core/bus/bus"
	"github.com/suborbital/e2core/bus/discovery/local"
	"github.com/suborbital/e2core/bus/transport/websocket"
	wruntime "github.com/suborbital/e2core/sat/engine/runtime"
	"github.com/suborbital/e2core/sat/sat/executor"
	"github.com/suborbital/e2core/sat/sat/metrics"
	"github.com/suborbital/e2core/sat/sat/process"
	"github.com/suborbital/e2core/scheduler"
	"github.com/suborbital/vektor/vk"
	"github.com/suborbital/vektor/vlog"
)

const (
	MsgTypeAtmoFnResult = "atmo.fnresult"
)

// Sat is a sat server with annoyingly terse field names (because it's smol)
type Sat struct {
	jobName string // the job name / FQFN

	config    *Config
	vektor    *vk.Server
	bus       *bus.Bus
	transport *websocket.Transport
	exec      *executor.Executor
	log       *vlog.Logger
	tracer    trace.Tracer
	metrics   metrics.Metrics
}

type loggerScope struct {
	RequestID string `json:"request_id"`
}

// New initializes Reactr, Vektor, and Grav in a Sat instance
// if config.UseStdin is true, only Reactr will be created
// if traceProvider is nil, the default NoopTraceProvider will be used
func New(config *Config, traceProvider trace.TracerProvider, mtx metrics.Metrics) (*Sat, error) {
	wruntime.UseInternalLogger(config.Logger)

	exec, err := executor.New(config.Logger, config.CapConfig)
	if err != nil {
		return nil, errors.Wrap(err, "failed to executor.New")
	}

	var runnable *tenant.WasmModuleRef
	if config.Module != nil && len(config.Module.WasmRef.Data) > 0 {
		runnable = tenant.NewWasmModuleRef(config.Module.WasmRef.Name, config.Module.WasmRef.FQMN, config.Module.WasmRef.Data)
	} else {
		ref, err := refFromFilename("", "", config.RunnableArg)
		if err != nil {
			return nil, errors.Wrap(err, "faild to refFromFilename")
		}

		runnable = ref
	}

	err = exec.Register(
		config.JobType,
		runnable,
		scheduler.Autoscale(24),
		scheduler.MaxRetries(0),
		scheduler.RetrySeconds(0),
		scheduler.PreWarm(),
	)

	if err != nil {
		return nil, errors.Wrap(err, "exec.Register")
	}

	if traceProvider == nil {
		traceProvider = trace.NewNoopTracerProvider()
	}

	var transport *websocket.Transport
	if config.ControlPlaneUrl != "" {
		transport = websocket.New()
	}

	sat := &Sat{
		jobName:   config.JobType,
		config:    config,
		transport: transport,
		exec:      exec,
		log:       config.Logger,
		tracer:    traceProvider.Tracer("sat"),
		metrics:   mtx,
	}

	// no need to continue setup if we're in stdin mode, so return here
	if config.UseStdin {
		return sat, nil
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
	if sat.transport != nil {
		sat.vektor.HandleHTTP(http.MethodGet, "/meta/message", sat.transport.HTTPHandlerFunc())
		sat.vektor.GET("/meta/metrics", sat.workerMetricsHandler())
	} else {
		// allow any HTTP method
		sat.vektor.GET("/*any", sat.handler(exec))
		sat.vektor.POST("/*any", sat.handler(exec))
		sat.vektor.PATCH("/*any", sat.handler(exec))
		sat.vektor.DELETE("/*any", sat.handler(exec))
		sat.vektor.HEAD("/*any", sat.handler(exec))
		sat.vektor.OPTIONS("/*any", sat.handler(exec))
	}

	return sat, nil
}

// Start starts Sat's Vektor server and Grav discovery
func (s *Sat) Start(ctx context.Context) error {
	vektorError := make(chan error, 1)

	// start Vektor first so that the server is started up before Grav starts discovery
	go func() {
		if err := s.vektor.Start(); err != nil {
			vektorError <- err
		}
	}()

	if s.transport != nil {
		if err := s.setupGrav(); err != nil {
			return errors.Wrap(err, "failed to setupGrav")
		}
	}

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
	s.log.Info("sat shutting down")

	// stop Bus with a 3s delay between Withdraw and Stop (to allow in-flight requests to drain)
	// s.vektor.Stop isn't called until all connections are ready to close (after said delay)
	// this is needed to ensure a safe withdraw from the constellation/mesh
	if s.transport != nil {
		if err := s.bus.Withdraw(); err != nil {
			s.log.Warn("encountered error during Withdraw, will proceed:", err.Error())
		}

		time.Sleep(time.Second * 3)

		if err := s.bus.Stop(); err != nil {
			s.log.Warn("encountered error during Stop, will proceed:", err.Error())
		}
	}

	if err := process.Delete(s.config.ProcUUID); err != nil {
		s.log.Debug("encountered error during process.Delete, will proceed:", err.Error())
	}

	stopCtx, _ := context.WithTimeout(context.Background(), time.Second)

	if err := s.vektor.StopCtx(stopCtx); err != nil {
		return errors.Wrap(err, "failed to StopCtx")
	}

	return nil
}

func (s *Sat) setupGrav() error {
	// configure Grav to join the mesh for its appropriate application
	// and broadcast its "interest" (i.e. the loaded function)
	s.bus = bus.New(
		bus.UseBelongsTo(s.config.Identifier),
		bus.UseInterests(s.config.JobType),
		bus.UseLogger(s.config.Logger),
		bus.UseMeshTransport(s.transport),
		bus.UseDiscovery(local.New()),
		bus.UseEndpoint(fmt.Sprintf("%d", s.config.Port), "/meta/message"),
	)

	// set up the Executor to listen for jobs and handle them
	s.exec.UseBus(s.bus)

	if err := s.exec.ListenAndRun(s.config.JobType, s.handleFnResult); err != nil {
		return errors.Wrap(err, "executor.ListenAndRun")
	}

	if err := connectStaticPeers(s.config.Logger, s.bus); err != nil {
		return errors.Wrap(err, "failed to connectStaticPeers")
	}

	return nil
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
