package instance

import (
	"github.com/bytecodealliance/wasmtime-go/v7"
	"github.com/pkg/errors"

	"github.com/suborbital/e2core/foundation/scheduler"
)

var ErrExportNotFound = errors.New("the requested export is not found in the module")

type Instance struct {
	inst  *wasmtime.Instance
	store *wasmtime.Store

	ctx *scheduler.Ctx

	resultChan chan []byte
	errChan    chan error
}

// New creates a new Instance wrapper
func New(inst *wasmtime.Instance, store *wasmtime.Store) *Instance {
	i := &Instance{
		inst:       inst,
		store:      store,
		resultChan: make(chan []byte, 1),
		errChan:    make(chan error, 1),
	}

	return i
}

func (w *Instance) Call(fn string, args ...interface{}) (interface{}, error) {
	wasmFunc := w.inst.GetExport(w.store, fn)

	if wasmFunc == nil {
		return nil, errors.Wrapf(ErrExportNotFound, "function %s not found", fn)
	}

	wasmResult, wasmErr := wasmFunc.Func().Call(w.store, args...)
	if wasmErr != nil {
		return nil, errors.Wrap(wasmErr, "failed to wasmFunc")
	}

	return wasmResult, nil
}

// ReadMemory reads memory from the instance
func (w *Instance) ReadMemory(pointer int32, size int32) []byte {
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
func (w *Instance) WriteMemory(data []byte) (int32, error) {
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
func (w *Instance) WriteMemoryAtLocation(pointer int32, data []byte) {
	memory := w.inst.GetExport(w.store, "memory").Memory()

	if memory == nil {
		// we failed
		return
	}

	scopedMemory := memory.UnsafeData(w.store)[pointer:]

	copy(scopedMemory, data)
}

// Deallocate deallocates memory in the instance
func (w *Instance) Deallocate(pointer int32, length int) {
	w.Call("deallocate", pointer, length)
}

// ExecutionResult gets the module's execution results
func (w *Instance) ExecutionResult() ([]byte, error) {
	// determine if the instance called return_result or return_error
	select {
	case res := <-w.resultChan:
		return res, nil
	case err := <-w.errChan:
		return nil, err
	default:
		// do nothing and fall through
	}

	return nil, nil
}

// SendExecutionResult allows FFI functions to send the run result
func (w *Instance) SendExecutionResult(result []byte, runErr error) {
	if runErr != nil {
		w.errChan <- runErr
	} else if result != nil {
		w.resultChan <- result
	}
}

// Ctx returns the internal Ctx
func (w *Instance) Ctx() *scheduler.Ctx {
	return w.ctx
}

// UseCtx sets the internal Ctx
func (w *Instance) UseCtx(ctx *scheduler.Ctx) {
	w.ctx = ctx
}

func (w *Instance) Close() {
	w.inst = nil
	w.ctx = nil
	w.store = nil
	w.resultChan = nil
	w.errChan = nil
}
