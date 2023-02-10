package bus

import (
	"github.com/pkg/errors"
	"github.com/rs/zerolog"

	"github.com/suborbital/e2core/foundation/bus/bus/withdraw"
)

type connectionHandler struct {
	UUID      string
	Conn      Connection
	Pod       *Pod
	Signaler  *withdraw.Signaler
	ErrChan   chan error
	BelongsTo string
	Interests []string
	Log       zerolog.Logger
}

// Start starts up a listener to read messages from the connection into the Grav bus
func (c *connectionHandler) Start() {
	ll := c.Log.With().Str("method", "Start").Logger()
	withdrawChan := c.Signaler.Listen()

	go func() {
		<-withdrawChan

		ll.Debug().Msg("sending withdraw and disconnecting")

		if err := c.Conn.SendWithdraw(&Withdraw{}); err != nil {
			ll.Err(err).
				Str("connectionUUID", c.UUID).
				Msg("failed to SendWithdraw to connection")
			c.ErrChan <- err
		}

		c.Signaler.Done()
	}()

	go func() {
		for {
			msg, connWithdraw, err := c.Conn.ReadMsg()
			if err != nil {
				if !(c.Signaler.SelfWithdrawn() || c.Signaler.PeerWithdrawn()) {
					ll.Err(err).Str("connectionUUID", c.UUID).Msg("failed to ReadMsg from connection")
					c.ErrChan <- err
				} else {
					ll.Debug().Msgf("failed to ReadMsg from withdrawn connection, ignoring: %s", err.Error())
				}

				return
			}

			if connWithdraw != nil {
				ll.Debug().Msg("peer has withdrawn, disconnecting")

				c.Signaler.SetPeerWithdrawn()

				return
			}

			ll.Debug().Str("messageUUID", msg.UUID()).Msg("received message")

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
		return errors.Wrap(err, "[connectionHandler.Close] failed to Conn.Close")
	}

	return nil
}
