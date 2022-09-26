package kafka

import (
	"context"

	"github.com/pkg/errors"
	"github.com/twmb/franz-go/pkg/kgo"

	"github.com/suborbital/e2core/bus/bus"
	"github.com/suborbital/vektor/vlog"
)

// Transport is a transport that connects Grav nodes via kafka
type Transport struct {
	opts *bus.BridgeOptions
	log  *vlog.Logger

	endpoint string
}

// Conn implements transport.TopicConnection and represents a subscribe/send pair for a Kafka topic
type Conn struct {
	topic string
	log   *vlog.Logger
	pod   *bus.Pod

	conn *kgo.Client
}

// New creates a new Kafka transport
func New(endpoint string) (*Transport, error) {
	t := &Transport{}

	t.endpoint = endpoint

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
	client, err := kgo.NewClient(
		kgo.SeedBrokers(t.endpoint),
		kgo.ConsumeTopics(topic),
		kgo.ConsumeResetOffset(kgo.NewOffset().AtEnd()),
	)

	if err != nil {
		return nil, errors.Wrap(err, "failed to NewClient")
	}

	conn := &Conn{
		topic: topic,
		log:   t.log,
		conn:  client,
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

		record := &kgo.Record{Topic: c.topic, Value: msgBytes}
		if err := c.conn.ProduceSync(context.Background(), record).FirstErr(); err != nil {
			return errors.Wrap(err, "failed to ProduceSync")
		}

		return nil
	})

	go func() {
		for {
			fetches := c.conn.PollFetches(context.Background())
			if errs := fetches.Errors(); len(errs) > 0 {
				c.log.Error(errors.Wrap(errs[0].Err, "failed to PollFetches"))
				continue
			}

			iter := fetches.RecordIter()
			for !iter.Done() {
				record := iter.Next()

				c.log.Debug("[bridge-kafka] recieved message via", c.topic)

				msg, err := bus.MsgFromBytes(record.Value)
				if err != nil {
					c.log.Debug(errors.Wrap(err, "[bridge-kafka] failed to MsgFromBytes, falling back to raw data").Error())

					msg = bus.NewMsg(c.topic, record.Value)
				}

				// send to the Grav instance
				c.pod.Send(msg)
			}
		}
	}()
}

// Close closes the underlying connection
func (c *Conn) Close() {
	c.log.Debug("[bridge-kafka] connection for", c.topic, "is closing")

	c.conn.Close()
}
