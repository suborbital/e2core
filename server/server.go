package server

import (
	"fmt"
	"net/http"

	"github.com/pkg/errors"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/suborbital/vektor/vk"
	"github.com/suborbital/velocity/fqfn"
	"github.com/suborbital/velocity/server/appsource"
	"github.com/suborbital/velocity/server/coordinator"
	"github.com/suborbital/velocity/server/options"
)

// Server is a Velocity server.
type Server struct {
	coordinator *coordinator.Coordinator
	server      *vk.Server

	options *options.Options
}

func (s *Server) testServer() (*vk.Server, error) {
	if err := s.coordinator.Start(); err != nil {
		return nil, errors.Wrap(err, "failed to coordinator.Start")
	}

	// mount and set up the app's handlers.
	router := s.coordinator.SetupHandlers()
	s.server.SwapRouter(router)

	return s.server, nil
}

// New creates a new Server instance.
func New(opts ...options.Modifier) (*Server, error) {
	vOpts := options.NewWithModifiers(opts...)

	if vOpts.PartnerAddress != "" {
		vOpts.Logger.Info("using partner", vOpts.PartnerAddress)
	}

	// @todo https://github.com/suborbital/velocity/issues/144, the first return value is a function that would close the
	// tracer in case of a shutdown. Usually that is put in a defer statement. Server doesn't have a graceful shutdown.
	_, err := setupTracing(vOpts.TracerConfig, vOpts.Logger)
	if err != nil {
		return nil, errors.Wrapf(err, "setupTracing(%s, %s, %f)", "atmo", "reporter_uri", 0.04)
	}

	appSource := appsource.NewBundleSource(vOpts.BundlePath)
	if vOpts.ControlPlane != "" {
		// the HTTP appsource gets Server's data from a remote server
		// which can essentially control Server's behaviour.
		appSource = appsource.NewHTTPSource(vOpts.ControlPlane)
	} else if *vOpts.Headless {
		// the headless appsource ignores the Directive and mounts
		// each Runnable as its own route (for testing, other purposes).
		appSource = appsource.NewHeadlessBundleSource(vOpts.BundlePath)
	}

	s := &Server{
		coordinator: coordinator.New(appSource, vOpts),
		options:     vOpts,
	}

	// set up the server so that Server can inspect
	// each request to trigger Router re-generation
	// when needed (during headless mode).
	s.server = vk.New(
		vk.UseEnvPrefix("VELOCITY"),
		vk.UseAppName(vOpts.AppName),
		vk.UseLogger(vOpts.Logger),
		vk.UseInspector(s.inspectRequest),
		vk.UseDomain(vOpts.Domain),
		vk.UseHTTPPort(vOpts.HTTPPort),
		vk.UseTLSPort(vOpts.TLSPort),
		vk.UseQuietRoutes(
			coordinator.VelocityHealthURI,
			coordinator.VelocityMetricsURI,
		),
		vk.UseRouterWrapper(func(inner http.Handler) http.Handler {
			return otelhttp.NewHandler(inner, "velocity")
		}),
		vk.UseFallbackAddress(vOpts.PartnerAddress),
	)

	return s, nil
}

// Start starts the Server server.
func (s *Server) Start() error {
	if err := s.coordinator.Start(); err != nil {
		return errors.Wrap(err, "failed to coordinator.Start")
	}

	// mount and set up the app's handlers.
	router := s.coordinator.SetupHandlers()
	s.server.SwapRouter(router)

	// mount the schedules defined in the App.
	s.coordinator.SetSchedules()

	if err := s.server.Start(); err != nil {
		return errors.Wrap(err, "failed to server.Start")
	}

	return nil
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
			s.options.Logger.Debug(errors.Wrap(err, "failed to fqfn.FromURL, likely invalid, request will proceed and fail").Error())
			return
		}

		// the Authorization header is passed through to the AppSource, and can be used to authenticate calls.
		auth := r.Header.Get("Authorization")

		// use the configured global env token for all requests (if available)
		if s.options.EnvironmentToken != "" {
			auth = s.options.EnvironmentToken
		}

		if _, err := s.coordinator.App.FindRunnable(FQFN, auth); err != nil {
			s.options.Logger.Debug(errors.Wrapf(err, "failed to FindRunnable %s, request will proceed and fail", FQFN).Error())
			return
		}

		s.options.Logger.Debug(fmt.Sprintf("found new Runnable %s, will be available at next sync", FQFN))

		// do a sync to load the new Runnable into Reactr.
		s.coordinator.SyncAppState()

		// re-generate the Router which should now include
		// the new function as a handler.
		newRouter := s.coordinator.SetupHandlers()
		s.server.SwapRouter(newRouter)

		s.options.Logger.Debug("app sync and router swap completed")
	}
}
