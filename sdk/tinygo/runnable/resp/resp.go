//go:build tinygo.wasm

package resp

import (
	"github.com/suborbital/reactr/api/tinygo/runnable/internal/ffi"
)

func SetHeader(key, value string) {
	ffi.RespSetHeader(key, value)
}

func ContentType(contentType string) {
	SetHeader("Content-Type", contentType)
}
