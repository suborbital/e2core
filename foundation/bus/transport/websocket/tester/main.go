package main

import (
	"fmt"
	"os"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"

	"github.com/suborbital/e2core/foundation/bus/bus"
	"github.com/suborbital/e2core/foundation/bus/discovery/local"
	"github.com/suborbital/e2core/foundation/bus/transport/websocket"
)

func main() {
	logger := zerolog.New(os.Stderr).With().Str("mode", "websocket-tester").Timestamp().Logger()
	gwss := websocket.New()
	locald := local.New()

	port := os.Getenv("ECHO_HTTP_PORT")

	g := bus.New(
		bus.UseLogger(logger),
		bus.UseEndpoint(port, "/meta/message"),
		bus.UseMeshTransport(gwss),
		bus.UseDiscovery(locald),
	)

	pod := g.Connect()
	pod.On(func(msg bus.Message) error {
		fmt.Println("received something:", string(msg.Data()))
		return nil
	})

	e := echo.New()
	e.GET("/meta/message", echo.WrapHandler(gwss.HTTPHandlerFunc()))

	go func() {
		<-time.After(time.Second * time.Duration(5))
		pod.Send(bus.NewMsg(bus.MsgTypeDefault, []byte("hello, world")))

		<-time.After(time.Second * time.Duration(5))
		pod.Send(bus.NewMsg(bus.MsgTypeDefault, []byte("hello, again")))

		<-time.After(time.Second * time.Duration(5))
		if err := g.Withdraw(); err != nil {
			logger.Err(err).Msg("failed to withdraw")
			os.Exit(1)
		}

		if err := g.Stop(); err != nil {
			logger.Err(err).Msg("failed to stop")
			os.Exit(1)
		}

		logger.Debug().Msg("withdrawn and stopped")
		os.Exit(0)
	}()

	if err := e.Start(fmt.Sprintf(":%s", port)); err != nil {
		logger.Err(err).Msgf("echo.start on port %s", port)
	}
}
