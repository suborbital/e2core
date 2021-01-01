package grav

import "github.com/pkg/errors"

// ErrNoReceipt is returned when a method is called on a nil ticket
var ErrNoReceipt = errors.New("message receipt is nil")

// MsgReceipt represents a "ticket" that references a message that was sent with the hopes of getting a response
// The embedded pod is a pointer to the pod that sent the original message, and therefore any ticket methods used
// will replace the OnFunc of the pod.
type MsgReceipt struct {
	UUID string
	pod  *Pod
}

// WaitOn will block until a response to the message is recieved and passes it to the provided onFunc.
// onFunc errors are propogated to the caller.
func (m *MsgReceipt) WaitOn(onFunc MsgFunc) error {
	return m.WaitUntil(nil, onFunc)
}

// WaitUntil will block until a response to the message is recieved and passes it to the provided onFunc.
// ErrWaitTimeout is returned if the timeout elapses, onFunc errors are propogated to the caller.
func (m *MsgReceipt) WaitUntil(timeout TimeoutFunc, onFunc MsgFunc) error {
	if m == nil {
		return ErrNoReceipt
	}

	return m.pod.waitOnReply(m, timeout, onFunc)
}

// OnReply will set the pod's OnFunc to the provided MsgFunc and set it to run asynchronously when a reply is received
// onFunc errors are discarded.
func (m *MsgReceipt) OnReply(mfn MsgFunc) error {
	if m == nil {
		return ErrNoReceipt
	}

	go func() {
		m.pod.waitOnReply(m, nil, mfn)
	}()

	return nil
}
