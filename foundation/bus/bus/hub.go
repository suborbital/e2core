package bus

import (
	"context"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/suborbital/e2core/foundation/bus/bus/tunnel"
	"github.com/suborbital/e2core/foundation/bus/bus/withdraw"
	"github.com/suborbital/e2core/foundation/tracing"
)

const tunnelRetryCount = 32

// hub is responsible for coordinating the transport and discovery plugins
type hub struct {
	nodeUUID    string
	belongsTo   string
	interests   []string
	mesh        MeshTransport
	bridge      BridgeTransport
	discovery   Discovery
	log         zerolog.Logger
	pod         *Pod
	connectFunc func() *Pod

	meshConnections   map[string]*connectionHandler
	bridgeConnections map[string]BridgeConnection

	capabilityBalancers map[string]*tunnel.Balancer

	lock sync.RWMutex
}

func initHub(nodeUUID string, options *Options, connectFunc func() *Pod) *hub {
	h := &hub{
		nodeUUID:            nodeUUID,
		belongsTo:           options.BelongsTo,
		interests:           options.Interests,
		mesh:                options.MeshTransport,
		bridge:              options.BridgeTransport,
		discovery:           options.Discovery,
		log:                 options.Logger.With().Str("module", "hub").Logger(),
		pod:                 connectFunc(),
		connectFunc:         connectFunc,
		meshConnections:     map[string]*connectionHandler{},
		bridgeConnections:   map[string]BridgeConnection{},
		capabilityBalancers: map[string]*tunnel.Balancer{},
		lock:                sync.RWMutex{},
	}

	// start mesh transport, then discovery if each have been configured (can have transport but no discovery)
	if h.mesh != nil {
		transportOpts := &MeshOptions{
			NodeUUID: nodeUUID,
			Port:     options.Port,
			URI:      options.URI,
			Logger:   options.Logger,
		}

		go func() {
			if err := h.mesh.Setup(transportOpts, h.handleIncomingConnection); err != nil {
				h.log.Err(err).Str("function", "initHub").Msg("failed to Setup transport")
			}

			// send all messages to all mesh connections
			h.pod.On(h.messageHandler)

			// scan forever to remove failed connections
			h.scanFailedMeshConnections()
		}()

		if h.discovery != nil {
			discoveryOpts := &DiscoveryOpts{
				NodeUUID:      nodeUUID,
				TransportPort: transportOpts.Port,
				TransportURI:  transportOpts.URI,
				Logger:        options.Logger.With().Str("module", "discovery").Logger().Level(zerolog.InfoLevel),
			}

			go func() {
				if err := h.discovery.Start(discoveryOpts, h.discoveryHandler()); err != nil {
					options.Logger.Err(err).Str("function", "initHub").Msg("failed to Start discovery")
				}
			}()
		}
	}

	if h.bridge != nil {
		transportOpts := &BridgeOptions{
			NodeUUID: nodeUUID,
			Logger:   options.Logger,
		}

		go func() {
			if err := h.bridge.Setup(transportOpts); err != nil {

				h.log.Err(err).Str("function", "initHub").Msg("failed to Setup bridge transport")
			}
		}()
	}

	return h
}

// messageHandler takes each message coming from the bus and sends it to currently active mesh connections
func (h *hub) messageHandler(msg Message) error {
	ctx, span := tracing.Tracer.Start(msg.Context(), "hub messagehandler")
	defer span.End()

	msg.SetContext(ctx)

	h.lock.RLock()
	defer h.lock.RUnlock()

	ll := h.log.With().Str("requestID", msg.ParentID()).Str("method", "hub.messageHandler").Logger()

	ll.Info().Msg("sending the message to all meshconnections")

	// send the message to each. withdrawn connections will result in a no-op
	for uuid := range h.meshConnections {
		ll.Info().
			Str("meshconnection-uuid", uuid).
			Msg("sending the message to the handler at this uuid")

		handler := h.meshConnections[uuid]
		err := handler.Send(ctx, msg)
		if err != nil {
			ll.Err(err).Str("meshconnection-uuid", uuid).
				Msg("send returned an error")
		}
	}

	return nil
}

