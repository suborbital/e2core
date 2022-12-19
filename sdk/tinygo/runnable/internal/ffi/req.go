//go:build tinygo.wasm

package ffi

// #include <reactr.h>
import "C"

func ReqGetField(fieldType int32, key string) []byte {
	ptr, size := rawSlicePointer([]byte(key))

	res, err := result(C.request_get_field(fieldType, ptr, size, Ident()))
	if err != nil {
		return []byte{}
	}
	return res
}

func ReqSetField(fieldType int32, key string, value string) ([]byte, error) {
	keyPtr, keySize := rawSlicePointer([]byte(key))
	valPtr, valSize := rawSlicePointer([]byte(value))

	return result(C.request_set_field(fieldType, keyPtr, keySize, valPtr, valSize, Ident()))
}
