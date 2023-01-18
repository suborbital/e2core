package main

import (
	"github.com/suborbital/reactr/api/tinygo/runnable"
	"github.com/suborbital/reactr/api/tinygo/runnable/req"
)

type TinygoQueryparam struct{}

func (h TinygoQueryparam) Run(input []byte) ([]byte, error) {
	message := req.QueryParam("message")

	return []byte("hello " + message), nil
}

// initialize runnable, do not edit //
func main() {
	runnable.Use(TinygoQueryparam{})
}
