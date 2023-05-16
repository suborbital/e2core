package bus

import (
	"context"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"

	"github.com/suborbital/e2core/foundation/bus/bus/withdraw"
	"github.com/suborbital/e2core/foundation/tracing"
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
	ll := c.Log.With().Str("method", "connectionHandler.Start").Logger()
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
				// the error that happened is not an "I withdrew" or "my peer withdrew", it's a broken conn
				if !(c.Signaler.SelfWithdrawn() || c.Signaler.PeerWithdrawn()) {
					ll.Err(err).Str("connectionUUID", c.UUID).Msg("failed to ReadMsg from connection, sending to errchan")
					c.ErrChan <- err
				} else {
					ll.Err(err).Msg("failed to ReadMsg from withdrawn connection, ignoring")
				}

				return
			}

			if connWithdraw != nil {
				ll.Debug().Msg("peer has withdrawn, disconnecting")

				c.Signaler.SetPeerWithdrawn()

				return
			}

			ctx := otel.GetTextMapPropagator().Extract(context.Background(), msg)
			ctx, span := tracing.Tracer.Start(ctx, "connectionHandler.ReadMsg")

			msg.SetContext(ctx)

			ll.Debug().Str("messageUUID", msg.UUID()).Str("requestID", msg.ParentID()).Msg("received message")

			c.Pod.Send(msg)

			span.End()
		}
	}()
}

func (c *connectionHandler) Send(ctx context.Context, msg Message) error {
	ctx, span := tracing.Tracer.Start(ctx, "connectionHandler.send")
	defer span.End()

	ll := c.Log.With().Str("requestID", msg.ParentID()).Logger()
	if c.Signaler.PeerWithdrawn() {
		span.AddEvent("peer withdrawn")
		return ErrNodeWithdrawn
	}

	if err := c.Conn.SendMsg(msg); err != nil {
		ll.Err(err).Msg("c.conn.sendmsg returned an error")
		c.ErrChan <- err
		return errors.Wrap(err, "failed to SendMsg")
	}

	ll.Info().Msg("message sent successfully")

	return nil
}

// Close stops outgoing messages and closes the underlying connection
func (c *connectionHandler) Close() error {
	if err := c.Conn.Close(); err != nil {
		return errors.Wrap(err, "[connectionHandler.Close] failed to Conn.Close")
	}

	return nil
}
