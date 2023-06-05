package handlers

import (
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"

	"github.com/suborbital/e2core/nuexecutor/worker"
)

var upgrader = websocket.Upgrader{}

func WS(wc chan<- worker.Job, l zerolog.Logger) echo.HandlerFunc {
	ll := l.With().Str("handler", "WS").Logger()
	return func(c echo.Context) error {
		ws, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError).SetInternal(errors.Wrap(err, "gorilla.websocket.Upgrader.Upgrade"))
		}

		defer func() {
			_ = ws.Close()
		}()

		for {
			// Write
			err := ws.WriteMessage(websocket.TextMessage, []byte("Hello, Client!"))
			if err != nil {
				c.Logger().Error(err)
			}

			// Read
			msgType, msg, err := ws.ReadMessage()
			if err != nil {
				c.Logger().Error(err)
			}

			switch msgType {
			case 1, 2, 8, 9, 10:
				ll.Info().Int("msgtype", msgType).Str("msgtype-text", types[msgType]).Msg("incoming message type")
			default:
				ll.Warn().Int("msgtype", msgType).Msg("no idea what message type this is")
			}

			ll.Info().Msg(string(msg))
		}
	}
}

var types = map[int]string{
	1:  "text message",
	2:  "binary message",
	8:  "close message",
	9:  "ping message",
	10: "pong message",
}
