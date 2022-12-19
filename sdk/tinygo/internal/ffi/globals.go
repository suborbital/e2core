//go:build tinygo.wasm

package ffi

import (
	"github.com/suborbital/e2core/sdk/tinygo/plugin"
)

var plugin_ plugin.Plugin
var ident_ int32

func Ident() int32 {
	return ident_
}

func Use(plugin plugin.Plugin) {
	plugin_ = plugin
}
