package withdraw

import (
	"sync/atomic"
)

// Signaler allows a connection to be notified about a withdraw event and report
// back to the hub that a withdraw has completed (using the WithdrawChan and DoneChan, respectively)
type Signaler struct {
	withdrawChan  chan struct{}
	doneChan      chan struct{}
	selfWithdrawn atomic.Value
	peerWithdrawn atomic.Value
}

// NewSignaler creates a new Signaler based on the provided context
func NewSignaler() *Signaler {
	w := &Signaler{
		withdrawChan:  make(chan struct{}, 1),
		doneChan:      make(chan struct{}, 1),
		selfWithdrawn: atomic.Value{},
		peerWithdrawn: atomic.Value{},
	}

	w.selfWithdrawn.Store(false)
	w.peerWithdrawn.Store(false)

	return w
}

// Signal sends the withdraw signal and returns a channel that is written to
// when the receiver has indicated a completed withdraw
func (s *Signaler) Signal() chan struct{} {
	s.selfWithdrawn.Store(true)
	s.withdrawChan <- struct{}{}

	return s.doneChan
}

// Listen returns a channel that is written to when the withdraw has been triggered
func (s *Signaler) Listen() chan struct{} {
	return s.withdrawChan
}

// Done indicates to the Signal() caller that the withdraw has completed
func (s *Signaler) Done() {
	s.doneChan <- struct{}{}
}

// SelfWithdrawn returns true if self has withdrawn
func (s *Signaler) SelfWithdrawn() bool {
	return s.selfWithdrawn.Load().(bool)
}

// SetPeerWithdrawn sets the peer as withdrawn
func (s *Signaler) SetPeerWithdrawn() {
	s.peerWithdrawn.Store(true)
}

// PeerWithdrawn returns true if peer has withdrawn
func (s *Signaler) PeerWithdrawn() bool {
	return s.peerWithdrawn.Load().(bool)
}
