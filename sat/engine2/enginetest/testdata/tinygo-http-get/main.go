package main

import (
	"github.com/suborbital/reactr/api/tinygo/runnable"
	"github.com/suborbital/reactr/api/tinygo/runnable/http"
	"github.com/suborbital/reactr/api/tinygo/runnable/log"
)

type TinygoHttpGet struct{}

func (h TinygoHttpGet) Run(input []byte) ([]byte, error) {
	headers := map[string]string{}
	headers["foo"] = "bar"

	res, err := http.POST(string(input), []byte("foobar"), headers)
	if err != nil {
		return nil, err
	}

	log.Info(string(res))

	return res, nil
}

// initialize runnable, do not edit //
func main() {
	runnable.Use(TinygoHttpGet{})
}
