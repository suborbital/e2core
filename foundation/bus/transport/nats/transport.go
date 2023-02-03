package nats

import (
	"time"

	"github.com/nats-io/nats.go"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"

	"github.com/suborbital/e2core/foundation/bus/bus"
)

// Transport is a transport that connects Grav nodes via NATS
type Transport struct {
	opts *bus.BridgeOptions
	log  zerolog.Logger

	serverConn *nats.Conn
}

// Conn implements transport.TopicConnection and represents a subscribe/send pair for a NATS topic
type Conn struct {
	topic string
	log   zerolog.Logger
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
	t.log = opts.Logger.With().Str("transportType", "NATS").Logger()

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
		log:   t.log.With().Str("topic", topic).Logger(),
		sub:   sub,
		pubFn: pubFn,
	}

	return conn, nil
}

// Start begins the receiving of messages
func (c *Conn) Start(pod *bus.Pod) {
	ll := c.log.With().Str("method", "Start").Logger()

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
			message, err := c.sub.NextMsg(time.Second * 60)
			if err != nil {
				if err == nats.ErrTimeout {
					continue
				}

				ll.Err(err).Msg("c.sub.NextMsg")

				break
			}

			ll.Debug().Msg("received message from topic")

			msg, err := bus.MsgFromBytes(message.Data)
			if err != nil {
				ll.Err(err).Msg("bus.MsgFromBytes, falling back to raw data")

				msg = bus.NewMsg(c.topic, message.Data)
			}

			// send to the Grav instance
			c.pod.Send(msg)
		}
	}()
}

// Close closes the underlying connection
func (c *Conn) Close() {
	ll := c.log.With().Str("method", "Close").Logger()

	ll.Debug().Msg("connection is closing")

	if err := c.sub.Unsubscribe(); err != nil {
		ll.Err(err).Msg("connection failed to close")
	}
}
