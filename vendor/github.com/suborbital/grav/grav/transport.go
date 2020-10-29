package grav

// ConnectFunc represents a function that returns a pod conntected to Grav
type ConnectFunc func() *Pod

// TransportOpts is a set of options for transports
type TransportOpts struct {
	Port   int
	Custom interface{}
}

// Transport represents a Grav transport plugin
type Transport interface {
	// Serve is a transport-specific function that exposes a connection point
	Serve(*TransportOpts, *Pod) error
	// ConnectEndpoint indicates to the Transport that a connection to a remote endpoint is needed
	ConnectEndpoint(string, ConnectFunc) error
}

// DefaultTransportOpts returns the default Grav Transport options
func DefaultTransportOpts() *TransportOpts {
	to := &TransportOpts{
		Port: 8080,
	}

	return to
}
