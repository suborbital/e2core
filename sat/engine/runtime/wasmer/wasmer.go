package runtimewasmer

import (
	"github.com/pkg/errors"
	"github.com/wasmerio/wasmer-go/wasmer"
)

// WasmerRuntime is a Wasmer implementation of the runtimeInstance interface
type WasmerRuntime struct {
	inst *wasmer.Instance
}

func (w *WasmerRuntime) Call(fn string, args ...interface{}) (interface{}, error) {
	wasmFunc, err := w.inst.Exports.GetFunction(fn)
	if err != nil || wasmFunc == nil {
		return nil, errors.New("missing required FFI function: " + fn)
	}

	wasmResult, wasmErr := wasmFunc(args...)
	if wasmErr != nil {
		return nil, errors.Wrap(wasmErr, "failed to wasmFunc")
	}

	return wasmResult, nil
}

// ReadMemory reads memory from the instance
func (w *WasmerRuntime) ReadMemory(pointer int32, size int32) []byte {
	memory, err := w.inst.Exports.GetMemory("memory")
	if err != nil || memory == nil {
		// we failed
		return []byte{}
	}

	data := memory.Data()[pointer:]
	result := make([]byte, size)

	copy(result, data)

	return result
}

// WriteMemory writes memory into the instance
func (w *WasmerRuntime) WriteMemory(data []byte) (int32, error) {
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
func (w *WasmerRuntime) WriteMemoryAtLocation(pointer int32, data []byte) {
	memory, err := w.inst.Exports.GetMemory("memory")
	if err != nil || memory == nil {
		// we failed
		return
	}

	scopedMemory := memory.Data()[pointer:]

	copy(scopedMemory, data)
}

// Deallocate deallocates memory in the instance
func (w *WasmerRuntime) Deallocate(pointer int32, length int) {
	w.Call("deallocate", pointer, length)
}

// Close closes the instance
func (w *WasmerRuntime) Close() {
	w.inst.Close()
}
