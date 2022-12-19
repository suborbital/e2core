//go:build tinygo.wasm

package ffi

// #include <reactr.h>
import "C"

func GetStaticFile(filename string) ([]byte, error) {
	ptr, size := rawSlicePointer([]byte(filename))

	return result(C.get_static_file(ptr, size, Ident()))
}
