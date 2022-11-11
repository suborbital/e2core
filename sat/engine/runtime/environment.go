package runtime

import (
	"sync"

	"github.com/google/uuid"
	"github.com/pkg/errors"

	"github.com/suborbital/e2core/scheduler"
	"github.com/suborbital/vektor/vlog"
)

/*
 In order to allow "easy" communication of data across the FFI barrier (outbound Go -> WASM and inbound WASM -> Go), engine provides
 an FFI API. Functions exported from a WASM module can be easily called by Go code via the Wasmer instance exports, but returning data
 to the host Go code is not quite as straightforward.

 In order to accomplish this, engine creates 'WasmEnvironments' which represent a single instantiated module. Each environment contains a pool of
 'wasmInstances', which are provided for use on a rotating basis. Instances can be added and removed from the pool as needed by the `wasmRunner`.

 When a WASM function calls one of the FFI API functions, it includes the `ident` value that was provided at the beginning
 of job execution, which allows engine to look up the wasmInstance from the global `instanceMapper` and send the result on
 the appropriate result channel. This is needed due to the way Go makes functions available to the FFI.
*/

// the internal Logger used by the Wasm runtime system
var internalLogger = vlog.Default()

// WasmEnvironment is an environment in which Wasm instances run
type WasmEnvironment struct {
	UUID    string
	builder RuntimeBuilder

	availableInstances chan *WasmInstance

	lock sync.RWMutex
}

// NewEnvironment creates a new environment with a pool of available wasmInstances
func NewEnvironment(builder RuntimeBuilder) *WasmEnvironment {
	e := &WasmEnvironment{
		UUID:               uuid.New().String(),
		builder:            builder,
		availableInstances: make(chan *WasmInstance, 64),
		lock:               sync.RWMutex{},
	}

	return e
}

// AddInstance adds a new Wasm instance to the environment's pool
func (w *WasmEnvironment) AddInstance() error {
	w.lock.Lock()
	defer w.lock.Unlock()

	inst, err := w.builder.New()
	if err != nil {
		return errors.Wrap(err, "failed to builder.New")
	}

	instance := &WasmInstance{
		runtime:    inst,
		resultChan: make(chan []byte, 1),
		errChan:    make(chan error, 1),
	}

	w.availableInstances <- instance

	return nil
}

// RemoveInstance removes one of the active instances from rotation and destroys it
func (w *WasmEnvironment) RemoveInstance() error {
	// grab an instance from the available queue
	// and we won't give it back becuase it's being destroyed
	inst := <-w.availableInstances

	// 4.
	inst.runtime.Close()
	inst.runtime = nil
	inst.ctx = nil
	inst.resultChan = nil
	inst.errChan = nil
	inst = nil

	return nil
}

// UseInstance provides an instance from the environment's pool to be used by a callback function
func (w *WasmEnvironment) UseInstance(ctx *scheduler.Ctx, instFunc func(*WasmInstance, int32)) error {
	// grab an instance from the available queue and then
	// return it to the environment when finished
	inst := <-w.availableInstances

	defer func() {
		w.availableInstances <- inst
	}()

	// generate a random identifier as a reference to the instance in use to
	// easily allow the Wasm module to reference itself when calling back over the FFI
	ident, err := setupNewIdentifier(inst)
	if err != nil {
		return errors.Wrap(err, "failed to setupNewIdentifier")
	}

	// setup the instance's temporary state
	inst.ctx = ctx

	// do the actual call into the Wasm module
	instFunc(inst, ident)

	// clear the instance's temporary state
	inst.ctx = nil

	// remove the instance from global state
	removeIdentifier(ident)

	return nil
}

// UseInternalLogger sets the logger to be used log internal wasm runtime messages
func UseInternalLogger(l *vlog.Logger) {
	internalLogger = l
}

func InternalLogger() *vlog.Logger {
	return internalLogger
}
