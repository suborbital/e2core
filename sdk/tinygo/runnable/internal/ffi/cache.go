//go:build tinygo.wasm

package ffi

// #include <reactr.h>
import "C"

func CacheGet(key string) ([]byte, error) {
	ptr, size := rawSlicePointer([]byte(key))

	return result(C.cache_get(ptr, size, Ident()))
}

func CacheSet(key, val string, ttl int) {
	keyPtr, keySize := rawSlicePointer([]byte(key))
	valPtr, valSize := rawSlicePointer([]byte(val))

	C.cache_set(keyPtr, keySize, valPtr, valSize, int32(ttl), Ident())
	return
}
