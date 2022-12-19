package main

import (
	"github.com/suborbital/e2core/sdk/tinygo"
	"github.com/suborbital/e2core/sdk/tinygo/http"
	"github.com/suborbital/e2core/sdk/tinygo/log"
)

type TinygoHttpGet struct{}

func (h TinygoHttpGet) Run(input []byte) ([]byte, error) {
	res, err := http.GET(string(input), nil)
	if err != nil {
		return nil, err
	}

	log.Info(string(res))

	return res, nil
}

// initialize runnable, do not edit //
func main() {
	tinygo.Use(TinygoHttpGet{})
}
