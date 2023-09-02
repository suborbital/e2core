package websocket

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/suborbital/e2core/foundation/bus/bus"
	"github.com/suborbital/e2core/foundation/tracing"
)

const (
	MsgTypeWebsocketMessage = "websocket.message"

	withdrawMessage    = "WITHDRAW"
	withdrawAckMessage = "WITHDRAW ACK"
)

var upgrader = websocket.Upgrader{}

// Transport is a transport that connects Grav nodes via standard websockets
type Transport struct {
	opts *bus.MeshOptions
	log  zerolog.Logger

	connectionFunc bus.ConnectFunc
}

// Conn implements transport.Connection and represents a websocket connection
type Conn struct {
	nodeUUID string
	log      zerolog.Logger

	conn *websocket.Conn
	lock sync.Mutex
}

// New creates a new websocket transport
func New() *Transport {
	t := &Transport{}

	return t
}

// Setup sets up the transport
func (t *Transport) Setup(opts *bus.MeshOptions, connFunc bus.ConnectFunc) error {
	// independent serving is not yet implemented, use the HTTP handler

	t.opts = opts
	t.log = opts.Logger.With().Str("transport", "websocket").Logger().Level(zerolog.InfoLevel)
	t.connectionFunc = connFunc

	return nil
}

// Connect adds a websocket endpoint to emit messages to
func (t *Transport) Connect(endpoint string) (bus.Connection, error) {
	if !strings.HasPrefix(endpoint, "ws") {
		endpoint = fmt.Sprintf("ws://%s", endpoint)
	}

	endpointURL, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}

	c, _, err := websocket.DefaultDialer.Dial(endpointURL.String(), nil)
	if err != nil {
		return nil, errors.Wrapf(err, "[transport-websocket] failed to Dial endpoint")
	}

	conn := &Conn{
		log:  t.log,
		conn: c,
		lock: sync.Mutex{},
	}

	return conn, nil
}

// HTTPHandlerFunc returns an http.HandlerFunc for incoming connections
func (t *Transport) HTTPHandlerFunc() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := tracing.Tracer.Start(r.Context(), "websocket.transport.httphanderfunc")
		defer span.End()

		r = r.Clone(ctx)

		if t.connectionFunc == nil {
			t.log.Error().Msg("incoming connection received, but no connFunc configured")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		t.log.Info().Msg("receiving a message I think")

		span.AddEvent("upgrading request to websocket connection")
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.log.Err(err).Msg("could not upgrade connection to websocket")
			return
		}

		t.log.Info().Str("connectionURL", r.URL.String()).Msg("upgraded connection")

		conn := &Conn{
			conn: c,
			log:  t.log,
		}

		t.log.Info().Interface("connectionfunc", t.connectionFunc).Msg("connection func is this, apparently, bus.Connect, again? request is in the conn.conn as an upgraded websocket connection")

		span.AddEvent("calling connection function")
		t.connectionFunc(conn)
	}
}

// SendMsg sends a message to the connection
func (c *Conn) SendMsg(msg bus.Message) error {
	ctx, span := tracing.Tracer.Start(msg.Context(), "conn.SendMsg", trace.WithAttributes(
		attribute.String("request ID", msg.ParentID()),
	))
	defer span.End()

	span.AddEvent("injecting the ctx into the message")
	fmt.Printf("\n\n!!!!\n\ninjecting context into message\n")
	otel.GetTextMapPropagator().Inject(ctx, msg)
	fmt.Printf("\n\n---\n\ndone injecting context into message\n")

	ll := c.log.With().Str("requestID", msg.ParentID()).
		Str("msg-uuid", msg.UUID()).
		Str("node-uuid", c.nodeUUID).Logger()

	ll.Info().Strs("traceinfo-keys", msg.Keys()).Msg("uh what")

	msgBytes, err := msg.Marshal()
	if err != nil {
		return errors.Wrap(err, "[transport-websocket] failed to Marshal message")
	}

	ll.Info().Str("messagebytes", string(msgBytes)).Msg("sending message to connection over binary")

	if err := c.WriteMessage(websocket.BinaryMessage, msgBytes); err != nil {
		if errors.Is(err, websocket.ErrCloseSent) {
			ll.Err(err).Msg("websocket error close sent bla bla")
			return bus.ErrConnectionClosed
		} else if err == bus.ErrNodeWithdrawn {
			ll.Err(err).Msg("node was withdrawn")
			return err
		}

		ll.Err(err).Msg("some super different error with connection")

		return errors.Wrap(err, "[transport-websocket] failed to WriteMessage")
	}

	ll.Info().Msg("sent message to connection")

	return nil
}

