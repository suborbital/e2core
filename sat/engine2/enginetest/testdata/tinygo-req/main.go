package main

import (
	"github.com/suborbital/reactr/api/tinygo/runnable"
	"github.com/suborbital/reactr/api/tinygo/runnable/log"
	"github.com/suborbital/reactr/api/tinygo/runnable/req"
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
	runnable.Use(TinygoReq{})
}
