//go:build tinygo.wasm

package tinygo

import (
	"github.com/suborbital/e2core/sdk/tinygo/internal/ffi"
	"github.com/suborbital/e2core/sdk/tinygo/plugin"
)

func Use(plugin plugin.Plugin) {
	ffi.Use(plugin)
}
