package bus

import (
	"github.com/pkg/errors"

	"github.com/suborbital/e2core/bus/bus/withdraw"
	"github.com/suborbital/vektor/vlog"
)

type connectionHandler struct {
	UUID      string
	Conn      Connection
	Pod       *Pod
	Signaler  *withdraw.Signaler
	ErrChan   chan error
	BelongsTo string
	Interests []string
	Log       *vlog.Logger
}

// Start starts up a listener to read messages from the connection into the Grav bus
func (c *connectionHandler) Start() {
	withdrawChan := c.Signaler.Listen()

	go func() {
		<-withdrawChan

		c.Log.Debug("[grav] sending withdraw and disconnecting")

		if err := c.Conn.SendWithdraw(&Withdraw{}); err != nil {
			c.Log.Error(errors.Wrapf(err, "[grav] failed to SendWithdraw to connection %s", c.UUID))
			c.ErrChan <- err
		}

		c.Signaler.Done()
	}()

	go func() {
		for {
			msg, withdraw, err := c.Conn.ReadMsg()
			if err != nil {
				if !(c.Signaler.SelfWithdrawn() || c.Signaler.PeerWithdrawn()) {
					c.Log.Error(errors.Wrapf(err, "[grav] failed to ReadMsg from connection %s", c.UUID))
					c.ErrChan <- err
				} else {
					c.Log.Debug("[grav] failed to ReadMsg from withdrawn connection, ignoring:", err.Error())
				}

				return
			}

			if withdraw != nil {
				c.Log.Debug("[grav] peer has withdrawn, disconnecting")

				c.Signaler.SetPeerWithdrawn()

				return
			}

			c.Log.Debug("received message", msg.UUID())

			c.Pod.Send(msg)
		}
	}()
}

func (c *connectionHandler) Send(msg Message) error {
	if c.Signaler.PeerWithdrawn() {
		return ErrNodeWithdrawn
	}

	if err := c.Conn.SendMsg(msg); err != nil {
		c.ErrChan <- err
		return errors.Wrap(err, "failed to SendMsg")
	}

	return nil
}

// Close stops outgoing messages and closes the underlying connection
func (c *connectionHandler) Close() error {
	if err := c.Conn.Close(); err != nil {
		return errors.Wrap(err, "[grav] failed to Conn.Close")
	}

	return nil
}
