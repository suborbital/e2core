package nats

import (
	"time"

	"github.com/nats-io/nats.go"
	"github.com/pkg/errors"
	"github.com/suborbital/grav/grav"
	"github.com/suborbital/vektor/vlog"
)

// Transport is a transport that connects Grav nodes via standard websockets
type Transport struct {
	opts *grav.TransportOpts
	log  *vlog.Logger

	serverConn *nats.Conn

	connectionFunc func(grav.Connection)
}

// Conn implements transport.TopicConnection and represents a subscribe/send pair for a NATS topic
type Conn struct {
	topic string
	log   *vlog.Logger
	pod   *grav.Pod

	sub   *nats.Subscription
	pubFn func(data []byte) error
}

// New creates a new websocket transport
func New(endpoint string) (*Transport, error) {
	t := &Transport{}

	nc, err := nats.Connect(endpoint)
	if err != nil {
		return nil, errors.Wrap(err, "failed to nats.Connect")
	}

	t.serverConn = nc

	return t, nil
}

// Type returns the transport's type
func (t *Transport) Type() grav.TransportType {
	return grav.TransportTypeBridge
}

// Setup sets up the transport
func (t *Transport) Setup(opts *grav.TransportOpts, connFunc grav.ConnectFunc, findFunc grav.FindFunc) error {
	t.opts = opts
	t.log = opts.Logger
	t.connectionFunc = connFunc

	return nil
}

// CreateConnection adds an endpoint to emit messages to
func (t *Transport) CreateConnection(endpoint string) (grav.Connection, error) {
	return nil, grav.ErrBridgeOnlyTransport
}

// ConnectBridgeTopic connects to a topic if the transport is a bridge
func (t *Transport) ConnectBridgeTopic(topic string) (grav.TopicConnection, error) {
	sub, err := t.serverConn.SubscribeSync(topic)
	if err != nil {
		return nil, errors.Wrap(err, "failed to SubscribeSync")
	}

	pubFn := func(data []byte) error {
		return t.serverConn.Publish(topic, data)
	}

	conn := &Conn{
		topic: topic,
		log:   t.log,
		sub:   sub,
		pubFn: pubFn,
	}

	return conn, nil
}

// Start begins the receiving of messages
func (c *Conn) Start(pod *grav.Pod) {
	c.pod = pod

	c.pod.OnType(c.topic, func(msg grav.Message) error {
		msgBytes, err := msg.Marshal()
		if err != nil {
			return errors.Wrap(err, "failed to Marshal message")
		}

		if err := c.pubFn(msgBytes); err != nil {
			return errors.Wrap(err, "failed to pubFn")
		}

		return nil
	})

	go func() {
		for {
			message, err := c.sub.NextMsg(time.Duration(time.Second * 60))
			if err != nil {
				if err == nats.ErrTimeout {
					c.log.Debug("[bridge-nats] NextMsg timeout")
					continue
				}

				c.log.Error(errors.Wrap(err, "[bridge-nats] failed to ReadMessage, terminating connection"))
				break
			}

			c.log.Debug("[bridge-nats] recieved message via", c.topic)

			msg, err := grav.MsgFromBytes(message.Data)
			if err != nil {
				c.log.Error(errors.Wrap(err, "[bridge-nats] failed to MsgFromBytes"))
				continue
			}

			// send to the Grav instance
			c.pod.Send(msg)
		}
	}()
}

// Close closes the underlying connection
func (c *Conn) Close() {
	c.log.Debug("[bridge-nats] connection for", c.topic, "is closing")
	if err := c.sub.Unsubscribe(); err != nil {
		c.log.Error(errors.Wrapf(err, "[bridge-nats] connection for %s failed to close", c.topic))
	}
}