func (c *Conn) ReadMsg() (bus.Message, *bus.Withdraw, error) {
	msgType, message, err := c.conn.ReadMessage()
	if err != nil {
		return nil, nil, errors.Wrap(err, "[transport-websocket] failed to ReadMessage, closing")
	}

	if msgType == websocket.TextMessage {
		if string(message) == withdrawMessage {
			// let Grav know this message was a withdraw
			return nil, &bus.Withdraw{Ack: false}, nil
		} else if string(message) == withdrawAckMessage {
			// let Grav know the peer acknowledged our withdraw
			return nil, &bus.Withdraw{Ack: true}, nil
		}
	}

	msg, err := bus.MsgFromBytes(message)
	if err != nil {
		c.log.Err(err).Msg("failed to MsgFromBytes, falling back to raw data")

		msg = bus.NewMsg(MsgTypeWebsocketMessage, message)
	}

	c.log.Debug().
		Str("requestID", msg.ParentID()).
		Str("msgUUID", msg.UUID()).
		Str("nodeUUID", c.nodeUUID).
		Msg("received message from node")

	return msg, nil, nil
}

// OutgoingHandshake performs a connection handshake and returns the UUID of the node that we're connected to
// so that it can be validated against the UUID that was provided in discovery (or if none was provided)
func (c *Conn) OutgoingHandshake(handshake *bus.TransportHandshake) (*bus.TransportHandshakeAck, error) {
	handshakeJSON, err := json.Marshal(handshake)
	if err != nil {
		return nil, errors.Wrap(err, "failed to Marshal handshake JSON")
	}

	c.log.Debug().Msg("sending handshake")

	if err := c.WriteMessage(websocket.BinaryMessage, handshakeJSON); err != nil {
		return nil, errors.Wrap(err, "failed to WriteMessage handshake")
	}

	mt, message, err := c.conn.ReadMessage()
	if err != nil {
		return nil, errors.Wrap(err, "failed to ReadMessage for handshake ack, terminating connection")
	}

	if mt != websocket.BinaryMessage {
		return nil, errors.New("first message recieved was not handshake ack")
	}

	c.log.Debug().Msg("received handshake ack")

	ack := bus.TransportHandshakeAck{}
	if err := json.Unmarshal(message, &ack); err != nil {
		return nil, errors.Wrap(err, "failed to Unmarshal handshake ack")
	}

	c.nodeUUID = ack.UUID

	return &ack, nil
}

// IncomingHandshake performs a connection handshake and returns the UUID of the node that we're connected to
// so that it can be validated against the UUID that was provided in discovery (or if none was provided)
func (c *Conn) IncomingHandshake(handshakeCallback bus.HandshakeCallback) error {
	mt, message, err := c.conn.ReadMessage()
	if err != nil {
		return errors.Wrap(err, "failed to ReadMessage for handshake, terminating connection")
	}

	if mt != websocket.BinaryMessage {
		return errors.New("first message recieved was not handshake")
	}

	c.log.Debug().Msg("received handshake")

	handshake := &bus.TransportHandshake{}
	if err := json.Unmarshal(message, handshake); err != nil {
		return errors.Wrap(err, "failed to Unmarshal handshake")
	}

	ack := handshakeCallback(handshake)

	ackJSON, err := json.Marshal(ack)
	if err != nil {
		return errors.Wrap(err, "failed to Marshal handshake ack JSON")
	}

	c.log.Debug().Msg("sending handshake ack")

	if err := c.WriteMessage(websocket.BinaryMessage, ackJSON); err != nil {
		return errors.Wrap(err, "failed to WriteMessage handshake ack")
	}

	c.log.Debug().Msg("sent handshake ack")

	c.nodeUUID = handshake.UUID

	return nil
}

// SendWithdraw sends a withdraw message to the peer
func (c *Conn) SendWithdraw(withdraw *bus.Withdraw) error {
	message := withdrawMessage
	if withdraw.Ack {
		message = withdrawAckMessage
	}

	if err := c.WriteMessage(websocket.TextMessage, []byte(message)); err != nil {
		return errors.Wrap(err, "[transport-websocket] failed to WriteMessage for withdraw")
	}

	return nil
}

// Close closes the underlying connection
func (c *Conn) Close() error {
	c.log.Debug().Str("nodeUUID", c.nodeUUID).Msg("connection is closing")

	if err := c.conn.Close(); err != nil {
		return errors.Wrap(err, "[transport-websocket] failed to Close connection")
	}

	return nil
}

// WriteMessage is a concurrent-safe wrapper around the websocket WriteMessage
func (c *Conn) WriteMessage(messageType int, data []byte) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	return c.conn.WriteMessage(messageType, data)
}
