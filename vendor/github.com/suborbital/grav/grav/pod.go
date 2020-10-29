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

var (
	// podFeedbackMsgReplay is the message sent via feedback channel when message replay is desired
	podFeedbackMsgReplay  = NewMsg(msgTypePodFeedback, []byte{})
	podFeedbackMsgSuccess = NewMsg(msgTypePodFeedback, []byte{})
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

// On sets the function to be called whenever this pod recieves a message from the bus. If nil is passed, the pod will ignore all messages.
// Calling On multiple times causes the function to be overwritten. To recieve using two different functions, create two pods.
func (p *Pod) On(onFunc MsgFunc) {
	p.onFuncLock.Lock()
	defer p.onFuncLock.Unlock()

	// reset the message filter when the onFunc is changed
	p.messageFilter = newMessageFilter()

	p.setOnFunc(onFunc)
}

// OnType sets the function to be called whenever this pod recieves a message and sets the pod's filter to only include certain message types
func (p *Pod) OnType(onFunc MsgFunc, msgTypes ...string) {
	p.onFuncLock.Lock()
	defer p.onFuncLock.Unlock()

	// reset the message filter when the onFunc is changed
	p.messageFilter = newMessageFilter()
	p.TypeInclusive = false // only allow the listed types

	for _, t := range msgTypes {
		p.FilterType(t, true)
	}

	p.setOnFunc(onFunc)
}

// ErrMsgNotWanted is used by WaitOn to determine if the current message is what's being waited on
var ErrMsgNotWanted = errors.New("message not wanted")

// WaitOn takes a function to be called whenever this pod recieves a message and blocks until that function returns
// something other than ErrMsgNotWanted. WaitOn should be used if there is a need to wait for a particular message.
// When the onFunc returns something other than ErrMsgNotWanted (such as nil or a different error), WaitOn will return and set
// the onFunc to nil. If an error other than ErrMsgNotWanted is returned from the onFunc, it will be propogated to the caller.
func (p *Pod) WaitOn(onFunc MsgFunc) error {
	p.onFuncLock.Lock()
	errChan := make(chan error)

	p.setOnFunc(func(msg Message) error {
		if err := onFunc(msg); err != ErrMsgNotWanted {
			errChan <- err
		}

		return nil
	})

	p.onFuncLock.Unlock() // can't stay locked here or the onFunc will never be called

	err := <-errChan

	p.onFuncLock.Lock()
	defer p.onFuncLock.Unlock()

	p.setOnFunc(nil)

	return err
}

// Send emits a message to be routed to the bus
func (p *Pod) Send(msg Message) {
	// check to see if the pod has died (aka disconnected)
	if p.dead.Load().(bool) == true {
		return
	}

	p.FilterUUID(msg.UUID(), false) // don't allow the same message to bounce back through this pod

	p.busChan <- msg
}

// setOnFunc sets the OnFunc. THIS DOES NOT LOCK. THE CALLER MUST LOCK.
func (p *Pod) setOnFunc(on MsgFunc) {
	p.onFunc = on

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
