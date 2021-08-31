package grav

import "sync"

// podConnection is a connection to a pod via its messageChan
// podConnection is also a circular linked list/ring of connections
// that is meant to be iterated around and inserted into/removed from
// forever as the bus sends events to the registered pods
type podConnection struct {
	ID   int64
	next *podConnection

	messageChan  MsgChan
	feedbackChan MsgChan

	failed []Message

	lock      *sync.RWMutex
	connected bool
}

// connStatus is used to communicate the status of a podConnection back to the bus
type connStatus struct {
	HadSuccess      bool
	WantsReplay     bool
	WantsDisconnect bool
	Error           error
}

func newPodConnection(id int64, pod *Pod) *podConnection {
	msgChan, feedbackChan := pod.busChans()

	p := &podConnection{
		ID:           id,
		next:         nil,
		messageChan:  msgChan,
		feedbackChan: feedbackChan,
		failed:       []Message{},
		lock:         &sync.RWMutex{},
		connected:    true,
	}

	return p
}

// send asynchronously writes a message to a connection's messageChan
// ordering to the messageChan if it becomes full is not guaranteed, this
// is sacrificed to ensure that the bus does not block because of a delinquient pod
func (p *podConnection) send(msg Message) {
	go func() {
		p.lock.RLock()
		defer p.lock.RUnlock()

		// if the conn is dead, abandon the attempt
		if !p.connected {
			return
		}

		p.messageChan <- msg
	}()
}

// checkStatus checks the pod's feedback for any information or failed messages and drains the failures into the failed Message buffer
func (p *podConnection) checkStatus() *connStatus {
	status := &connStatus{
		HadSuccess:      false,
		WantsReplay:     false,
		WantsDisconnect: false,
		Error:           nil,
	}

	done := false
	for !done {
		select {
		case feedbackMsg := <-p.feedbackChan:
			if feedbackMsg == podFeedbackMsgSuccess {
				status.HadSuccess = true
			} else if feedbackMsg == podFeedbackMsgReplay {
				status.WantsReplay = true
			} else if feedbackMsg == podFeedbackMsgDisconnect {
				status.WantsDisconnect = true
			} else {
				p.failed = append(p.failed, feedbackMsg)
				status.Error = errFailedMessage
			}
		default:
			done = true
		}
	}

	if len(p.failed) >= highWaterMark {
		status.Error = errFailedMessageMax
	}

	return status
}

func (p *podConnection) disconnect() {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.connected = false
	close(p.messageChan)
}

// flushFailed takes all of the failed messages in the failed queue
// and pushes them back out onto the pod's channel
func (p *podConnection) flushFailed() {
	for i := range p.failed {
		failedMsg := p.failed[i]

		p.send(failedMsg)
	}

	if len(p.failed) > 0 {
		p.failed = []Message{}
	}
}

// insertAfter inserts a new connection into the ring
func (p *podConnection) insertAfter(conn *podConnection) {
	next := p
	if p.next != nil {
		next = p.next
	}

	p.next = conn
	conn.next = next
}
