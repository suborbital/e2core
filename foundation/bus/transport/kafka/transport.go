package kafka

import (
	"context"

	"github.com/pkg/errors"
	"github.com/twmb/franz-go/pkg/kgo"

	"github.com/suborbital/e2core/foundation/bus/bus"
	"github.com/suborbital/vektor/vlog"
)

const busMetadataHeaderKey = "bus.metadata"

// Transport is a transport that connects bus nodes via kafka
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

	t.log.Info("[bridge-kafka] connected to topic", topic)

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
		metadataBytes, err := msg.MarshalMetadata()
		if err != nil {
			return errors.Wrap(err, "failed to MarshalMetadata message")
		}

		// construct a record that contains the message payload
		// and store the Bus metadata (message UUID, etc) in a header
		record := &kgo.Record{
			Topic: c.topic,
			Value: msg.Data(),
			Headers: []kgo.RecordHeader{
				{
					Key:   busMetadataHeaderKey,
					Value: metadataBytes,
				},
			},
		}

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

				var msg bus.Message

				metaHeader := findMetaHeaderValue(busMetadataHeaderKey, record.Headers)
				if metaHeader == nil {
					// if there's no metadata, create a brand new message
					msg = bus.NewMsg(c.topic, record.Value)
				} else {
					reconstructedMsg, err := bus.MsgFromDataAndMeta(record.Value, metaHeader)
					if err != nil {
						c.log.Debug(errors.Wrap(err, "[bridge-kafka] failed to MsgFromDataAndMeta").Error())
						continue
					}

					msg = reconstructedMsg
				}

				// send to the Grav instance
				c.pod.Send(msg)
			}
		}
	}()
}

// findMetaHeaderValue returns the value of the header with the given key, or nil if it's not found
func findMetaHeaderValue(key string, headers []kgo.RecordHeader) []byte {
	for i, h := range headers {
		if h.Key == key {
			return headers[i].Value
		}
	}

	return nil
}

// Close closes the underlying connection
func (c *Conn) Close() {
	c.log.Debug("[bridge-kafka] connection for", c.topic, "is closing")

	c.conn.Close()
}
