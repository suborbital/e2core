package vk

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/julienschmidt/httprouter"
	"golang.org/x/crypto/acme/autocert"
)

const defaultEnvPrefix = "VK"

// Server represents a vektor API server
type Server struct {
	router  *Router
	lock    sync.RWMutex
	started atomic.Value

	server  *http.Server
	options *Options
}

// New creates a new vektor API server
func New(opts ...OptionsModifier) *Server {
	options := newOptsWithModifiers(opts...)

	router := NewRouter(options.Logger)

	s := &Server{
		router:  router,
		lock:    sync.RWMutex{},
		started: atomic.Value{},
		options: options,
	}

	s.started.Store(false)

	// yes this creates a circular reference,
	// but the VK server and HTTP server are
	// extremely tightly wound together so
	// we have to make this compromise
	s.server = createGoServer(options, s)

	return s
}

// Start starts the server listening
func (s *Server) Start() error {
	// lock the router modifiers (GET, POST etc.)
	s.started.Store(true)

	// mount the root set of routes before starting
	s.router.Finalize()

	if s.options.AppName != "" {
		s.options.Logger.Info("starting", s.options.AppName, "...")
	}

	s.options.Logger.Info("serving on", s.server.Addr)

	if !s.options.HTTPPortSet() && !s.options.ShouldUseTLS() {
		s.options.Logger.ErrorString("domain and HTTP port options are both unset, server will start up but fail to acquire a certificate. reconfigure and restart")
	} else if s.options.ShouldUseHTTP() {
		return s.server.ListenAndServe()
	}

	return s.server.ListenAndServeTLS("", "")
}

// ServeHTTP serves HTTP requests using the internal router while allowing
// said router to be swapped out underneath at any time in a thread-safe way
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// run the inspector with a dereferenced pointer
	// so that it can view but not change said request
	//
	// we intentionally run this before the lock as it's
	// possible the inspector may trigger a router-swap
	// and that would cause a nasty deadlock
	s.options.PreRouterInspector(*r)

	// now lock to ensure the router isn't being swapped
	// out from underneath us while we're serving this req
	s.lock.RLock()
	defer s.lock.RUnlock()

	s.router.ServeHTTP(w, r)
}

// SwapRouter allows swapping VK's router out in realtime while
// continuing to serve requests in the background
func (s *Server) SwapRouter(router *Router) {
	router.Finalize()

	// lock after Finalizing the router so
	// the lock is released as quickly as possible
	s.lock.Lock()
	defer s.lock.Unlock()

	s.router = router
}

// CanHandle returns true if the server can handle a given method and path
func (s *Server) CanHandle(method, path string) bool {
	s.lock.RLock()
	defer s.lock.RUnlock()

	return s.router.canHandle(method, path)
}

// GET is a shortcut for router.Handle(http.MethodGet, path, handle)
func (s *Server) GET(path string, handler HandlerFunc) {
	if s.started.Load().(bool) {
		return
	}

	s.router.GET(path, handler)
}

// HEAD is a shortcut for router.Handle(http.MethodHead, path, handle)
func (s *Server) HEAD(path string, handler HandlerFunc) {
	if s.started.Load().(bool) {
		return
	}

	s.router.HEAD(path, handler)
}

// OPTIONS is a shortcut for router.Handle(http.MethodOptions, path, handle)
func (s *Server) OPTIONS(path string, handler HandlerFunc) {
	if s.started.Load().(bool) {
		return
	}

	s.router.OPTIONS(path, handler)
}

// POST is a shortcut for router.Handle(http.MethodPost, path, handle)
func (s *Server) POST(path string, handler HandlerFunc) {
	if s.started.Load().(bool) {
		return
	}

	s.router.POST(path, handler)
}

// PUT is a shortcut for router.Handle(http.MethodPut, path, handle)
func (s *Server) PUT(path string, handler HandlerFunc) {
	if s.started.Load().(bool) {
		return
	}

	s.router.PUT(path, handler)
}

// PATCH is a shortcut for router.Handle(http.MethodPatch, path, handle)
func (s *Server) PATCH(path string, handler HandlerFunc) {
	if s.started.Load().(bool) {
		return
	}

	s.router.PATCH(path, handler)
}

// DELETE is a shortcut for router.Handle(http.MethodDelete, path, handle)
func (s *Server) DELETE(path string, handler HandlerFunc) {
	if s.started.Load().(bool) {
		return
	}

	s.router.DELETE(path, handler)
}

// Handle adds a route to be handled
func (s *Server) Handle(method, path string, handler HandlerFunc) {
	if s.started.Load().(bool) {
		return
	}

	s.router.Handle(method, path, handler)
}

// AddGroup adds a RouteGroup to be handled
func (s *Server) AddGroup(group *RouteGroup) {
	if s.started.Load().(bool) {
		return
	}

	s.router.AddGroup(group)
}

// HandleHTTP allows vk to handle a standard http.HandlerFunc
func (s *Server) HandleHTTP(method, path string, handler http.HandlerFunc) {
	if s.started.Load().(bool) {
		return
	}

	s.router.hrouter.Handle(method, path, func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		handler(w, r)
	})
}

func createGoServer(options *Options, handler http.Handler) *http.Server {
	if useHTTP := options.ShouldUseHTTP(); useHTTP {
		return goHTTPServerWithPort(options, handler)
	}

	return goTLSServerWithDomain(options, handler)
}

func goTLSServerWithDomain(options *Options, handler http.Handler) *http.Server {
	if options.TLSConfig != nil {
		options.Logger.Info("configured for HTTPS with custom configuration")
	} else if options.Domain != "" {
		options.Logger.Info("configured for HTTPS using domain", options.Domain)
	}

	tlsConfig := options.TLSConfig

	if tlsConfig == nil {
		m := &autocert.Manager{
			Cache:      autocert.DirCache("~/.autocert"),
			Prompt:     autocert.AcceptTOS,
			HostPolicy: autocert.HostWhitelist(options.Domain),
		}

		addr := fmt.Sprintf(":%d", options.HTTPPort)
		if options.HTTPPort == 0 {
			addr = ":8080"
		}

		options.Logger.Info("serving TLS challenges on", addr)

		go http.ListenAndServe(addr, m.HTTPHandler(nil))

		tlsConfig = &tls.Config{GetCertificate: m.GetCertificate}
	}

	addr := fmt.Sprintf(":%d", options.TLSPort)
	if options.TLSPort == 0 {
		addr = ":443"
	}

	s := &http.Server{
		Addr:      addr,
		TLSConfig: tlsConfig,
		Handler:   handler,
	}

	return s
}

func goHTTPServerWithPort(options *Options, handler http.Handler) *http.Server {
	options.Logger.Warn("configured to use HTTP with no TLS")

	s := &http.Server{
		Addr:    fmt.Sprintf(":%d", options.HTTPPort),
		Handler: handler,
	}

	return s
}
