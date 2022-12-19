//go:build tinygo.wasm

package ffi

import (
	"github.com/suborbital/reactr/api/tinygo/runnable/runnable"
)

var runnable_ runnable.Runnable
var ident_ int32

func Ident() int32 {
	return ident_
}

func Use(runnable runnable.Runnable) {
	runnable_ = runnable
}
