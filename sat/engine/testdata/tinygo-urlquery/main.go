package main

import (
	"github.com/suborbital/e2core/sdk/tinygo"
	"github.com/suborbital/e2core/sdk/tinygo/req"
)

type TinygoQueryparam struct{}

func (h TinygoQueryparam) Run(input []byte) ([]byte, error) {
	message := req.QueryParam("message")

	return []byte("hello " + message), nil
}

// initialize runnable, do not edit //
func main() {
	tinygo.Use(TinygoQueryparam{})
}
