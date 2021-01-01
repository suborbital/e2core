package grav

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// MsgTypeDefault and other represent message consts
const (
	MsgTypeDefault     string = "grav.default"
	msgTypePodFeedback string = "grav.feedback"
)

// MsgFunc is a callback function that accepts a message and returns an error
type MsgFunc func(Message) error

// MsgChan is a channel that accepts a message
type MsgChan chan Message

// Message represents a message
type Message interface {
	// Unique ID for this message
	UUID() string
	// ID of the parent event or request, such as HTTP request
	ParentID() string
	// The UUID of the message being replied to, if any
	ReplyTo() string
	// Allow setting a message UUID that this message is a response to
	SetReplyTo(string)
	// Type of message (application-specific)
	Type() string
	// Time the message was sent
	Timestamp() time.Time
	// Raw data of message
	Data() []byte
	// Unmarshal the message's data into a struct
	UnmarshalData(interface{}) error
	// Marshal the message itself to encoded bytes (JSON or otherwise)
	Marshal() ([]byte, error)
	// Unmarshal encoded Message into object
	Unmarshal([]byte) error
}

// NewMsg creates a new Message with the built-in `_message` type
func NewMsg(msgType string, data []byte) Message {
	return new(msgType, "", data)
}

// NewMsgWithParentID returns a new message with the provided parent ID
func NewMsgWithParentID(msgType, parentID string, data []byte) Message {
	return new(msgType, parentID, data)
}

// NewMsgReplyTo creates a new message in response to a previous message
func NewMsgReplyTo(ticket MsgReceipt, msgType string, data []byte) Message {
	m := new(msgType, "", data)
	m.SetReplyTo(ticket.UUID)

	return m
}

// MsgFromBytes returns a default _message that has been unmarshalled from bytes.
// Should only be used if the default _message type is being used.
func MsgFromBytes(bytes []byte) (Message, error) {
	m := &_message{}
	if err := m.Unmarshal(bytes); err != nil {
		return nil, err
	}

	return m, nil
}

// MsgFromRequest extracts an encoded Message from an HTTP request
func MsgFromRequest(r *http.Request) (Message, error) {
	defer r.Body.Close()
	bytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	return MsgFromBytes(bytes)
}

func new(msgType, parentID string, data []byte) Message {
	uuid := uuid.New()

	m := &_message{
		Meta: _meta{
			UUID:      uuid.String(),
			ParentID:  parentID,
			ReplyTo:   "",
			MsgType:   msgType,
			Timestamp: time.Now(),
		},
		Payload: _payload{
			Data: data,
		},
	}

	return m
}

// _message is a basic built-in implementation of Message
// most applications should define their own data structure
// that implements the interface
type _message struct {
	Meta    _meta    `json:"meta"`
	Payload _payload `json:"payload"`
}

type _meta struct {
	UUID      string    `json:"uuid"`
	ParentID  string    `json:"parent_id"`
	ReplyTo   string    `json:"response_to"`
	MsgType   string    `json:"msg_type"`
	Timestamp time.Time `json:"timestamp"`
}

type _payload struct {
	Data []byte `json:"data"`
}

func (m *_message) UUID() string {
	return m.Meta.UUID
}

func (m *_message) ParentID() string {
	return m.Meta.ParentID
}

func (m *_message) ReplyTo() string {
	return m.Meta.ReplyTo
}

func (m *_message) SetReplyTo(uuid string) {
	m.Meta.ReplyTo = uuid
}

func (m *_message) Type() string {
	return m.Meta.MsgType
}

func (m *_message) Timestamp() time.Time {
	return m.Meta.Timestamp
}

func (m *_message) Data() []byte {
	return m.Payload.Data
}

func (m *_message) UnmarshalData(target interface{}) error {
	return json.Unmarshal(m.Payload.Data, target)
}

func (m *_message) Marshal() ([]byte, error) {
	bytes, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}

	return bytes, nil
}

func (m *_message) Unmarshal(bytes []byte) error {
	return json.Unmarshal(bytes, m)
}
