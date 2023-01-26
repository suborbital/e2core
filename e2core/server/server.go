package server

import (
	"context"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/suborbital/e2core/e2core/auth"
	"github.com/suborbital/e2core/e2core/options"
	"github.com/suborbital/e2core/e2core/syncer"
	"github.com/suborbital/e2core/foundation/bus/bus"
	"github.com/suborbital/e2core/foundation/bus/discovery/local"
	"github.com/suborbital/e2core/foundation/bus/transport/websocket"
	"github.com/suborbital/vektor/vk"
)

const E2CoreHealthURI = "/health"

// Server is a E2Core server.
type Server struct {
	server *echo.Echo
	syncer *syncer.Syncer

	bus        *bus.Bus
	dispatcher *dispatcher

	options *options.Options
}

// New creates a new Server instance.
func New(l zerolog.Logger, sync *syncer.Syncer, opts *options.Options) (*Server, error) {
	ll := l.With().Str("module", "syncer").Logger()
	// @todo https://github.com/suborbital/e2core/issues/144, the first return value is a function that would close the
	// tracer in case of a shutdown. Usually that is put in a defer statement. Server doesn't have a graceful shutdown.
	tracingShutdown, err := setupTracing(opts.TracerConfig, ll)
	if err != nil {
		return nil, errors.Wrapf(err, "setupTracing(%s, %s, %f)", "e2core", "reporter_uri", 0.04)
	}

	busOpts := []bus.OptionsModifier{
		bus.UseMeshTransport(websocket.New()),
		bus.UseDiscovery(local.New()),
	}

	b := bus.New(busOpts...)

	e := echo.New()
	// trace middleware
	e.Use(otelecho.Middleware("e2core"))

	d := newDispatcher(ll, b.Connect())

	s := vk.New(
		vk.UseEnvPrefix("E2CORE"),
		vk.UseAppName(opts.AppName),
		vk.UseLogger(opts.Logger()),
		vk.UseDomain(opts.Domain),
		vk.UseHTTPPort(opts.HTTPPort),
		vk.UseTLSPort(opts.TLSPort),
		vk.UseQuietRoutes(
			E2CoreHealthURI,
		),
		vk.UseRouterWrapper(func(inner http.Handler) http.Handler {
			return otelhttp.NewHandler(inner, "e2core")
		}),
	)

	server := &Server{
		server:     e,
		syncer:     sync,
		options:    opts,
		bus:        b,
		dispatcher: d,
	}

	router := vk.NewRouter(opts.Logger(), "")

	if opts.AdminEnabled() {
		router.POST("/name/:ident/:namespace/:name", auth.AuthorizationMiddleware(opts, server.executePluginByNameHandler()))
	} else {
		router.POST("/name/:ident/:namespace/:name", server.executePluginByNameHandler())
		router.POST("/ref/:ref", server.executePluginByRefHandler())
		router.POST("/workflow/:ident/:namespace/:name", server.executeWorkflowHandler())
	}

	router.GET("/health", server.healthHandler())

	server.server.SwapRouter(router)

	return server, nil
}

// Start starts the Server server.
func (s *Server) Start(ctx context.Context) error {
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

// Shutdown shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	if err := s.server.StopCtx(ctx); err != nil {
		return errors.Wrap(err, "http.Server.StopCtx")
	}

	return nil
}

func (s *Server) testServer() *vk.Server {
	return s.server
}
