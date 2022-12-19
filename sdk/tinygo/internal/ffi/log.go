//go:build tinygo.wasm

package ffi

// #include <plugin.h>
import "C"

func LogAtLevel(message string, level int32) {
	msgPtr, size := unsafeSlicePointer([]byte(message))

	C.log_msg(msgPtr, size, level, Ident())
}
