package nats

import (
	"time"

	"github.com/nats-io/nats.go"
	"github.com/pkg/errors"
	"github.com/suborbital/vektor/vlog"
	"github.com/suborbital/velocity/bus/bus"
)

// Transport is a transport that connects Grav nodes via NATS
type Transport struct {
	opts *bus.BridgeOptions
	log  *vlog.Logger

	serverConn *nats.Conn

	connectionFunc func(bus.Connection)
}

// Conn implements transport.TopicConnection and represents a subscribe/send pair for a NATS topic
type Conn struct {
	topic string
	log   *vlog.Logger
	pod   *bus.Pod

	sub   *nats.Subscription
	pubFn func(data []byte) error
}

// New creates a new NATS transport
func New(endpoint string) (*Transport, error) {
	t := &Transport{}

	nc, err := nats.Connect(endpoint)
	if err != nil {
		return nil, errors.Wrap(err, "failed to nats.Connect")
	}

	t.serverConn = nc

	return t, nil
}

// Setup sets up the transport
func (t *Transport) Setup(opts *bus.BridgeOptions) error {
	t.opts = opts
	t.log = opts.Logger

	return nil
}

// ConnectTopic connects to a topic if the transport is a bridge
func (t *Transport) ConnectTopic(topic string) (bus.BridgeConnection, error) {
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
func (c *Conn) Start(pod *bus.Pod) {
	c.pod = pod

	c.pod.OnType(c.topic, func(msg bus.Message) error {
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
					continue
				}

				c.log.Error(errors.Wrap(err, "[bridge-nats] failed to ReadMessage, terminating connection"))
				break
			}

			c.log.Debug("[bridge-nats] recieved message via", c.topic)

			msg, err := bus.MsgFromBytes(message.Data)
			if err != nil {
				c.log.Debug(errors.Wrap(err, "[bridge-nats] failed to MsgFromBytes, falling back to raw data").Error())

				msg = bus.NewMsg(c.topic, message.Data)
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
