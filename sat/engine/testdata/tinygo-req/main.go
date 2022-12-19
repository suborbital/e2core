package main

import (
	"github.com/suborbital/e2core/sdk/tinygo"
	"github.com/suborbital/e2core/sdk/tinygo/log"
	"github.com/suborbital/e2core/sdk/tinygo/req"
)

type TinygoReq struct{}

func (h TinygoReq) Run(input []byte) ([]byte, error) {
	method := req.Method()
	url := req.URL()

	param := req.URLParam("foobar")

	log.Infof("%s: %s?%s", method, url, param)
	return []byte("Success"), nil
}

// initialize runnable, do not edit //
func main() {
	tinygo.Use(TinygoReq{})
}
