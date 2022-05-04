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

func (a *Server) testServer() (*vk.Server, error) {
	if err := a.coordinator.Start(); err != nil {
		return nil, errors.Wrap(err, "failed to coordinator.Start")
	}

	// mount and set up the app's handlers.
	router := a.coordinator.SetupHandlers()
	a.server.SwapRouter(router)

	return a.server, nil
}

// New creates a new Server instance.
func New(opts ...options.Modifier) (*Server, error) {
	atmoOpts := options.NewWithModifiers(opts...)

	// @todo https://github.com/suborbital/velocity/issues/144, the first return value is a function that would close the
	// tracer in case of a shutdown. Usually that is put in a defer statement. Server doesn't have a graceful shutdown.
	_, err := setupTracing(atmoOpts.TracerConfig, atmoOpts.Logger)
	if err != nil {
		return nil, errors.Wrapf(err, "setupTracing(%s, %s, %f)", "atmo", "reporter_uri", 0.04)
	}

	appSource := appsource.NewBundleSource(atmoOpts.BundlePath)
	if atmoOpts.ControlPlane != "" {
		// the HTTP appsource gets Server's data from a remote server
		// which can essentially control Server's behaviour.
		appSource = appsource.NewHTTPSource(atmoOpts.ControlPlane)
	} else if *atmoOpts.Headless {
		// the headless appsource ignores the Directive and mounts
		// each Runnable as its own route (for testing, other purposes).
		appSource = appsource.NewHeadlessBundleSource(atmoOpts.BundlePath)
	}

	a := &Server{
		coordinator: coordinator.New(appSource, atmoOpts),
		options:     atmoOpts,
	}

	// set up the server so that Server can inspect
	// each request to trigger Router re-generation
	// when needed (during headless mode).
	a.server = vk.New(
		vk.UseEnvPrefix("VELOCITY"),
		vk.UseAppName(atmoOpts.AppName),
		vk.UseLogger(atmoOpts.Logger),
		vk.UseInspector(a.inspectRequest),
		vk.UseDomain(atmoOpts.Domain),
		vk.UseHTTPPort(atmoOpts.HTTPPort),
		vk.UseTLSPort(atmoOpts.TLSPort),
		vk.UseQuietRoutes(
			coordinator.VelocityHealthURI,
			coordinator.VelocityMetricsURI,
		),
		vk.UseRouterWrapper(func(inner http.Handler) http.Handler {
			return otelhttp.NewHandler(inner, "atmo")
		}),
	)

	return a, nil
}

// Start starts the Server server.
func (a *Server) Start() error {
	if err := a.coordinator.Start(); err != nil {
		return errors.Wrap(err, "failed to coordinator.Start")
	}

	// mount and set up the app's handlers.
	router := a.coordinator.SetupHandlers()
	a.server.SwapRouter(router)

	// mount the schedules defined in the App.
	a.coordinator.SetSchedules()

	if err := a.server.Start(); err != nil {
		return errors.Wrap(err, "failed to server.Start")
	}

	return nil
}

// inspectRequest is critical and runs BEFORE every single request that Server receives, which means it must be very efficient
// and only block the request if it's absolutely needed. It is a no-op unless Server is in headless mode, and even then only
// does anything if a request is made for a function that we've never seen before. Read on to see what it does.
func (a *Server) inspectRequest(r http.Request) {
	// we only need to inspect the request
	// if we're in headless mode.
	if !*a.options.Headless {
		return
	}

	// if Vektor tells us it cannot handle the headless request (i.e. we have no knowledge of the function in question)
	// then ask the AppSource to find it, and if successful sync Runnables into Reactr and generate a new Router.
	if !a.server.CanHandle(r.Method, r.URL.Path) {
		FQFN, err := fqfn.FromURL(r.URL)
		if err != nil {
			a.options.Logger.Debug(errors.Wrap(err, "failed to fqfn.FromURL, likely invalid, request will proceed and fail").Error())
			return
		}

		// the Authorization header is passed through to the AppSource, and can be used to authenticate calls.
		auth := r.Header.Get("Authorization")

		// use the configured global env token for all requests (if available)
		if a.options.EnvironmentToken != "" {
			auth = a.options.EnvironmentToken
		}

		if _, err := a.coordinator.App.FindRunnable(FQFN, auth); err != nil {
			a.options.Logger.Debug(errors.Wrapf(err, "failed to FindRunnable %s, request will proceed and fail", FQFN).Error())
			return
		}

		a.options.Logger.Debug(fmt.Sprintf("found new Runnable %s, will be available at next sync", FQFN))

		// do a sync to load the new Runnable into Reactr.
		a.coordinator.SyncAppState()

		// re-generate the Router which should now include
		// the new function as a handler.
		newRouter := a.coordinator.SetupHandlers()
		a.server.SwapRouter(newRouter)

		a.options.Logger.Debug("app sync and router swap completed")
	}
}
