package grav

import (
	"errors"
	"sync"
	"sync/atomic"
)

const (
	// defaultPodChanSize is the default size of the channels used for pod - bus communication
	defaultPodChanSize = 128
)

// podFeedbackMsgReplay and others are the messages sent via feedback channel when the pod needs to communicate its state to the bus
var (
	podFeedbackMsgReplay     = NewMsg(msgTypePodFeedback, []byte{})
	podFeedbackMsgSuccess    = NewMsg(msgTypePodFeedback, []byte{})
	podFeedbackMsgDisconnect = NewMsg(msgTypePodFeedback, []byte{})
)

/**
                              ┌─────────────────────┐
                              │                     │
            ──messageChan─────▶─────────────────────▶─────On────▶
┌────────┐                    │       		        │             ┌───────────────┐
│  Bus   │                    │        Pod          │             │   Pod Owner   │
└────────┘                    │       		        │             └───────────────┘
            ◀───BusChan------─◀─────────────────────◀────Send────
                              │                     │
                              └─────────────────────┘

Created with Monodraw
**/

// Pod is a connection to Grav
// Pods are bi-directional. Messages can be sent to them from the bus, and they can be used to send messages
// to the bus. Pods are meant to be extremely lightweight with no persistence they are meant to quickly
// and immediately route a message between its owner and the Bus. The Bus is responsible for any "smarts".
// Messages coming from the bus are filtered using the pod's messageFilter, which is configurable by the caller.
type Pod struct {
	onFunc     MsgFunc // the onFunc is called whenever a message is recieved
	onFuncLock sync.RWMutex

	messageChan  MsgChan // messageChan is used to recieve messages coming from the bus
	feedbackChan MsgChan // feedbackChan is used to send "feedback" to the bus about the pod's status
	busChan      MsgChan // busChan is used to emit messages to the bus

	*messageFilter // the embedded messageFilter controls which messages reach the onFunc

	opts *podOpts

	dead *atomic.Value
}

type podOpts struct {
	WantsReplay bool
	replayOnce  sync.Once
}

// newPod creates a new Pod
func newPod(busChan MsgChan, opts *podOpts) *Pod {
	p := &Pod{
		onFuncLock:    sync.RWMutex{},
		messageChan:   make(chan Message, defaultPodChanSize),
		feedbackChan:  make(chan Message, defaultPodChanSize),
		busChan:       busChan,
		messageFilter: newMessageFilter(),
		opts:          opts,
		dead:          &atomic.Value{},
	}

	// do some "delayed setup"
	p.opts.replayOnce = sync.Once{}
	p.dead.Store(false)

	p.start()

	return p
}

// Send emits a message to be routed to the bus
// If the returned ticket is nil, it means the pod was unable to send
// It is safe to call methods on a nil ticket, they will error with ErrNoTicket
// This means error checking can be done on a chained call such as err := p.Send(msg).Wait(...)
func (p *Pod) Send(msg Message) *MsgReceipt {
	// check to see if the pod has died (aka disconnected)
	if p.dead.Load().(bool) == true {
		return nil
	}

	p.FilterUUID(msg.UUID(), false) // don't allow the same message to bounce back through this pod

	p.busChan <- msg

	t := &MsgReceipt{
		UUID: msg.UUID(),
		pod:  p,
	}

	return t
}

// ReplyTo sends a response to a message. The reply message's ticket is returned.
func (p *Pod) ReplyTo(inReplyTo Message, msg Message) *MsgReceipt {
	msg.SetReplyTo(inReplyTo.UUID())

	return p.Send(msg)
}

// On sets the function to be called whenever this pod recieves a message from the bus. If nil is passed, the pod will ignore all messages.
// Calling On multiple times causes the function to be overwritten. To recieve using two different functions, create two pods.
// Errors returned from the onFunc are interpreted as problems handling messages. Too many errors will result in the pod being disconnected.
// Failed messages will be replayed when messages begin to succeed. Returning an error is inadvisable unless there is a real problem handling messages.
func (p *Pod) On(onFunc MsgFunc) {
	p.onFuncLock.Lock()
	defer p.onFuncLock.Unlock()

	p.setOnFunc(onFunc)
}

// OnType sets the function to be called whenever this pod recieves a message and sets the pod's filter to only receive certain message types.
// The same rules as `On` about error handling apply to OnType.
func (p *Pod) OnType(msgType string, onFunc MsgFunc) {
	p.onFuncLock.Lock()
	defer p.onFuncLock.Unlock()

	p.setOnFunc(onFunc)

	p.FilterType(msgType, true)
	p.TypeInclusive = false // only allow the listed types
}

