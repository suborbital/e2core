package coordinator

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/suborbital/atmo/directive"
	"github.com/suborbital/grav/grav"
	"github.com/suborbital/reactr/request"
	"github.com/suborbital/vektor/vk"
)

func (c *Coordinator) streamConnectionForDirectiveHandler(handler directive.Handler) {
	handlerIdent := fmt.Sprintf("%s:%s", handler.Input.Source, handler.Input.Resource)

	connection, exists := c.connections[handler.Input.Source]
	if !exists {
		c.log.ErrorString("connection to", handler.Input.Source, "not configured, handler will not be mounted")
		return
	}

	if err := connection.ConnectBridgeTopic(handler.Input.Resource); err != nil {
		c.log.Error(errors.Wrapf(err, "failed to ConnectBridgeTopic for resource %s", handler.Input.Resource))
		return
	}

	existingPod, exists := c.handlerPods[handlerIdent]
	if exists {
		existingPod.Disconnect()
		delete(c.handlerPods, handlerIdent)
	}

	pod := connection.Connect()
	pod.OnType(handler.Input.Resource, func(msg grav.Message) error {
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
		seq := newSequence(handler.Steps, c.grav.Connect, vk.NewCtx(c.log, nil, nil))

		seqState, err := seq.exec(req)
		if err != nil {
			if errors.Is(err, ErrSequenceRunErr) && seqState.err != nil {
				c.log.Error(errors.Wrapf(seqState.err, "handler for %s returned an error", handler.Input.Resource))
			} else {
				c.log.Error(errors.Wrapf(err, "schedule %s failed", handler.Input.Resource))
			}
		}

		result := resultFromState(handler, seqState.state)

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
