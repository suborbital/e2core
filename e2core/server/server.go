package server

import (
	"context"
	"fmt"

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"

	"github.com/suborbital/e2core/e2core/auth"
	"github.com/suborbital/e2core/e2core/options"
	"github.com/suborbital/e2core/e2core/syncer"
	"github.com/suborbital/e2core/foundation/bus/bus"
	"github.com/suborbital/e2core/foundation/bus/discovery/local"
	"github.com/suborbital/e2core/foundation/bus/transport/websocket"
	kitError "github.com/suborbital/go-kit/web/error"
	"github.com/suborbital/go-kit/web/mid"
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
	ll := l.With().Str("module", "server").Logger()

	// @todo https://github.com/suborbital/e2core/issues/144, the first return value is a function that would close the
	// tracer in case of a shutdown. Usually that is put in a defer statement. Server doesn't have a graceful shutdown.
	_, err := setupTracing(opts.TracerConfig, ll)
	if err != nil {
		return nil, errors.Wrapf(err, "setupTracing(%s, %s, %f)", "e2core", "reporter_uri", 0.04)
	}

	busOpts := []bus.OptionsModifier{
		bus.UseMeshTransport(websocket.New()),
		bus.UseDiscovery(local.New()),
	}

	b := bus.New(busOpts...)

	e := echo.New()
	e.HTTPErrorHandler = kitError.Handler(l)
	e.HideBanner = true

	e.Use(
		mid.UUIDRequestID(),
		otelecho.Middleware("e2core"),
	)

	d := newDispatcher(ll, b.Connect())

	server := &Server{
		server:     e,
		syncer:     sync,
		options:    opts,
		bus:        b,
		dispatcher: d,
	}

	if opts.AdminEnabled() {
		e.POST("/name/:ident/:namespace/:name", server.executePluginByNameHandler(), auth.AuthorizationMiddleware(opts))
	} else {
		e.POST("/name/:ident/:namespace/:name", server.executePluginByNameHandler())
		e.POST("/ref/:ref", server.executePluginByRefHandler(ll))
		e.POST("/workflow/:ident/:namespace/:name", server.executeWorkflowHandler())
	}

	e.GET("/health", server.healthHandler())

	return server, nil
}

// Start starts the Server.
func (s *Server) Start() error {
	if err := s.server.Start(fmt.Sprintf(":%d", s.Options().HTTPPort)); err != nil {
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
	if err := s.server.Shutdown(ctx); err != nil {
		return errors.Wrap(err, "http.Server.StopCtx")
	}

	return nil
}

func (s *Server) testServer() *echo.Echo {
	return s.server
}
