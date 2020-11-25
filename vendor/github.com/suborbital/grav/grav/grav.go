package grav

import (
	"github.com/pkg/errors"

	"github.com/google/uuid"
	"github.com/suborbital/vektor/vlog"
)

// ErrTransportNotConfigured represent package-level vars
var (
	ErrTransportNotConfigured = errors.New("transport plugin not configured")
)

// Grav represents a Grav message bus instance
type Grav struct {
	NodeUUID  string
	bus       *messageBus
	logger    *vlog.Logger
	transport Transport
	discovery Discovery
}

// New creates a new Grav with the provided options
func New(opts ...OptionsModifier) *Grav {
	nodeUUID := uuid.New().String()

	options := newOptionsWithModifiers(opts...)

	g := &Grav{
		NodeUUID:  nodeUUID,
		bus:       newMessageBus(),
		logger:    options.Logger,
		transport: options.Transport,
		discovery: options.Discovery,
	}

	// start transport, then discovery if each have been configured (can have transport but no discovery)
	if g.transport != nil {
		transportOpts := &TransportOpts{
			NodeUUID: nodeUUID,
			Port:     options.Port,
			Logger:   options.Logger,
		}

		go func() {
			if err := g.transport.Serve(transportOpts, g.Connect); err != nil {
				options.Logger.Error(errors.Wrap(err, "failed to Serve transport"))
			}

			if g.discovery != nil {
				discoveryOpts := &DiscoveryOpts{
					NodeUUID:      nodeUUID,
					TransportPort: transportOpts.Port,
					Logger:        options.Logger,
				}

				if err := g.discovery.Start(discoveryOpts, g.transport, g.Connect); err != nil {
					options.Logger.Error(errors.Wrap(err, "failed to Start discovery"))
				}
			}
		}()
	}

	return g
}

// Connect creates a new connection (pod) to the bus
func (g *Grav) Connect() *Pod {
	opts := &podOpts{WantsReplay: false}

	return g.connectWithOpts(opts)
}

// ConnectWithReplay creates a new connection (pod) to the bus
// and replays recent messages when the pod sets its onFunc
func (g *Grav) ConnectWithReplay() *Pod {
	opts := &podOpts{WantsReplay: true}

	return g.connectWithOpts(opts)
}

// ConnectEndpoint uses the configured transport to connect the bus to an external endpoint
func (g *Grav) ConnectEndpoint(endpoint string) error {
	if g.transport == nil {
		return ErrTransportNotConfigured
	}

	return g.transport.ConnectEndpoint(endpoint, g.Connect)
}

// ConnectEndpointWithReplay uses the configured transport to connect the bus to an external endpoint
// and replays recent messages to the endpoint when the pod registers its onFunc
func (g *Grav) ConnectEndpointWithReplay(endpoint string) error {
	if g.transport == nil {
		return ErrTransportNotConfigured
	}

	return g.transport.ConnectEndpoint(endpoint, g.ConnectWithReplay)
}

func (g *Grav) connectWithOpts(opts *podOpts) *Pod {
	pod := newPod(g.bus.busChan, opts)

	g.bus.addPod(pod)

	return pod
}
