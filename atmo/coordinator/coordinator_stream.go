package coordinator

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/suborbital/atmo/atmo/coordinator/sequence"
	"github.com/suborbital/atmo/directive"
	"github.com/suborbital/grav/grav"
	"github.com/suborbital/reactr/request"
	"github.com/suborbital/reactr/rt"
	"github.com/suborbital/vektor/vk"
)

type messageScope struct {
	MessageUUID string `json:"messageUUID"`
}

func (c *Coordinator) streamConnectionForDirectiveHandler(handler directive.Handler) {
	handlerIdent := fmt.Sprintf("%s:%s", handler.Input.Source, handler.Input.Resource)

	c.log.Debug("setting up stream connection for", handlerIdent)

	connection, exists := c.connections[handler.Input.Source]
	if !exists {
		c.log.ErrorString("connection to", handler.Input.Source, "not configured, handler will not be mounted")
		return
	}

	if err := connection.ConnectBridgeTopic(handler.Input.Resource); err != nil {
		c.log.Error(errors.Wrapf(err, "failed to ConnectBridgeTopic for resource %s", handler.Input.Resource))
		return
	}

	if handler.RespondTo != "" {
		c.log.Debug("setting up respondTo stream connection for", handler.RespondTo)
		if err := connection.ConnectBridgeTopic(handler.RespondTo); err != nil {
			c.log.Error(errors.Wrapf(err, "failed to ConnectBridgeTopic for resource %s's respondTo topic %s", handler.Input.Resource, handler.RespondTo))
			return
		}
	}

	existingPod, exists := c.handlerPods[handlerIdent]
	if exists {
		existingPod.Disconnect()
		delete(c.handlerPods, handlerIdent)
	}

	pod := connection.Connect()
	pod.OnType(handler.Input.Resource, func(msg grav.Message) error {
		ctx := vk.NewCtx(c.log, nil, nil)
		ctx.UseScope(messageScope{msg.UUID()})

		ctx.Log.Info("handling message", msg.UUID(), "for handler", handlerIdent)

		req := &request.CoordinatedRequest{
			Method:      atmoMethodStream,
			URL:         handler.Input.Resource,
			ID:          uuid.New().String(),
			Body:        msg.Data(),
			Headers:     map[string]string{},
			RespHeaders: map[string]string{},
			Params:      map[string]string{},
			State:       map[string][]byte{},
		}

		// a sequence executes the handler's steps and manages its state
		seq := sequence.New(handler.Steps, c.exec, ctx)

		if err := seq.Execute(req); err != nil {
			if runErr, isRunErr := err.(rt.RunErr); isRunErr {
				c.log.Error(errors.Wrapf(runErr, "handler for %s returned an error", handler.Input.Resource))
			} else {
				c.log.Error(errors.Wrapf(err, "schedule %s failed", handler.Input.Resource))
			}
		}

		result := resultFromState(handler, req.State)

		replyTopic := handler.Input.Resource
		if handler.RespondTo != "" {
			replyTopic = handler.RespondTo
		}

		pod.ReplyTo(msg, grav.NewMsg(replyTopic, result))

		return nil
	})

	// keep the pod in state so it isn't GC'd
	c.handlerPods[handlerIdent] = pod
}
