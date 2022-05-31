package coordinator

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"

	"github.com/suborbital/vektor/vk"
	"github.com/suborbital/velocity/directive"
	"github.com/suborbital/velocity/scheduler"
	"github.com/suborbital/velocity/server/coordinator/sequence"
	"github.com/suborbital/velocity/server/request"
)

func (c *Coordinator) websocketHandlerForDirectiveHandler(handler directive.Handler) http.HandlerFunc {
	upgrader := websocket.Upgrader{} // use default options.

	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			c.log.Error(errors.Wrap(err, "failed to Upgrade connection"))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		defer conn.Close()

		var breakErr error

		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				breakErr = errors.Wrap(err, "failed to ReadMessage")
				break
			}

			ctx := vk.NewCtx(c.log, nil, nil)
			ctx.UseScope(messageScope{ctx.RequestID()})

			ctx.Log.Info("handling message", ctx.RequestID(), "from handler", handler.Input.Resource)

			req := &request.CoordinatedRequest{
				Method:      http.MethodGet,
				URL:         r.URL.String(),
				ID:          uuid.New().String(),
				Body:        message,
				Headers:     map[string]string{},
				RespHeaders: map[string]string{},
				Params:      map[string]string{},
				State:       map[string][]byte{},
			}

			// a sequence executes the handler's steps and manages its state.
			seq, err := sequence.New(handler.Steps, req, ctx)
			if err != nil {
				ctx.Log.Error(errors.Wrap(err, "failed to sequence.New"))
				breakErr = err
				break
			}

			if err := seq.Execute(c.exec); err != nil {
				if runErr, isRunErr := err.(scheduler.RunErr); isRunErr {
					if err := conn.WriteJSON(runErr); err != nil {
						breakErr = err
						break
					}
				}

				if err := conn.WriteJSON(vk.Wrap(http.StatusInternalServerError, err)); err != nil {
					breakErr = err
					break
				}

				continue
			}

			result := resultFromState(handler, req.State)

			if err := conn.WriteMessage(websocket.TextMessage, result); err != nil {
				breakErr = err
				break
			}
		}

		if breakErr != nil {
			c.log.Error(errors.Wrap(breakErr, "stream connection ended"))
		}
	}
}
