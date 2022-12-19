//go:build tinygo.wasm

package ffi

// #include <reactr.h>
import "C"

func RespSetHeader(key, value string) {
	keyPtr, keySize := rawSlicePointer([]byte(key))
	valPtr, valSize := rawSlicePointer([]byte(value))

	C.resp_set_header(keyPtr, keySize, valPtr, valSize, Ident())
}
