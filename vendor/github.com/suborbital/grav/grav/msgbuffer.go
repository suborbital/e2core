package grav

import (
	"sync"
)

const (
	defaultBufferSize = 128
)

// MsgBuffer is a buffer of messages with a particular size limit.
// Oldest messages are automatically evicted as new ones are added
// past said limit. Push() and Iter() are thread-safe.
type MsgBuffer struct {
	msgs       map[string]Message
	order      []string
	limit      int
	startIndex int
	lock       sync.RWMutex
}

func NewMsgBuffer(limit int) *MsgBuffer {
	m := &MsgBuffer{
		msgs:       map[string]Message{},
		order:      []string{},
		limit:      limit,
		startIndex: 0,
		lock:       sync.RWMutex{},
	}

	return m
}

// Push pushes a new message onto the end of the buffer and evicts the oldest, if needed (based on limit)
func (m *MsgBuffer) Push(msg Message) {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.msgs[msg.UUID()] = msg

	lastIndex := len(m.order) - 1

	if len(m.order) == m.limit {
		delete(m.msgs, m.order[m.startIndex]) // delete the current "first"

		m.order[m.startIndex] = msg.UUID()

		if m.startIndex == lastIndex {
			m.startIndex = 0
		} else {
			m.startIndex++
		}
	} else {
		m.order = append(m.order, msg.UUID())
	}
}

// Iter calls msgFunc once per message in the buffer
func (m *MsgBuffer) Iter(msgFunc MsgFunc) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	if len(m.order) == 0 {
		return
	}

	index := m.startIndex
	lastIndex := len(m.order) - 1

	more := true
	for more {
		uuid := m.order[index]
		msg := m.msgs[uuid]

		msgFunc(msg)

		newIndex := index
		if newIndex == lastIndex {
			newIndex = 0
		} else {
			newIndex++
		}

		if newIndex == m.startIndex {
			more = false
		}

		index = newIndex
	}
}
