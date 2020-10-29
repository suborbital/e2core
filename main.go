package main

import (
	"log"

	"github.com/suborbital/atmo/atmo"
)

func main() {
	server := atmo.New()

	if err := server.Start("./runnables.wasm.zip"); err != nil {
		log.Fatal(err)
	}
}