func (h *hub) discoveryHandler() func(endpoint string, uuid string) {
	ll := h.log.With().Str("method", "discoveryHandler").Logger()

	return func(endpoint string, uuid string) {
		if uuid == h.nodeUUID {
			ll.Debug().Str("uuid", uuid).Msg("discovered self, discarding")
			return
		}

		// this reduces the number of extraneous outgoing handshakes that get attempted.
		if h.connectionExists(uuid) {
			ll.Debug().Str("uuid", uuid).Msg("encountered duplicate connection,discarding")
			return
		}

		if err := h.connectEndpoint(endpoint, uuid); err != nil {
			ll.Err(err).Str("uuid", uuid).Msg("failed to connectEndpoint for discovered peer")
		}
	}
}

// connectEndpoint creates a new outgoing connection
func (h *hub) connectEndpoint(endpoint, uuid string) error {
	if h.mesh == nil {
		return ErrTransportNotConfigured
	}

	h.log.Debug().Str("method", "connectEndpoint").Str("endpoint", endpoint).Msg("connecting to endpoint")

	conn, err := h.mesh.Connect(endpoint)
	if err != nil {
		return errors.Wrap(err, "[hub.connectEndpoint] failed to transport.CreateConnection")
	}

	h.setupOutgoingConnection(conn, uuid)

	return nil
}

// connectBridgeTopic creates a new outgoing connection
func (h *hub) connectBridgeTopic(topic string) error {
	if h.bridge == nil {
		return ErrTransportNotConfigured
	}

	h.log.Debug().Str("method", "connectBridgeTopic").Str("topic", topic).Msg("connecting to topic")

	conn, err := h.bridge.ConnectTopic(topic)
	if err != nil {
		return errors.Wrap(err, "[hub.connectBridgeTopic] failed to transport.CreateConnection")
	}

	h.addTopicConnection(conn, topic)

	return nil
}

func (h *hub) setupOutgoingConnection(connection Connection, uuid string) {
	ll := h.log.With().Str("method", "setupOutgoingConnection").Logger()

	handshake := &TransportHandshake{h.nodeUUID, h.belongsTo, h.interests}

	ack, err := connection.OutgoingHandshake(handshake)
	if err != nil {
		ll.Err(err).Msg("connection.OutgoingHandshake")
		connection.Close()
		return
	}

	if !ack.Accept {
		ll.Debug().Msg("connection handshake was not accepted, terminating connection")
		connection.Close()

		return
	} else if uuid == "" {
		if ack.UUID == "" {
			ll.Error().Msg("connection handshake returned empty UUID, terminating connection")
			connection.Close()

			return
		}

		uuid = ack.UUID
	} else if ack.UUID != uuid {
		ll.Error().Str("uuid", uuid).Str("ack.UUID", ack.UUID).Msg("connection handshake Ack did not match Discovery Ack, terminating connection")
		connection.Close()

		return
	}

	h.setupNewConnection(connection, uuid, ack.BelongsTo, ack.Interests)
}

func (h *hub) handleIncomingConnection(connection Connection) {
	ll := h.log.With().Str("method", "handleIncomingConnection").Logger()

	var handshake *TransportHandshake
	var ack *TransportHandshakeAck

	callback := func(incomingHandshake *TransportHandshake) *TransportHandshakeAck {
		handshake = incomingHandshake

		ack = &TransportHandshakeAck{
			Accept: true,
			UUID:   h.nodeUUID,
		}

		if incomingHandshake.BelongsTo != h.belongsTo && incomingHandshake.BelongsTo != "*" {
			ack.Accept = false
		} else {
			ack.BelongsTo = h.belongsTo
			ack.Interests = h.interests
		}

		return ack
	}

	if err := connection.IncomingHandshake(callback); err != nil {
		ll.Err(err).Msg("connection.DoIncomingHandshake")
		connection.Close()

		return
	}

	if handshake == nil || handshake.UUID == "" {
		ll.Error().Msg("connection handshake returned empty UUID, terminating connection")
		connection.Close()

		return
	}

	if !ack.Accept {
		ll.Debug().Str("belongsTo", handshake.BelongsTo).Msg("rejecting connection with incompatible BelongsTo")
		connection.Close()

		return
	}

	h.setupNewConnection(connection, handshake.UUID, handshake.BelongsTo, handshake.Interests)
}

