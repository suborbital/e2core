//go:build tinygo.wasm

package ffi

// #include <plugin.h>
import "C"

func ReqGetField(fieldType int32, key string) []byte {
	ptr, size := unsafeSlicePointer([]byte(key))

	res, err := result(C.request_get_field(fieldType, ptr, size, Ident()))
	if err != nil {
		return []byte{}
	}
	return res
}

func ReqSetField(fieldType int32, key string, value string) ([]byte, error) {
	keyPtr, keySize := unsafeSlicePointer([]byte(key))
	valPtr, valSize := unsafeSlicePointer([]byte(value))

	return result(C.request_set_field(fieldType, keyPtr, keySize, valPtr, valSize, Ident()))
}
