//go:build tinygo.wasm

package method

type MethodType int32

const (
	GET MethodType = iota
	HEAD
	OPTIONS
	POST
	PUT
	PATCH
	DELETE
)
