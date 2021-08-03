package grav

import (
	"github.com/pkg/errors"
	"github.com/suborbital/vektor/vlog"
)

// TransportMsgTypeHandshake and others represent internal Transport message types used for handshakes and metadata transfer
const (
	TransportMsgTypeHandshake = 1
	TransportMsgTypeUser      = 2
)

// ErrConnectionClosed and others are transport and connection related errors
var (
	ErrConnectionClosed    = errors.New("connection was closed")
	ErrNodeUUIDMismatch    = errors.New("handshake UUID did not match node UUID")
	ErrNotBridgeTransport  = errors.New("transport is not a bridge")
	ErrBridgeOnlyTransport = errors.New("transport only supports bridge connection")
)

var (
	TransportTypeMesh   = TransportType("transport.mesh")
	TransportTypeBridge = TransportType("transport.bridge")
)

type (
	// ReceiveFunc is a function that allows passing along a received message
	ReceiveFunc func(msg Message)
	// ConnectFunc is a function that provides a new Connection
	ConnectFunc func(Connection)
	// FindFunc allows a Transport to query Grav for an active connection for the given UUID
	FindFunc func(uuid string) (Connection, bool)
	// TransportType defines the type of Transport (mesh or bridge)
	TransportType string
)

// TransportOpts is a set of options for transports
type TransportOpts struct {
	NodeUUID string
	Port     string
	URI      string
	Logger   *vlog.Logger
	Custom   interface{}
}

// Transport represents a Grav transport plugin
type Transport interface {
	// Type returns the transport's type (mesh or bridge)
	Type() TransportType
	// Setup is a transport-specific function that allows bootstrapping
	// Setup can block forever if needed; for example if a webserver is bring run
	Setup(opts *TransportOpts, connFunc ConnectFunc, findFunc FindFunc) error
	// CreateConnection connects to an endpoint and returns the Connection
	CreateConnection(endpoint string) (Connection, error)
	// ConnectBridgeTopic connects to a topic and returns a TopicConnection
	ConnectBridgeTopic(topic string) (TopicConnection, error)
}

// Connection represents a connection to another node
type Connection interface {
	// Called when the connection handshake is complete and the connection can actively start exchanging messages
	Start(recvFunc ReceiveFunc)
	// Send a message from the local instance to the connected node
	Send(msg Message) error
	// CanReplace returns true if the connection can be replaced (i.e. is not a persistent connection like a websocket)
	CanReplace() bool
	// Initiate a handshake for an outgoing connection and return the remote Ack
	DoOutgoingHandshake(handshake *TransportHandshake) (*TransportHandshakeAck, error)
	// Wait for an incoming handshake and return the provided Ack to the remote connection
	DoIncomingHandshake(handshakeAck *TransportHandshakeAck) (*TransportHandshake, error)
	// Close requests that the Connection close itself
	Close()
}

// TopicConnection is a connection to something via a bridge such as a topic
type TopicConnection interface {
	// Called when the connection can actively start exchanging messages
	Start(pod *Pod)
	// Close requests that the Connection close itself
	Close()
}

// TransportHandshake represents a handshake sent to a node that you're trying to connect to
type TransportHandshake struct {
	UUID string `json:"uuid"`
}

// TransportHandshakeAck represents a handshake response
type TransportHandshakeAck struct {
	UUID string `json:"uuid"`
}
