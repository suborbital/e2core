//go:build wasmedge
// +build wasmedge

// we compile-exclude wasmedge by default as tests will fail unless WasmEdge is installed

package runtimewasmedge

import (
	"github.com/pkg/errors"
	"github.com/second-state/WasmEdge-go/wasmedge"
)

// WasmEdgeRuntime is a WasmEdge implementation of the runtimeInstance interface
type WasmEdgeRuntime struct {
	imports  *wasmedge.ImportObject
	store    *wasmedge.Store
	executor *wasmedge.Executor
}

func (w *WasmEdgeRuntime) Call(fn string, args ...interface{}) (interface{}, error) {
	wasmResult, wasmErr := w.executor.Invoke(w.store, fn, args...)
	if wasmErr != nil {
		return nil, errors.Wrap(wasmErr, "failed to execute wasm func")
	}

	if len(wasmResult) == 0 {
		return nil, nil
	} else {
		return wasmResult[0], nil
	}
}

// ReadMemory reads memory from the vm
func (w *WasmEdgeRuntime) ReadMemory(pointer int32, size int32) []byte {
	memory := w.store.FindMemory("memory")
	if memory == nil {
		return []byte{}
	}

	data, err := memory.GetData(uint(pointer), uint(size))
	if err != nil || data == nil {
		return []byte{}
	}

	result := make([]byte, size)

	copy(result, data)

	return result
}

// WriteMemory writes memory into the instance
func (w *WasmEdgeRuntime) WriteMemory(data []byte) (int32, error) {
	lengthOfInput := len(data)
	allocateResult, err := w.Call("allocate", int32(lengthOfInput))
	if err != nil {
		return 0, errors.Wrap(err, "failed to Call allocate")
	}

	pointer := allocateResult.(int32)

	w.WriteMemoryAtLocation(pointer, data)

	return pointer, nil
}

// WriteMemoryAtLocation writes memory at the given location
func (w *WasmEdgeRuntime) WriteMemoryAtLocation(pointer int32, data []byte) {
	memory := w.store.FindMemory("memory")
	if memory == nil {
		return
	}

	memory.SetData(data, uint(pointer), uint(len(data)))
}

// Deallocate deallocates memory in the instance
func (w *WasmEdgeRuntime) Deallocate(pointer int32, length int) {
	w.Call("deallocate", pointer, int32(length))
}

// Close closes the instance
func (w *WasmEdgeRuntime) Close() {
	w.executor.Release()
	w.store.Release()
	w.imports.Release()
}
