package grav

import "github.com/suborbital/vektor/vlog"

// DiscoveryFunc is a function that allows a plugin to report a newly discovered node
type DiscoveryFunc func(endpoint string, uuid string)

// Discovery represents a discovery plugin
type Discovery interface {
	// Start is called to start the Discovery plugin
	Start(*DiscoveryOpts, DiscoveryFunc) error
}

// DiscoveryOpts is a set of options for transports
type DiscoveryOpts struct {
	NodeUUID      string
	TransportPort string
	TransportURI  string
	Logger        *vlog.Logger
	Custom        interface{}
}