func (h *hub) setupNewConnection(connection Connection, uuid, belongsTo string, interests []string) {
	if h.connectionExists(uuid) {
		connection.Close()
		h.log.Debug().Str("method", "setupNewConnection").Msg("encountered duplicate connection, discarding")
	} else {
		h.addConnection(connection, uuid, belongsTo, interests)
	}
}

func (h *hub) addConnection(connection Connection, uuid, belongsTo string, interests []string) {
	h.lock.Lock()
	defer h.lock.Unlock()

	h.log.Debug().Str("method", "addConnection").Str("uuid", uuid).Msg("adding connection for")

	signaler := withdraw.NewSignaler()

	handler := &connectionHandler{
		UUID:      uuid,
		Conn:      connection,
		Pod:       h.pod,
		Signaler:  signaler,
		ErrChan:   make(chan error, 1),
		BelongsTo: belongsTo,
		Interests: interests,
		Log:       h.log,
	}

	handler.Start()

	h.meshConnections[uuid] = handler

	for _, c := range interests {
		if _, exists := h.capabilityBalancers[c]; !exists {
			h.capabilityBalancers[c] = tunnel.NewBalancer()
		}

		h.capabilityBalancers[c].Add(uuid)
	}
}

func (h *hub) addTopicConnection(connection BridgeConnection, topic string) {
	h.lock.Lock()
	defer h.lock.Unlock()

	h.log.Debug().Str("method", "addTopicConnection").Str("topic", topic).Msg("adding bridge connection for topic")

	connection.Start(h.connectFunc())

	h.bridgeConnections[topic] = connection
}

// removeMeshConnection removes an entry from the known list of connections. This is called from
// scanFailedMeshConnections which will scan for failed connections, close them, populate the list, and for each element
// of the list call this method.
//
// That means actually closing the connections is done in scanFailedMeshConnections, and we don't need to do it here.
func (h *hub) removeMeshConnection(uuid string) {
	h.lock.Lock()
	defer h.lock.Unlock()

	h.log.Debug().Str("method", "removeMeshConnection").Str("uuid", uuid).Msg("removing connection")

	for _, balancer := range h.capabilityBalancers {
		balancer.Remove(uuid)
	}

	delete(h.meshConnections, uuid)
}

func (h *hub) connectionExists(uuid string) bool {
	h.lock.RLock()
	defer h.lock.RUnlock()

	conn, exists := h.meshConnections[uuid]
	if exists && conn.Conn != nil {
		return true
	}

	return false
}

// scanFailedMeshConnections should be run on a goroutine to constantly
// check for failed connections and clean them up
func (h *hub) scanFailedMeshConnections() {
	ll := h.log.With().Str("method", "scanFailedMeshConnections").Logger()

	ll.Info().Msg("starting the loop to scan for failed mesh connections")

	ticker := time.NewTicker(time.Second)

	for {
		select {
		case <-ticker.C:
			// ll.Info().Msg("starting loop")
			// we don't want to edit the `meshConnections` map while in the loop, so do it after
			toRemove := make([]string, 0)

			// for each connection, check if it has errored or if its peer has withdrawn,
			// and in either case close it and remove it from circulation
			for _, conn := range h.meshConnections {
				select {
				case <-conn.ErrChan:
					if err := conn.Close(); err != nil {
						ll.Err(err).Str("connUUID", conn.UUID).Msg("failed to Close connection")
					}

					ll.Warn().Str("conn-uuid", conn.UUID).Msg("adding this to removal")
					toRemove = append(toRemove, conn.UUID)
				default:
					// ll.Info().Str("conn-uuid", conn.UUID).Msg("no error came in, doing default")
					if conn.Signaler.PeerWithdrawn() {
						if err := conn.Close(); err != nil {
							ll.Err(err).Str("connUUID", conn.UUID).Msg(
								"failed to Close connection")
						}

						ll.Warn().Str("conn-uuid", conn.UUID).Msg("peer has withdrawn, so removing it from here")

						toRemove = append(toRemove, conn.UUID)
					}
				}
			}

			for _, uuid := range toRemove {
				ll.Info().Str("conn-uuid", uuid).Msg("removing mesh connection")
				h.removeMeshConnection(uuid)
			}
		}
	}
}

