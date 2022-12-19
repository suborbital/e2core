package main

import (
	"github.com/suborbital/e2core/sdk/tinygo"
	"github.com/suborbital/e2core/sdk/tinygo/log"
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
	tinygo.Use(TinygoLog{})
}