// Disconnect indicates to the bus that this pod is no longer needed and should be disconnected.
// Sending will immediately become unavailable, and the pod will soon stop recieving messages.
func (p *Pod) Disconnect() {
	// stop future messages from being sent and then indicate to the bus that disconnection is desired
	// The bus will close the busChan, which will cause the onFunc listener to quit.
	p.dead.Store(true)
	p.feedbackChan <- podFeedbackMsgDisconnect
}

// ErrMsgNotWanted is used by WaitOn to determine if the current message is what's being waited on
var ErrMsgNotWanted = errors.New("message not wanted")

// ErrWaitTimeout is returned if a timeout is exceeded
var ErrWaitTimeout = errors.New("waited past timeout")

// WaitOn takes a function to be called whenever this pod recieves a message and blocks until that function returns
// something other than ErrMsgNotWanted. WaitOn should be used if there is a need to wait for a particular message.
// When the onFunc returns something other than ErrMsgNotWanted (such as nil or a different error), WaitOn will return and set
// the onFunc to nil. If an error other than ErrMsgNotWanted is returned from the onFunc, it will be propogated to the caller.
// WaitOn will block forever if the desired message is never found. Use WaitUntil if a timeout is desired.
func (p *Pod) WaitOn(onFunc MsgFunc) error {
	return p.WaitUntil(nil, onFunc)
}

// WaitUntil takes a function to be called whenever this pod recieves a message and blocks until that function returns
// something other than ErrMsgNotWanted. WaitOn should be used if there is a need to wait for a particular message.
// When the onFunc returns something other than ErrMsgNotWanted (such as nil or a different error), WaitUntil will return and set
// the onFunc to nil. If an error other than ErrMsgNotWanted is returned from the onFunc, it will be propogated to the caller.
// A timeout can be provided. If the timeout is non-nil and greater than 0, ErrWaitTimeout is returned if the time is exceeded.
func (p *Pod) WaitUntil(timeout TimeoutFunc, onFunc MsgFunc) error {
	p.onFuncLock.Lock()
	errChan := make(chan error)

	p.setOnFunc(func(msg Message) error {
		if err := onFunc(msg); err != nil {
			if err == ErrMsgNotWanted {
				return nil // don't do anything
			}

			errChan <- err
		} else {
			errChan <- nil
		}

		return nil
	})

	p.onFuncLock.Unlock() // can't stay locked here or the onFunc will never be called

	var onFuncErr error
	if timeout == nil {
		timeout = Timeout(-1)
	}

	select {
	case err := <-errChan:
		onFuncErr = err
	case <-timeout():
		onFuncErr = ErrWaitTimeout
	}

	p.onFuncLock.Lock()
	defer p.onFuncLock.Unlock()

	p.setOnFunc(nil)

	return onFuncErr
}

// waitOnReply waits on a reply message to arrive at the pod and then calls onFunc with that message.
// If the onFunc produces an error, it will be propogated to the caller.
// If a non-nil timeout greater than 0 is passed, the function will return ErrWaitTimeout if the timeout elapses.
func (p *Pod) waitOnReply(ticket *MsgReceipt, timeout TimeoutFunc, onFunc MsgFunc) error {
	var reply Message

	if err := p.WaitUntil(timeout, func(msg Message) error {
		if msg.ReplyTo() != ticket.UUID {
			return ErrMsgNotWanted
		}

		reply = msg

		return nil
	}); err != nil {
		return err
	}

	return onFunc(reply)
}

// setOnFunc sets the OnFunc. THIS DOES NOT LOCK. THE CALLER MUST LOCK.
func (p *Pod) setOnFunc(on MsgFunc) {
	// reset the message filter when the onFunc is changed
	p.messageFilter = newMessageFilter()

	p.onFunc = on

	// request replay from the bus if needed
	if on != nil {
		p.opts.replayOnce.Do(func() {
			if p.opts.WantsReplay {
				p.feedbackChan <- podFeedbackMsgReplay
			}
		})
	}
}

// busChans returns the messageChan and feedbackChan to be used by the bus
func (p *Pod) busChans() (MsgChan, MsgChan) {
	return p.messageChan, p.feedbackChan
}

func (p *Pod) start() {
	go func() {
		// this loop ends when the bus closes the messageChan
		for {
			msg, ok := <-p.messageChan
			if !ok {
				break
			}

			go func() {
				p.onFuncLock.RLock() // in case the onFunc gets replaced
				defer p.onFuncLock.RUnlock()

				if p.onFunc == nil {
					return
				}

				if p.allow(msg) {
					if err := p.onFunc(msg); err != nil {
						// if the onFunc failed, send it back to the bus to be re-sent later
						p.feedbackChan <- msg
					} else {
						// if it was successful, a success message on the channel lets the conn know all is well
						p.feedbackChan <- podFeedbackMsgSuccess
					}
				}
			}()
		}

		// if we've gotten this far, it means the pod has been killed and should not be allowed to send
		p.dead.Store(true)
	}()
}
