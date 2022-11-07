package runtimewasmtime

import (
	"github.com/bytecodealliance/wasmtime-go"
	"github.com/pkg/errors"

	"github.com/suborbital/e2core/sat/engine/runtime"
)

type WasmtimeInstance struct {
	inst  wasmtime.Instance
	store *wasmtime.Store
}

func (w *WasmtimeInstance) Call(fn string, args ...interface{}) (interface{}, error) {
	wasmFunc := w.inst.GetExport(w.store, fn)

	if wasmFunc == nil {
		return nil, errors.Wrapf(runtime.ErrExportNotFound, "function %s not found", fn)
	}

	wasmResult, wasmErr := wasmFunc.Func().Call(w.store, args...)
	if wasmErr != nil {
		return nil, errors.Wrap(wasmErr, "failed to wasmFunc")
	}

	return wasmResult, nil
}

// ReadMemory reads memory from the instance
func (w *WasmtimeInstance) ReadMemory(pointer int32, size int32) []byte {
	memory := w.inst.GetExport(w.store, "memory").Memory()

	if memory == nil {
		// we failed
		return []byte{}
	}

	data := memory.UnsafeData(w.store)[pointer:]
	result := make([]byte, size)

	copy(result, data)

	return result
}

// WriteMemory writes memory into the instance
func (w *WasmtimeInstance) WriteMemory(data []byte) (int32, error) {
	lengthOfInput := len(data)

	allocateResult, err := w.Call("allocate", lengthOfInput)
	if err != nil {
		return 0, errors.Wrap(err, "failed to Call allocate")
	}

	pointer := allocateResult.(int32)

	w.WriteMemoryAtLocation(pointer, data)

	return pointer, nil
}

// WriteMemoryAtLocation writes memory at the given location
func (w *WasmtimeInstance) WriteMemoryAtLocation(pointer int32, data []byte) {
	memory := w.inst.GetExport(w.store, "memory").Memory()

	if memory == nil {
		// we failed
		return
	}

	scopedMemory := memory.UnsafeData(w.store)[pointer:]

	copy(scopedMemory, data)
}

// Deallocate deallocates memory in the instance
func (w *WasmtimeInstance) Deallocate(pointer int32, length int) {
	w.Call("deallocate", pointer, length)
}

// Close closes the instance
func (w *WasmtimeInstance) Close() {
	// Wasmtime relies on golang garbage collector to clean up cgo allocations.
	// This makes the API simpler as you don't need to explicitly close anything.
	//
	// See also:
	// https://github.com/bytecodealliance/wasmtime-go/blob/main/ffi.go
}
