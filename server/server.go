package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/pkg/errors"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/suborbital/appspec/appsource/bundle"
	"github.com/suborbital/appspec/appsource/client"
	"github.com/suborbital/e2core/fqfn"
	"github.com/suborbital/e2core/options"
	"github.com/suborbital/e2core/server/coordinator"
	"github.com/suborbital/vektor/vk"
)

// Server is a DeltaV server.
type Server struct {
	coordinator *coordinator.Coordinator
	server      *vk.Server

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

	s := &Server{
		coordinator: coordinator.New(appSource, vOpts),
		options:     vOpts,
	}

	// set up the server so that Server can inspect
	// each request to trigger Router re-generation
	// when needed (during headless mode).
	s.server = vk.New(
		vk.UseEnvPrefix("DELTAV"),
		vk.UseAppName(vOpts.AppName),
		vk.UseLogger(vOpts.Logger()),
		vk.UseInspector(s.inspectRequest),
		vk.UseDomain(vOpts.Domain),
		vk.UseHTTPPort(vOpts.HTTPPort),
		vk.UseTLSPort(vOpts.TLSPort),
		vk.UseQuietRoutes(
			coordinator.DeltavHealthURI,
			coordinator.DeltavMetricsURI,
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

// inspectRequest is critical and runs BEFORE every single request that Server receives, which means it must be very efficient
// and only block the request if it's absolutely needed. It is a no-op unless Server is in headless mode, and even then only
// does anything if a request is made for a function that we've never seen before. Read on to see what it does.
func (s *Server) inspectRequest(r http.Request) {
	// we only need to inspect the request
	// if we're in headless mode.
	if !*s.options.Headless {
		return
	}

	// if Vektor tells us it cannot handle the headless request (i.e. we have no knowledge of the function in question)
	// then ask the AppSource to find it, and if successful sync Runnables into Reactr and generate a new Router.
	if !s.server.CanHandle(r.Method, r.URL.Path) {
		FQFN, err := fqfn.FromURL(r.URL)
		if err != nil {
			s.options.Logger().Debug(errors.Wrap(err, "failed to fqfn.FromURL, likely invalid, request will proceed and fail").Error())
			return
		}

		// the Authorization header is passed through to the AppSource, and can be used to authenticate calls.
		// TODO: implement properly with CredentialSupplier
		// auth := r.Header.Get("Authorization")

		// use the configured global env token for all requests (if available)
		// if s.options.EnvironmentToken != "" {
		// 	auth = s.options.EnvironmentToken
		// }

		if _, err := s.coordinator.App.GetModule(FQFN); err != nil {
			s.options.Logger().Debug(errors.Wrapf(err, "failed to FindRunnable %s, request will proceed and fail", FQFN).Error())
			return
		}

		s.options.Logger().Debug(fmt.Sprintf("found new Runnable %s, will be available at next sync", FQFN))

		// re-generate the Router which should now include
		// the new function as a handler.
		newRouter, err := s.coordinator.SetupHandlers()
		if err != nil {
			s.options.Logger().Error(errors.Wrap(err, "failed to SetupHandlers"))
		}

		s.server.SwapRouter(newRouter)

		s.options.Logger().Debug("app sync and router swap completed")
	}
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
