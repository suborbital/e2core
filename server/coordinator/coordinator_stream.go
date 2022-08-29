package coordinator

// TODO: implement streaming triggers properly

// import (
// 	"fmt"

// 	"github.com/pkg/errors"

// 	"github.com/suborbital/appspec/appsource"
// 	"github.com/suborbital/deltav/bus/bus"
// 	"github.com/suborbital/appspec/directive"
// 	"github.com/suborbital/deltav/scheduler"
// 	"github.com/suborbital/deltav/server/coordinator/sequence"
// 	"github.com/suborbital/appspec/request"
// 	"github.com/suborbital/vektor/vk"
// )

// type messageScope struct {
// 	MessageUUID string `json:"messageUUID"`
// }

// func (c *Coordinator) streamConnectionForDirectiveHandler(handler directive.Handler, appInfo appsource.Meta) {
// 	handlerIdent := fmt.Sprintf("%s:%s", handler.Input.Source, handler.Input.Resource)

// 	c.log.Debug("setting up stream connection for", handlerIdent)

// 	connectionKey := fmt.Sprintf(connectionKeyFormat, appInfo.Identifier, appInfo.AppVersion, handler.Input.Source)

// 	connection, exists := c.connections[connectionKey]
// 	if !exists {
// 		c.log.ErrorString("connection to", handler.Input.Source, "not configured, handler will not be mounted")
// 		return
// 	}

// 	if err := connection.ConnectBridgeTopic(handler.Input.Resource); err != nil {
// 		c.log.Error(errors.Wrapf(err, "failed to ConnectBridgeTopic for resource %s", handler.Input.Resource))
// 		return
// 	}

// 	if handler.RespondTo != "" {
// 		c.log.Debug("setting up respondTo stream connection for", handler.RespondTo)
// 		if err := connection.ConnectBridgeTopic(handler.RespondTo); err != nil {
// 			c.log.Error(errors.Wrapf(err, "failed to ConnectBridgeTopic for resource %s's respondTo topic %s", handler.Input.Resource, handler.RespondTo))
// 			return
// 		}
// 	}

// 	existingPod, exists := c.handlerPods[handlerIdent]
// 	if exists {
// 		existingPod.Disconnect()
// 		delete(c.handlerPods, handlerIdent)
// 	}

// 	pod := connection.Connect()
// 	pod.OnType(handler.Input.Resource, func(msg bus.Message) error {
// 		ctx := vk.NewCtx(c.log, nil, nil)
// 		ctx.UseScope(messageScope{msg.UUID()})

// 		ctx.Log.Debug("handling message", msg.UUID(), "for handler", handlerIdent)

// 		req := &request.CoordinatedRequest{
// 			Method:      deltavMethodStream,
// 			URL:         handler.Input.Resource,
// 			ID:          ctx.RequestID(),
// 			Body:        msg.Data(),
// 			Headers:     map[string]string{},
// 			RespHeaders: map[string]string{},
// 			Params:      map[string]string{},
// 			State:       map[string][]byte{},
// 		}

// 		// a sequence executes the handler's steps and manages its state.
// 		seq, err := sequence.New(handler.Steps, req, ctx)
// 		if err != nil {
// 			c.log.Error(errors.Wrap(err, "failed to sequence.New"))
// 			return nil
// 		}

// 		if err := seq.Execute(c.exec); err != nil {
// 			if runErr, isRunErr := err.(scheduler.RunErr); isRunErr {
// 				c.log.Error(errors.Wrapf(runErr, "handler for %s returned an error", handler.Input.Resource))
// 			} else {
// 				c.log.Error(errors.Wrapf(err, "schedule %s failed", handler.Input.Resource))
// 			}
// 		}

// 		result := resultFromState(handler, req.State)

// 		replyTopic := handler.Input.Resource
// 		if handler.RespondTo != "" {
// 			replyTopic = handler.RespondTo
// 		}

// 		pod.ReplyTo(msg, bus.NewMsgWithParentID(replyTopic, ctx.RequestID(), result))

// 		return nil
// 	})

// 	// keep the pod in state so it isn't GC'd.
// 	c.handlerPods[handlerIdent] = pod
// }
