package rwasm

import (
	"github.com/pkg/errors"
	"github.com/suborbital/reactr/rt"
	"github.com/wasmerio/wasmer-go/wasmer"
)

type wasmInstance struct {
	wasmerInst *wasmer.Instance

	ctx *rt.Ctx

	ffiResult []byte

	resultChan chan []byte
	errChan    chan rt.RunErr
}

// instanceReference holds a reference to a particular wasmInstance
type instanceReference struct {
	Inst *wasmInstance
}

/////////////////////////////////////////////////////////////////////////////
// below is the wasm glue code used to manipulate wasm instance memory     //
// this requires a set of functions to be available within the wasm module //
// - allocate                                                              //
// - deallocate                                                            //
/////////////////////////////////////////////////////////////////////////////

func (w *wasmInstance) setFFIResult(data []byte) error {
	if w.ffiResult != nil {
		return errors.New("instance ffiResult is already set")
	}

	w.ffiResult = data

	return nil
}

func (w *wasmInstance) useFFIResult() ([]byte, error) {
	if w.ffiResult == nil {
		return nil, errors.New("instance ffiResult is not set")
	}

	defer func() {
		w.ffiResult = nil
	}()

	return w.ffiResult, nil
}

func (w *wasmInstance) readMemory(pointer int32, size int32) []byte {
	memory, err := w.wasmerInst.Exports.GetMemory("memory")
	if err != nil || memory == nil {
		// we failed
		return []byte{}
	}

	data := memory.Data()[pointer:]
	result := make([]byte, size)

	for index := 0; int32(index) < size; index++ {
		result[index] = data[index]
	}

	return result
}

func (w *wasmInstance) writeMemory(data []byte) (int32, error) {
	lengthOfInput := len(data)

	allocate, err := w.wasmerInst.Exports.GetFunction("allocate")
	if err != nil || allocate == nil {
		return -1, errors.New("missing required FFI function: allocate")
	}

	// Allocate memory for the input, and get a pointer to it.
	allocateResult, err := allocate(lengthOfInput)
	if err != nil {
		return -1, errors.Wrap(err, "failed to call allocate")
	}

	pointer := allocateResult.(int32)

	w.writeMemoryAtLocation(pointer, data)

	return pointer, nil
}

func (w *wasmInstance) writeMemoryAtLocation(pointer int32, data []byte) {
	memory, err := w.wasmerInst.Exports.GetMemory("memory")
	if err != nil || memory == nil {
		// we failed
		return
	}

	scopedMemory := memory.Data()[pointer:]

	copy(scopedMemory, data)
}

func (w *wasmInstance) deallocate(pointer int32, length int) {
	dealloc, err := w.wasmerInst.Exports.GetFunction("deallocate")
	if err != nil || dealloc == nil {
		// we failed
		return
	}

	dealloc(pointer, length)
}
