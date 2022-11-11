package main

import (
	"github.com/suborbital/e2core/sat/api/tinygo/runnable"
	"github.com/suborbital/e2core/sat/api/tinygo/runnable/file"
)

type TinygoGetStatic struct{}

func (h TinygoGetStatic) Run(input []byte) ([]byte, error) {
	return file.Bytes("important.md")
}

// initialize runnable, do not edit //
func main() {
	runnable.Use(TinygoGetStatic{})
}