func (h *hub) sendTunneledMessage(ctx context.Context, capability string, msg Message) error {
	ctx, span := tracing.Tracer.Start(ctx, "hub.sendtunneledmessage")
	defer span.End()

	ll := h.log.With().Str("method", "sendTunneledMessage").
		Str("requestID", msg.ParentID()).Logger()

	ll.Info().Str("capability", capability).Msg("sending a message with cap. Checking the hub's capabilityBalancers map. It seems to be a list of UUIDs for ... things? belonging to the same capability.")

	balancer, exists := h.capabilityBalancers[capability]
	if !exists {
		return ErrTunnelNotEstablished
	}

	ll.Info().Interface("balancer", balancer).Str("capability", capability).Msg("balancer for capability")

	ll.Info().Int("tunnel-retry-count", tunnelRetryCount).Msg("starting iteration to check whether we can send a message to someplace")

	handlerFactory := func(ctx context.Context) (*connectionHandler, error) {
		ctx, span := tracing.Tracer.Start(ctx, "handlerFactory")
		defer span.End()

		h.lock.RLock()
		defer h.lock.RUnlock()

		uuid := balancer.Next()
		if uuid == "" {
			span.AddEvent("balancer doesn't exit")
			return nil, ErrTunnelNotEstablished
		}

		handler, exists := h.meshConnections[uuid]
		if !exists {
			span.AddEvent("handler doesn't exist for uuid", trace.WithAttributes(
				attribute.String("uuid", uuid),
			))
			return nil, ErrTunnelNotEstablished
		}

		span.AddEvent("returning a handler for uuid", trace.WithAttributes(
			attribute.String("uuid", uuid),
		))
		return handler, nil
	}

	// iterate a reasonable number of times to find a connection that's not removed or dead
	for i := 0; i < tunnelRetryCount; i++ {
		// wrap this in a function to avoid any sloppy mutex issues
		handler, err := handlerFactory(ctx)
		if err != nil {
			continue
		}

		if handler.Conn != nil {
			if err := handler.Send(ctx, msg); err != nil {
				ll.Err(err).Msg("failed to SendMsg on tunneled connection, will remove")
				return errors.Wrap(err, "handler.Send died")
			} else {
				ll.Info().Str("handlerUUID", handler.UUID).Msg("tunneled to handler")
				return nil
			}
		}

		ll.Info().Msg("handler connection was nil")
	}

	return ErrTunnelNotEstablished
}

func (h *hub) withdraw() error {
	h.lock.Lock()
	defer h.lock.Unlock()

	// first, stop broadcsting to other nodes that we exist
	if h.discovery != nil {
		h.discovery.Stop()
	}

	doneChans := map[string]chan struct{}{}

	// indicate to each signaler that the withdraw should begin
	for uuid := range h.meshConnections {
		conn := h.meshConnections[uuid]

		doneChans[uuid] = conn.Signaler.Signal()
	}

	// the withdraw attempt will time out after 3 seconds
	timeoutChan := time.After(time.Second * 3)
	doneChan := make(chan struct{})

	go func() {
		count := len(h.meshConnections)

		// continually go through each connection and check if its withdraw is complete
		// until we've gotten the signal from every single one
		for {
			for uuid := range h.meshConnections {
				doneChan := doneChans[uuid]

				select {
				case <-doneChan:
					count--
				default:
					// continue
				}
			}

			if count == 0 {
				doneChan <- struct{}{}
				break
			}
		}
	}()

	// return when either the withdraw is complete or we timed out
	select {
	case <-doneChan:
		// cool, done
	case <-timeoutChan:
		return ErrWaitTimeout
	}

	return nil
}

func (h *hub) stop() error {
	ll := h.log.With().Str("method", "stop").Logger()

	ll.Info().Msg("stopping")

	var lastErr error
	for _, c := range h.meshConnections {
		if err := c.Conn.Close(); err != nil {
			lastErr = err
			ll.Err(err).Str("connectionUUID", c.UUID).Msg("failed to Close connection")
		}
	}

	return lastErr
}
