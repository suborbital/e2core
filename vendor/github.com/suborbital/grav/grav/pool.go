package grav

import (
	"errors"
	"sync"
)

const (
	highWaterMark = 64
)

var (
	errFailedMessage    = errors.New("pod reports failed message")
	errFailedMessageMax = errors.New("pod reports max number of failed messages, will terminate connection")
)

// connectionPool is a ring of connections to pods
// which will be iterated over constantly in order to send
// incoming messages to them
type connectionPool struct {
	current *podConnection

	maxID int64
	lock  sync.Mutex
}

func newConnectionPool() *connectionPool {
	p := &connectionPool{
		current: nil,
		maxID:   0,
		lock:    sync.Mutex{},
	}

	return p
}

// insert inserts a new connection into the ring
func (c *connectionPool) insert(pod *Pod) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.maxID++
	id := c.maxID

	conn := newPodConnection(id, pod)

	// if there's nothing in the ring, create a "ring of one"
	if c.current == nil {
		conn.next = conn
		c.current = conn
	} else {
		c.current.insertAfter(conn)
	}
}

// peek returns a peek at the next connection in the ring wihout advancing the ring's current location
func (c *connectionPool) peek() *podConnection {
	c.lock.Lock()
	defer c.lock.Unlock()

	return c.current.next
}

// next returns the next connection in the ring
func (c *connectionPool) next() *podConnection {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.current = c.current.next

	return c.current
}

// prepareNext ensures that the next pod connection in the ring is ready to recieve
// new messages by checking its status, deleting it if unhealthy or disconnected, replaying the message
// buffer if needed, or flushing failed messages back onto its channel if needeed.
func (c *connectionPool) prepareNext(buffer *MsgBuffer) error {
	// peek gives us the next conn without advancing the ring
	// this makes it easy to delete the next conn if it's unhealthy
	next := c.peek()

	// check the state of the next connection
	status := next.checkStatus()

	if status.Error != nil {
		// if the connection has an issue, handle it
		if status.Error == errFailedMessageMax {
			c.deleteNext()
			return errors.New("removing next podConnection")
		}
	} else if status.WantsDisconnect {
		// if the pod has requested disconnection, grant its wish
		c.deleteNext()
		return errors.New("next pod requested disconnection, removing podConnection")
	} else if status.WantsReplay {
		// if the pod has indicated that it wants a replay of recent messages, do so
		c.replayNext(buffer)
	}

	if status.HadSuccess {
		// if the most recent status check indicates there was a success,
		// then tell the connection to flush any failed messages
		// this is a no-op if there are no failed messages queued
		next.flushFailed()
	}

	return nil
}

// replayNext replays the current message buffer into the next connection
func (c *connectionPool) replayNext(buffer *MsgBuffer) {
	next := c.peek()

	// iterate over the buffer and send each message to the pod
	buffer.Iter(func(msg Message) error {
		next.send(msg)

		return nil
	})
}

// deleteNext deletes the next connection in the ring
// this is useful after having checkError'd the next conn
// and seeing that it's unhealthy
func (c *connectionPool) deleteNext() {
	c.lock.Lock()
	defer c.lock.Unlock()

	next := c.current.next

	// indicate the conn is dead so future attempts to send are abandonded
	next.disconnect()

	if next == c.current {
		// if there's only one thing in the ring, empty the ring
		c.current = nil
	} else {
		// cut out `next` and link `current` to `next-next`
		c.current.next = next.next
	}
}
