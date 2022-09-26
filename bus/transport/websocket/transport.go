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

	"github.com/suborbital/e2core/bus/bus"
	"github.com/suborbital/vektor/vlog"
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
	log  *vlog.Logger

	connectionFunc bus.ConnectFunc
}

// Conn implements transport.Connection and represents a websocket connection
type Conn struct {
	nodeUUID string
	log      *vlog.Logger

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
	t.log = opts.Logger
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
		if t.connectionFunc == nil {
			t.log.ErrorString("[transport-websocket] incoming connection received, but no connFunc configured")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.log.Error(errors.Wrap(err, "[transport-websocket] failed to upgrade connection"))
			return
		}

		t.log.Debug("[transport-websocket] upgraded connection:", r.URL.String())

		conn := &Conn{
			conn: c,
			log:  t.log,
		}

		t.connectionFunc(conn)
	}
}

// SendMsg sends a message to the connection
func (c *Conn) SendMsg(msg bus.Message) error {
	msgBytes, err := msg.Marshal()
	if err != nil {
		return errors.Wrap(err, "[transport-websocket] failed to Marshal message")
	}

	c.log.Debug("[transport-websocket] sending message", msg.UUID(), "to connection", c.nodeUUID)

	if err := c.WriteMessage(websocket.BinaryMessage, msgBytes); err != nil {
		if errors.Is(err, websocket.ErrCloseSent) {
			return bus.ErrConnectionClosed
		} else if err == bus.ErrNodeWithdrawn {
			return err
		}

		return errors.Wrap(err, "[transport-websocket] failed to WriteMessage")
	}

	c.log.Debug("[transport-websocket] sent message", msg.UUID(), "to connection", c.nodeUUID)

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
		c.log.Debug(errors.Wrap(err, "[transport-websocket] failed to MsgFromBytes, falling back to raw data").Error())

		msg = bus.NewMsg(MsgTypeWebsocketMessage, message)
	}

	c.log.Debug("[transport-websocket] received message", msg.UUID(), "via", c.nodeUUID)

	return msg, nil, nil
}

// OutgoingHandshake performs a connection handshake and returns the UUID of the node that we're connected to
// so that it can be validated against the UUID that was provided in discovery (or if none was provided)
func (c *Conn) OutgoingHandshake(handshake *bus.TransportHandshake) (*bus.TransportHandshakeAck, error) {
	handshakeJSON, err := json.Marshal(handshake)
	if err != nil {
		return nil, errors.Wrap(err, "failed to Marshal handshake JSON")
	}

	c.log.Debug("[transport-websocket] sending handshake")

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

	c.log.Debug("[transport-websocket] recieved handshake ack")

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

	c.log.Debug("[transport-websocket] recieved handshake")

	handshake := &bus.TransportHandshake{}
	if err := json.Unmarshal(message, handshake); err != nil {
		return errors.Wrap(err, "failed to Unmarshal handshake")
	}

	ack := handshakeCallback(handshake)

	ackJSON, err := json.Marshal(ack)
	if err != nil {
		return errors.Wrap(err, "failed to Marshal handshake ack JSON")
	}

	c.log.Debug("[transport-websocket] sending handshake ack")

	if err := c.WriteMessage(websocket.BinaryMessage, ackJSON); err != nil {
		return errors.Wrap(err, "failed to WriteMessage handshake ack")
	}

	c.log.Debug("[transport-websocket] sent handshake ack")

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
	c.log.Debug("[transport-websocket] connection for", c.nodeUUID, "is closing")

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
