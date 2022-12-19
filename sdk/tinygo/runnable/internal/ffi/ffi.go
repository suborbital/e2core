//go:build tinygo.wasm

package ffi

// #include <reactr.h>
import "C"

func result(size int32) ([]byte, HostErr) {
	allocSize := size

	if size < 0 {
		if size == -1 {
			return nil, NewHostError("unknown error returned from host")
		}

		allocSize = -size
	}

	result := make([]byte, allocSize)
	resultPtr, _ := rawSlicePointer(result)

	if code := C.get_ffi_result(resultPtr, Ident()); code != 0 {
		return nil, NewHostError("unknown error returned from host")
	}

	if size < 0 {
		return nil, NewHostError(string(result))
	}

	return result, nil
}

func addVar(name, value string) {
	namePtr, nameSize := rawSlicePointer([]byte(name))
	valuePtr, valueSize := rawSlicePointer([]byte(value))

	C.add_ffi_var(namePtr, nameSize, valuePtr, valueSize, Ident())
}
