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
	"github.com/suborbital/grav/grav"
	"github.com/suborbital/vektor/vlog"
)

var upgrader = websocket.Upgrader{}

// Transport is a transport that connects Grav nodes via standard websockets
type Transport struct {
	opts *grav.TransportOpts
	log  *vlog.Logger

	connectionFunc func(grav.Connection)
}

// Conn implements transport.Connection and represents a websocket connection
type Conn struct {
	nodeUUID string
	log      *vlog.Logger

	conn  *websocket.Conn
	cLock sync.Mutex

	recvFunc grav.ReceiveFunc
}

// New creates a new websocket transport
func New() *Transport {
	t := &Transport{}

	return t
}

// Type returns the transport's type
func (t *Transport) Type() grav.TransportType {
	return grav.TransportTypeMesh
}

// Setup sets up the transport
func (t *Transport) Setup(opts *grav.TransportOpts, connFunc grav.ConnectFunc, findFunc grav.FindFunc) error {
	// independent serving is not yet implemented, use the HTTP handler

	t.opts = opts
	t.log = opts.Logger
	t.connectionFunc = connFunc

	return nil
}

// CreateConnection adds a websocket endpoint to emit messages to
func (t *Transport) CreateConnection(endpoint string) (grav.Connection, error) {
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
		log:   t.log,
		conn:  c,
		cLock: sync.Mutex{},
	}

	return conn, nil
}

// ConnectBridgeTopic connects to a topic if the transport is a bridge
func (t *Transport) ConnectBridgeTopic(topic string) (grav.TopicConnection, error) {
	return nil, grav.ErrNotBridgeTransport
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

// Start begins the receiving of messages
func (c *Conn) Start(recvFunc grav.ReceiveFunc) {
	c.recvFunc = recvFunc

	c.conn.SetCloseHandler(func(code int, text string) error {
		c.log.Warn(fmt.Sprintf("[transport-websocket] connection closing with code: %d", code))
		return nil
	})

	go func() {
		for {
			_, message, err := c.conn.ReadMessage()
			if err != nil {
				c.log.Error(errors.Wrap(err, "[transport-websocket] failed to ReadMessage, terminating connection"))
				break
			}

			c.log.Debug("[transport-websocket] recieved message via", c.nodeUUID)

			msg, err := grav.MsgFromBytes(message)
			if err != nil {
				c.log.Error(errors.Wrap(err, "[transport-websocket] failed to MsgFromBytes"))
				continue
			}

			// send to the Grav instance
			c.recvFunc(msg)
		}
	}()
}

// Send sends a message to the connection
func (c *Conn) Send(msg grav.Message) error {
	msgBytes, err := msg.Marshal()
	if err != nil {
		// not exactly sure what to do here (we don't want this going into the dead letter queue)
		c.log.Error(errors.Wrap(err, "[transport-websocket] failed to Marshal message"))
		return nil
	}

	c.log.Debug("[transport-websocket] sending message to connection", c.nodeUUID)

	if err := c.WriteMessage(grav.TransportMsgTypeUser, msgBytes); err != nil {
		if errors.Is(err, websocket.ErrCloseSent) {
			return grav.ErrConnectionClosed
		}

		return errors.Wrap(err, "[transport-websocket] failed to WriteMessage")
	}

	c.log.Debug("[transport-websocket] sent message to connection", c.nodeUUID)

	return nil
}

// CanReplace returns true if the connection can be replaced
func (c *Conn) CanReplace() bool {
	return false
}

// DoOutgoingHandshake performs a connection handshake and returns the UUID of the node that we're connected to
// so that it can be validated against the UUID that was provided in discovery (or if none was provided)
func (c *Conn) DoOutgoingHandshake(handshake *grav.TransportHandshake) (*grav.TransportHandshakeAck, error) {
	handshakeJSON, err := json.Marshal(handshake)
	if err != nil {
		return nil, errors.Wrap(err, "failed to Marshal handshake JSON")
	}

	c.log.Debug("[transport-websocket] sending handshake")

	if err := c.WriteMessage(grav.TransportMsgTypeHandshake, handshakeJSON); err != nil {
		return nil, errors.Wrap(err, "failed to WriteMessage handshake")
	}

	mt, message, err := c.conn.ReadMessage()
	if err != nil {
		return nil, errors.Wrap(err, "failed to ReadMessage for handshake ack, terminating connection")
	}

	if mt != grav.TransportMsgTypeHandshake {
		return nil, errors.New("first message recieved was not handshake ack")
	}

	c.log.Debug("[transport-websocket] recieved handshake ack")

	ack := grav.TransportHandshakeAck{}
	if err := json.Unmarshal(message, &ack); err != nil {
		return nil, errors.Wrap(err, "failed to Unmarshal handshake ack")
	}

	c.nodeUUID = ack.UUID

	return &ack, nil
}

// DoIncomingHandshake performs a connection handshake and returns the UUID of the node that we're connected to
// so that it can be validated against the UUID that was provided in discovery (or if none was provided)
func (c *Conn) DoIncomingHandshake(handshakeAck *grav.TransportHandshakeAck) (*grav.TransportHandshake, error) {
	mt, message, err := c.conn.ReadMessage()
	if err != nil {
		return nil, errors.Wrap(err, "failed to ReadMessage for handshake, terminating connection")
	}

	if mt != grav.TransportMsgTypeHandshake {
		return nil, errors.New("first message recieved was not handshake")
	}

	c.log.Debug("[transport-websocket] recieved handshake")

	handshake := grav.TransportHandshake{}
	if err := json.Unmarshal(message, &handshake); err != nil {
		return nil, errors.Wrap(err, "failed to Unmarshal handshake")
	}

	ackJSON, err := json.Marshal(handshakeAck)
	if err != nil {
		return nil, errors.Wrap(err, "failed to Marshal handshake JSON")
	}

	c.log.Debug("[transport-websocket] sending handshake ack")

	if err := c.WriteMessage(grav.TransportMsgTypeHandshake, ackJSON); err != nil {
		return nil, errors.Wrap(err, "failed to WriteMessage handshake ack")
	}

	c.log.Debug("[transport-websocket] sent handshake ack")

	c.nodeUUID = handshake.UUID

	return &handshake, nil
}

// Close closes the underlying connection
func (c *Conn) Close() {
	c.log.Debug("[transport-websocket] connection for", c.nodeUUID, "is closing")
	c.conn.Close()
}

// WriteMessage is a concurrent-safe wrapper around the websocket WriteMessage
func (c *Conn) WriteMessage(messageType int, data []byte) error {
	c.cLock.Lock()
	defer c.cLock.Unlock()

	return c.conn.WriteMessage(messageType, data)
}
