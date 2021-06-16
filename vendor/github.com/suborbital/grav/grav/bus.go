package grav

const (
	defaultBusChanSize = 256
)

// messageBus is responsible for emitting messages among the connected pods
// and managing the failure cases for those pods
type messageBus struct {
	busChan MsgChan
	pool    *connectionPool
	buffer  *MsgBuffer
}

// newMessageBus creates a new messageBus
func newMessageBus() *messageBus {
	b := &messageBus{
		busChan: make(chan Message, defaultBusChanSize),
		pool:    newConnectionPool(),
		buffer:  NewMsgBuffer(defaultBufferSize),
	}

	b.start()

	return b
}

// addPod adds a pod to the connection pool
func (b *messageBus) addPod(pod *Pod) {
	b.pool.insert(pod)
}

func (b *messageBus) start() {
	go func() {
		// continually take new messages and for each,
		// grab the next active connection from the ring and then
		// start traversing around the ring to emit the message to
		// each connection until landing back at the beginning of the
		// ring, and repeat forever when each new message arrives
		for msg := range b.busChan {
			for {
				// make sure the next pod is ready for messages
				if err := b.pool.prepareNext(b.buffer); err == nil {
					break
				}
			}

			startingConn := b.pool.next()

			b.traverse(msg, startingConn)

			b.buffer.Push(msg)
		}
	}()
}

func (b *messageBus) traverse(msg Message, start *podConnection) {
	startID := start.ID
	conn := start

	for {
		// send the message to the pod
		conn.send(msg)

		// run checks on the next podConnection to see if
		// anything needs to be done (including potentially deleting it)
		next := b.pool.peek()
		if err := b.pool.prepareNext(b.buffer); err != nil {
			if startID == next.ID {
				startID = next.next.ID
			}
		}

		// now advance the ring
		conn = b.pool.next()

		if startID == conn.ID {
			// if we have arrived back at the starting point on the ring
			// we have done our job and are ready for the next message
			break
		}
	}
}
