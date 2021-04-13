package vk

import (
	"crypto/tls"
	"fmt"
	"net/http"

	"golang.org/x/crypto/acme/autocert"
)

const defaultEnvPrefix = "VK"

// Server represents a vektor API server
type Server struct {
	*Router
	server  *http.Server
	options *Options
}

// New creates a new vektor API server
func New(opts ...OptionsModifier) *Server {
	options := newOptsWithModifiers(opts...)

	router := routerWithOptions(options)

	server := createGoServer(options, router)

	s := &Server{
		Router:  router,
		server:  server,
		options: options,
	}

	return s
}

// Start starts the server listening
func (s *Server) Start() error {
	// mount the root set of routes before starting
	s.mountGroup(s.Router.rootGroup())

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

func createGoServer(options *Options, handler http.Handler) *http.Server {
	if useHTTP := options.ShouldUseHTTP(); useHTTP {
		return goHTTPServerWithPort(options, handler)
	}

	return goTLSServerWithDomain(options, handler)
}

func goTLSServerWithDomain(options *Options, handler http.Handler) *http.Server {
	if options.Domain != "" {
		options.Logger.Info("configured for HTTPS using domain", options.Domain)
	}

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

	s := &http.Server{
		Addr:      ":443",
		TLSConfig: &tls.Config{GetCertificate: m.GetCertificate},
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
