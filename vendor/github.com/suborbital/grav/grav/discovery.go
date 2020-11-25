package grav

import "github.com/suborbital/vektor/vlog"

// Discovery represents a discovery plugin
type Discovery interface {
	Start(*DiscoveryOpts, Transport, ConnectFunc) error
}

// DiscoveryOpts is a set of options for transports
type DiscoveryOpts struct {
	NodeUUID      string
	TransportPort string
	Logger        *vlog.Logger
	Custom        interface{}
}
