package main

import (
	"github.com/suborbital/reactr/api/tinygo/runnable"
	"github.com/suborbital/reactr/api/tinygo/runnable/resp"
)

type TinygoResp struct{}

func (h TinygoResp) Run(input []byte) ([]byte, error) {
	resp.SetHeader("X-Reactr", string(input))
	resp.ContentType("application/json")
	return []byte("Hello, " + string(input)), nil
}

// initialize runnable, do not edit //
func main() {
	runnable.Use(TinygoResp{})
}
