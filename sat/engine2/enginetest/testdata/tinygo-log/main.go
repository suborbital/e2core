package main

import (
	"github.com/suborbital/reactr/api/tinygo/runnable"
	"github.com/suborbital/reactr/api/tinygo/runnable/log"
)

type TinygoLog struct{}

func (h TinygoLog) Run(input []byte) ([]byte, error) {
	log.Info(string(input))
	log.Info("info log")
	log.Error("some error")

	warnMsg := "warning message"
	log.Warnf("some %s", warnMsg)

	log.Debug("debug message")

	return []byte(""), nil
}

// initialize runnable, do not edit //
func main() {
	runnable.Use(TinygoLog{})
}
