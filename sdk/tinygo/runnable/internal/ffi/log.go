//go:build tinygo.wasm

package ffi

// #include <reactr.h>
import "C"

func LogAtLevel(message string, level int32) {
	msgPtr, size := rawSlicePointer([]byte(message))

	C.log_msg(msgPtr, size, level, Ident())
}
