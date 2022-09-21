package server

import (
	"context"
	"net/http"

	"github.com/pkg/errors"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/suborbital/appspec/appsource/bundle"
	"github.com/suborbital/appspec/appsource/client"
	"github.com/suborbital/e2core/options"
	"github.com/suborbital/e2core/server/coordinator"
	"github.com/suborbital/e2core/syncer"
	"github.com/suborbital/vektor/vk"
)

// Server is a E2Core server.
type Server struct {
	coordinator *coordinator.Coordinator
	server      *vk.Server
	syncer      *syncer.Syncer

	options *options.Options
}

// New creates a new Server instance.
func New(opts ...options.Modifier) (*Server, error) {
	vOpts := options.NewWithModifiers(opts...)

	// @todo https://github.com/suborbital/e2core/issues/144, the first return value is a function that would close the
	// tracer in case of a shutdown. Usually that is put in a defer statement. Server doesn't have a graceful shutdown.
	_, err := setupTracing(vOpts.TracerConfig, vOpts.Logger())
	if err != nil {
		return nil, errors.Wrapf(err, "setupTracing(%s, %s, %f)", "atmo", "reporter_uri", 0.04)
	}

	// TODO: implement and use a CredentialSupplier
	appSource := bundle.NewBundleSource(vOpts.BundlePath)
	if vOpts.ControlPlane != "" {
		// the HTTP appsource gets Server's data from a remote server
		// which can essentially control Server's behaviour.
		appSource = client.NewHTTPSource(vOpts.ControlPlane, nil)
	}

	syncer := syncer.New(vOpts, appSource)

	s := &Server{
		coordinator: coordinator.New(syncer, vOpts),
		syncer:      syncer,
		options:     vOpts,
	}

	// set up the server so that Server can inspect
	// each request to trigger Router re-generation
	// when needed (during headless mode).
	s.server = vk.New(
		vk.UseEnvPrefix("E2CORE"),
		vk.UseAppName(vOpts.AppName),
		vk.UseLogger(vOpts.Logger()),
		vk.UseDomain(vOpts.Domain),
		vk.UseHTTPPort(vOpts.HTTPPort),
		vk.UseTLSPort(vOpts.TLSPort),
		vk.UseQuietRoutes(
			coordinator.E2CoreHealthURI,
			coordinator.E2CoreMetricsURI,
		),
		vk.UseRouterWrapper(func(inner http.Handler) http.Handler {
			return otelhttp.NewHandler(inner, "e2core")
		}),
	)

	return s, nil
}

// Start starts the Server server.
func (s *Server) Start(ctx context.Context) error {
	if err := s.coordinator.Start(); err != nil {
		return errors.Wrap(err, "failed to coordinator.Start")
	}

	router, err := s.coordinator.SetupHandlers()
	if err != nil {
		return errors.Wrap(err, "failed to SetupHandlers")
	}

	if err := s.coordinator.SetupWorkflows(router); err != nil {
		return errors.Wrap(err, "failed to SetupWorkflows")
	}

	s.server.SwapRouter(router)

	if err := s.server.Start(); err != nil {
		return errors.Wrap(err, "failed to server.Start")
	}

	return nil
}

// Options returns the options that the server was configured with
func (s *Server) Options() options.Options {
	return *s.options
}

// Syncer returns the Syncer that the server was configured with
func (s *Server) Syncer() *syncer.Syncer {
	return s.syncer
}

func (s *Server) testServer() (*vk.Server, error) {
	if err := s.coordinator.Start(); err != nil {
		return nil, errors.Wrap(err, "failed to coordinator.Start")
	}

	// mount and set up the app's handlers.
	router, err := s.coordinator.SetupHandlers()
	if err != nil {
		return nil, errors.Wrap(err, "failed to SetupHandlers")
	}

	if err := s.coordinator.SetupWorkflows(router); err != nil {
		return nil, errors.Wrap(err, "failed to SetupWorkflows")
	}

	s.server.SwapRouter(router)

	return s.server, nil
}
