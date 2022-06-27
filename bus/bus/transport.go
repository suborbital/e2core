package bus

import (
	"github.com/pkg/errors"
	"github.com/suborbital/vektor/vlog"
)

// ErrConnectionClosed and others are transport and connection related errors
var (
	ErrConnectionClosed  = errors.New("connection was closed")
	ErrNodeUUIDMismatch  = errors.New("handshake UUID did not match node UUID")
	ErrBelongsToMismatch = errors.New("new connection doesn't belongTo the same group or *")
	ErrNodeWithdrawn     = errors.New("node has withdrawn from the mesh")
)

type (
	// ReceiveFunc is a function that allows passing along a received message
	ReceiveFunc func(msg Message)
	// ConnectFunc is a function that provides a new Connection
	ConnectFunc func(Connection)
	// FindFunc allows a Transport to query Grav for an active connection for the given UUID
	FindFunc func(uuid string) (Connection, bool)
	// HandshakeCallback allows the hub to determine if a connection should be accepted
	HandshakeCallback func(*TransportHandshake) *TransportHandshakeAck
)

// Withdraw is a type used to indicate that a Withdraw is occuring
// Withdraws are sent / recieved in transport-specific ways (can vary)
// If Ack is true, it indicates the value is an Ack to a Withdraw message
type Withdraw struct {
	Ack bool
}

// MeshOptions is a set of options for mesh transports
type MeshOptions struct {
	NodeUUID string
	Port     string
	URI      string
	Logger   *vlog.Logger
	Custom   interface{}
}

// BridgeOptions is a set of options for mesh transports
type BridgeOptions struct {
	NodeUUID string
	Logger   *vlog.Logger
	Custom   interface{}
}

// MeshTransport represents a transport plugin for connecting to meshed peers
type MeshTransport interface {
	// Setup is a transport-specific function that allows bootstrapping
	// Setup can block forever if needed; for example if a webserver is bring run
	Setup(opts *MeshOptions, connFunc ConnectFunc) error
	// Connect connects to an endpoint and returns the Connection
	Connect(endpoint string) (Connection, error)
}

// BridgeTransport represents a transport plugin that connects to centralized brokers
type BridgeTransport interface {
	// Setup is a transport-specific function that allows bootstrapping
	Setup(opts *BridgeOptions) error
	// ConnectTopic connects to a topic and returns a BridgeConnection
	ConnectTopic(topic string) (BridgeConnection, error)
}

// Connection represents a connection to another node in the mesh
type Connection interface {
	// SendMsg a message from the local instance to the connected node
	SendMsg(msg Message) error
	// ReadMsg prompts the connection to read the next incoming message, and returns either a message or a withdraw (if recieved)
	ReadMsg() (Message, *Withdraw, error)
	// Initiate a handshake for an outgoing connection and return the remote Ack
	OutgoingHandshake(handshake *TransportHandshake) (*TransportHandshakeAck, error)
	// Wait for an incoming handshake and allow the hub to determine what ack to send using the HandshakeCallback
	IncomingHandshake(HandshakeCallback) error
	// SendWithdraw indicates the connection should send the transport-specific withdraw message
	SendWithdraw(*Withdraw) error
	// Close requests that the Connection close itself
	Close() error
}

// BridgeConnection is a connection to something via a bridge such as a topic
type BridgeConnection interface {
	// Called when the connection can actively start exchanging messages
	Start(pod *Pod)
	// Close requests that the Connection close itself
	Close()
}

// TransportHandshake represents a handshake sent to a node that you're trying to connect to
type TransportHandshake struct {
	UUID      string   `json:"uuid"`
	BelongsTo string   `json:"belongsTo"`
	Interests []string `json:"interests"`
}

// TransportHandshakeAck represents a handshake response
type TransportHandshakeAck struct {
	Accept    bool     `json:"accept"`
	UUID      string   `json:"uuid"`
	BelongsTo string   `json:"belongsTo"`
	Interests []string `json:"interests"`
}

// TransportWithdraw represents a message sent to a peer indicating a withdrawal from the mesh
type TransportWithdraw struct {
	UUID string `json:"uuid"`
}
