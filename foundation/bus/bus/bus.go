package bus

import (
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

// ErrTransportNotConfigured represent package-level vars
var (
	ErrTransportNotConfigured = errors.New("transport plugin not configured")
	ErrTunnelNotEstablished   = errors.New("tunnel cannot be established")
)

// Bus represents a Bus message bus instance
type Bus struct {
	NodeUUID  string
	BelongsTo string
	Interests []string
	bus       *messageBus
	logger    zerolog.Logger
	hub       *hub
}

// New creates a new Bus with the provided options
func New(opts ...OptionsModifier) *Bus {
	nodeUUID := uuid.New().String()

	options := newOptionsWithModifiers(opts...)

	b := &Bus{
		NodeUUID:  nodeUUID,
		BelongsTo: options.BelongsTo,
		Interests: options.Interests,
		bus:       newMessageBus(),
		logger:    options.Logger,
	}

	// the hub handles coordinating the transport and discovery plugins
	b.hub = initHub(nodeUUID, options, b.Connect)

	return b
}

// Connect creates a new connection (pod) to the bus
func (b *Bus) Connect() *Pod {
	opts := &podOpts{WantsReplay: false}

	return b.connectWithOpts(opts)
}

// ConnectWithReplay creates a new connection (pod) to the bus
// and replays recent messages when the pod sets its onFunc
func (b *Bus) ConnectWithReplay() *Pod {
	opts := &podOpts{WantsReplay: true}

	return b.connectWithOpts(opts)
}

// ConnectEndpoint uses the configured transport to connect the bus to an external endpoint
func (b *Bus) ConnectEndpoint(endpoint string) error {
	return b.hub.connectEndpoint(endpoint, "")
}

// ConnectBridgeTopic connects the Bus instance to a particular topic on the connected bridge
func (b *Bus) ConnectBridgeTopic(topic string) error {
	return b.hub.connectBridgeTopic(topic)
}

// Tunnel sends a message to a specific connection that has advertised it has the required capability.
// This bypasses the main Bus bus, which is why it isn't a method on Pod.
// Messages are load balanced between the connections that advertise the capability in question.
func (b *Bus) Tunnel(capability string, msg Message) error {
	return b.hub.sendTunneledMessage(capability, msg)
}

// Withdraw cancels discovery, sends withdraw messages to all peers,
// and returns when all peers have acknowledged the withdraw
func (b *Bus) Withdraw() error {
	return b.hub.withdraw()
}

// Stop stops Bus's meshing entirely, causing all connections to peers to close.
// It is recommended to call `Withdraw` first to give peers notice and stop receiving messages
func (b *Bus) Stop() error {
	return b.hub.stop()
}

func (b *Bus) connectWithOpts(opts *podOpts) *Pod {
	b.logger.Info().Msg("creating a new pod with the bus's Tunnel method. That one takes the hub on the bus, and calls the sendTunneledMessage")
	pod := newPod(b.bus.busChan, b.Tunnel, opts, b.logger.With().Str("component", "pod").Logger())

	b.bus.addPod(pod)

	return pod
}
