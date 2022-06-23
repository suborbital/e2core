package main

import (
	"fmt"
	"log"
	"time"

	"github.com/pkg/errors"
	"github.com/suborbital/vektor/vlog"
	"github.com/suborbital/velocity/bus/bus"
	"github.com/suborbital/velocity/bus/transport/nats"
)

func main() {
	logger := vlog.Default(vlog.Level(vlog.LogLevelDebug))

	gnats, err := nats.New("nats://localhost:4222")
	if err != nil {
		log.Fatal(errors.Wrap(err, "failed to nats.New"))
	}

	g := bus.New(
		bus.UseLogger(logger),
		bus.UseBridgeTransport(gnats),
	)

	if err := g.ConnectBridgeTopic(bus.MsgTypeDefault); err != nil {
		log.Fatal(errors.Wrap(err, "failed to ConnectTopic"))
	}

	if err := g.ConnectBridgeTopic("bus.reply"); err != nil {
		log.Fatal(errors.Wrap(err, "failed to ConnectTopic"))
	}

	pod := g.Connect()
	pod.On(func(msg bus.Message) error {
		fmt.Println("received something:", string(msg.Data()), msg.Type())
		return nil
	})

	go func() {
		<-time.After(time.Second * time.Duration(5))
		fmt.Println("sending 1")
		pod.Send(bus.NewMsg(bus.MsgTypeDefault, []byte("world")))

		<-time.After(time.Second * time.Duration(5))
		fmt.Println("sending 2")
		pod.Send(bus.NewMsg(bus.MsgTypeDefault, []byte("again")))
	}()

	<-time.After(time.Minute)
}
