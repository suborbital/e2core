package main

import (
	"fmt"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

func main() {
	conn, _, err := websocket.DefaultDialer.Dial("ws://localhost:8080/stream", nil)
	if err != nil {
		log.Fatal(err)
	}

	for {
		fmt.Print("Enter a message: ")
		message := "world"

		// uncomment to make it interactive, don't want to import Subo into Atmo otherwise
		// message, err := input.ReadStdinString()
		// if err != nil {
		// 	log.Fatal(errors.Wrap(err, "failed to ReadStdinString"))
		// }

		if err := conn.WriteMessage(websocket.TextMessage, []byte(message)); err != nil {
			log.Fatal(err)
		}

		_, response, err := conn.ReadMessage()
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println(string(response))

		time.Sleep(time.Second * 3)
	}

}
