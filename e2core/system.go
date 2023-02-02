package e2core

import (
	"github.com/suborbital/e2core/e2core/backend"
	"github.com/suborbital/e2core/e2core/server"
)

// System describes a E2Core system, which is comprised of a server and a backend
type System struct {
	Server  *server.Server
	Backend backend.Backend
}

// NewSystem creates a new System with the provided server and backend
func NewSystem(server *server.Server, backend backend.Backend) *System {
	s := &System{
		Server:  server,
		Backend: backend,
	}

	return s
}

// StartAll starts the Server and the backend.Backend, if they are configured
func (s *System) StartAll() {
	if s.Server != nil {
		s.signaler.Start(s.Server.Start)
	}

	if s.Backend != nil {
		s.signaler.Start(s.Backend.Start)
	}
}

// StartServer starts the Server if it is configured
func (s *System) StartServer() {
	if s.Server != nil {
		s.signaler.Start(s.Server.Start)
	}
}

// StartBackend starts the backend.Backend if it is configured
func (s *System) StartBackend() {
	if s.Backend != nil {
		s.signaler.Start(s.Backend.Start)
	}
}

func (s *System) ShutdownWait() error {
	return s.signaler.Wait(shutdownWaitTime)
}
