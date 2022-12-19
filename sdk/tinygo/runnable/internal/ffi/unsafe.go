//go:build tinygo.wasm

package ffi

import (
	"reflect"
	"runtime"
	"unsafe"
)

//export allocate
func allocate(size int32) uintptr {
	arr := make([]byte, size)

	header := (*reflect.SliceHeader)(unsafe.Pointer(&arr))

	runtime.KeepAlive(arr)

	return uintptr(header.Data)
}

//export deallocate
func deallocate(pointer uintptr, size int32) {
	var arr []byte

	header := (*reflect.SliceHeader)(unsafe.Pointer(&arr))
	header.Data = pointer
	header.Len = uintptr(size) // Interestingly, the types of .Len and .Cap here
	header.Cap = uintptr(size) // differ from standard Go, where they are both int

	arr = nil // I think this is sufficient to mark the slice for garbage collection
}

func rawSlicePointer(slice []byte) (unsafe.Pointer, int32) {
	header := (*reflect.SliceHeader)(unsafe.Pointer(&slice))

	return unsafe.Pointer(header.Data), int32(len(slice))
}

func unsafeSlice(size int) ([]byte, uintptr) {
	slice := make([]byte, size)
	header := (*reflect.SliceHeader)(unsafe.Pointer(&slice))

	runtime.KeepAlive(slice)
	ptr := unsafe.Pointer(header.Data)

	return slice, uintptr(ptr)
}
