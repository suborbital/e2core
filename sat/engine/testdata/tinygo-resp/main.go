package main

import (
	"github.com/suborbital/e2core/sdk/tinygo"

	"github.com/suborbital/e2core/sdk/tinygo/resp"
)

type TinygoResp struct{}

func (h TinygoResp) Run(input []byte) ([]byte, error) {
	resp.SetHeader("X-Reactr", string(input))
	resp.ContentType("application/json")
	return []byte("Hello, " + string(input)), nil
}

// initialize runnable, do not edit //
func main() {
	tinygo.Use(TinygoResp{})
}
