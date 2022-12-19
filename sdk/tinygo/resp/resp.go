//go:build tinygo.wasm

package resp

import (
	"github.com/suborbital/e2core/sdk/tinygo/internal/ffi"
)

func SetHeader(key, value string) {
	ffi.RespSetHeader(key, value)
}

func ContentType(contentType string) {
	SetHeader("Content-Type", contentType)
}
