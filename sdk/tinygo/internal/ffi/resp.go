//go:build tinygo.wasm

package ffi

// #include <plugin.h>
import "C"

func RespSetHeader(key, value string) {
	keyPtr, keySize := unsafeSlicePointer([]byte(key))
	valPtr, valSize := unsafeSlicePointer([]byte(value))

	C.resp_set_header(keyPtr, keySize, valPtr, valSize, Ident())
}
