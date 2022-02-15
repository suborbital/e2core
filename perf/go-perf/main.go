package main

import (
	"github.com/suborbital/reactr/api/tinygo/runnable" 
)

type GoPerf struct{}

func (h GoPerf) Run(input []byte) ([]byte, error) {
	return []byte("Hello, " + string(input)), nil
}

// initialize runnable, do not edit //
func main() {
	runnable.Use(GoPerf{})
}
