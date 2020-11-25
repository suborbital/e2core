package grav

import "github.com/suborbital/vektor/vlog"

// TransportMsgTypeHandshake and others represent internal Transport message types used for handshakes and metadata transfer
const (
	TransportMsgTypeHandshake = 1
	TransportMsgTypeUser      = 2
)

// ConnectFunc represents a function that returns a pod conntected to Grav
type ConnectFunc func() *Pod

// TransportOpts is a set of options for transports
type TransportOpts struct {
	NodeUUID string
	Port     string
	Logger   *vlog.Logger
	Custom   interface{}
}

// Transport represents a Grav transport plugin
type Transport interface {
	// Serve is a transport-specific function that exposes a connection point
	Serve(*TransportOpts, ConnectFunc) error
	// ConnectEndpoint indicates to the Transport that a connection to a remote endpoint is needed
	ConnectEndpoint(string, ConnectFunc) error
	// ConnectEndpointWithUUID connects to an endpoint with a known identifier
	ConnectEndpointWithUUID(string, string, ConnectFunc) error
}

// TransportHandshake represents a handshake sent to a node that you're trying to connect to
type TransportHandshake struct {
	UUID string `json:"uuid"`
}

// TransportHandshakeAck represents a handshake response
type TransportHandshakeAck struct {
	UUID string `json:"uuid"`
}
