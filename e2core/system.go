package e2core

import (
	"time"

	"github.com/suborbital/e2core/server"
	"github.com/suborbital/e2core/signaler"
)

const (
	shutdownWaitTime = time.Second * 3
)

// System describes a DeltaV system, which is comprised of a server and a backend
type System struct {
	Server  *server.Server
	Backend Backend

	signaler *signaler.Signaler
}

// NewSystem creates a new System with the provided server and backend
func NewSystem(server *server.Server, backend Backend) *System {
	s := &System{
		Server:   server,
		Backend:  backend,
		signaler: signaler.Setup(),
	}

	return s
}

// StartAll starts the Server and the Backend, if they are configured
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

// StartBackend starts the Backend if it is configured
func (s *System) StartBackend() {
	if s.Backend != nil {
		s.signaler.Start(s.Backend.Start)
	}
}

func (s *System) ShutdownWait() error {
	return s.signaler.Wait(shutdownWaitTime)
}
