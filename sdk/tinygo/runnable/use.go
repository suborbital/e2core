//go:build tinygo.wasm

package runnable

import (
	"github.com/suborbital/reactr/api/tinygo/runnable/internal/ffi"
	"github.com/suborbital/reactr/api/tinygo/runnable/runnable"
)

func Use(runnable runnable.Runnable) {
	ffi.Use(runnable)
}
