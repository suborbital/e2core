package main

import (
	"fmt"
	"log"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"github.com/suborbital/subo/subo/input"
)

func main() {
	conn, _, err := websocket.DefaultDialer.Dial("ws://localhost:8080/stream", nil)
	if err != nil {
		log.Fatal(err)
	}

	for {
		fmt.Print("Enter a message: ")
		message, err := input.ReadStdinString()
		if err != nil {
			log.Fatal(errors.Wrap(err, "failed to ReadStdinString"))
		}

		if err := conn.WriteMessage(websocket.TextMessage, []byte(message)); err != nil {
			log.Fatal(err)
		}

		_, response, err := conn.ReadMessage()
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println(string(response))
	}

}
