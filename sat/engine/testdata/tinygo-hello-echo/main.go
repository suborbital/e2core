package main

import "github.com/suborbital/e2core/sat/engine/runtime/api/tinygo/runnable"

type Hello struct{}

func (h Hello) Run(input []byte) ([]byte, error) {
	return []byte("Hello, " + string(input)), nil
}

func main() {
	runnable.Use(Hello{})
}
