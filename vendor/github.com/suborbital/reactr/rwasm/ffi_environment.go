package rwasm

import (
	"sync"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/suborbital/reactr/rt"
	"github.com/suborbital/reactr/rwasm/moduleref"
	"github.com/suborbital/vektor/vlog"
	"github.com/wasmerio/wasmer-go/wasmer"
)

/*
 In order to allow "easy" communication of data across the FFI barrier (outbound Go -> WASM and inbound WASM -> Go), rwasm provides
 an FFI API. Functions exported from a WASM module can be easily called by Go code via the Wasmer instance exports, but returning data
 to the host Go code is not quite as straightforward.

 In order to accomplish this, rwasm creates 'wasmEnvironments' which represent a single instantiated module. Each environment contains a pool of
 'wasmInstances', which are provided for use on a rotating basis. Instances can be added and removed from the pool as needed by the `wasmRunner`.

 When a WASM function calls one of the FFI API functions, it includes the `ident` value that was provided at the beginning
 of job execution, which allows rwasm to look up the wasmInstance from the global `instanceMapper` and send the result on
 the appropriate result channel. This is needed due to the way Go makes functions available to the FFI.
*/

// the internal Logger used by the Wasm runtime system
var internalLogger = vlog.Default()

// wasmEnvironment is an environment in which Wasm instances run
type wasmEnvironment struct {
	UUID    string
	ref     *moduleref.WasmModuleRef
	module  *wasmer.Module
	store   *wasmer.Store
	imports *wasmer.ImportObject

	availableInstances chan *wasmInstance

	lock sync.RWMutex
}

// newEnvironment creates a new environment with a pool of available wasmInstances
func newEnvironment(ref *moduleref.WasmModuleRef) *wasmEnvironment {
	e := &wasmEnvironment{
		UUID:               uuid.New().String(),
		ref:                ref,
		availableInstances: make(chan *wasmInstance, 64),
		lock:               sync.RWMutex{},
	}

	return e
}

// addInstance adds a new Wasm instance to the environment's pool
func (w *wasmEnvironment) addInstance() error {
	w.lock.Lock()
	defer w.lock.Unlock()

	module, _, imports, err := w.internals()
	if err != nil {
		return errors.Wrap(err, "failed to ModuleBytes")
	}

	inst, err := wasmer.NewInstance(module, imports)
	if err != nil {
		return errors.Wrap(err, "failed to NewInstance")
	}

	// if the module has exported a WASI start, call it
	wasiStart, err := inst.Exports.GetWasiStartFunction()
	if err == nil && wasiStart != nil {
		if _, err := wasiStart(); err != nil {
			return errors.Wrap(err, "failed to wasiStart")
		}
	} else {
		// if the module has exported a _start function, call it
		_start, err := inst.Exports.GetFunction("_start")
		if err == nil && _start != nil {
			if _, err := _start(); err != nil {
				return errors.Wrap(err, "failed to _start")
			}
		}
	}

	// if the module has exported an init function, call it
	init, err := inst.Exports.GetFunction("init")
	if err == nil && init != nil {
		if _, err := init(); err != nil {
			return errors.Wrap(err, "failed to init")
		}
	}

	instance := &wasmInstance{
		wasmerInst: inst,
		resultChan: make(chan []byte, 1),
		errChan:    make(chan rt.RunErr, 1),
	}

	w.availableInstances <- instance

	return nil
}

func (w *wasmEnvironment) removeInstance() error {
	// grab an instance from the available queue
	// and we won't give it back becuase it's being destroyed
	inst := <-w.availableInstances

	// 4.
	inst.wasmerInst.Close()
	inst.wasmerInst = nil
	inst.ctx = nil
	inst.ffiResult = nil
	inst.resultChan = nil
	inst.errChan = nil
	inst = nil

	return nil
}

// useInstance provides an instance from the environment's pool to be used by a callback function
func (w *wasmEnvironment) useInstance(ctx *rt.Ctx, instFunc func(*wasmInstance, int32)) error {
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
	inst.ffiResult = nil
	inst.ctx = ctx

	// do the actual call into the Wasm module
	instFunc(inst, ident)

	// clear the instance's temporary state
	inst.ctx = nil
	inst.ffiResult = nil

	// remove the instance from global state
	removeIdentifier(ident)

	return nil
}

func (w *wasmEnvironment) internals() (*wasmer.Module, *wasmer.Store, *wasmer.ImportObject, error) {
	if w.module == nil {
		moduleBytes, err := w.ref.Bytes()
		if err != nil {
			return nil, nil, nil, errors.Wrap(err, "failed to get ref ModuleBytes")
		}

		engine := wasmer.NewEngine()
		store := wasmer.NewStore(engine)

		// Compiles the module
		mod, err := wasmer.NewModule(store, moduleBytes)
		if err != nil {
			return nil, nil, nil, errors.Wrap(err, "failed to NewModule")
		}

		env, err := wasmer.NewWasiStateBuilder(w.ref.Name).Finalize()
		if err != nil {
			return nil, nil, nil, errors.Wrap(err, "failed to NewWasiStateBuilder.Finalize")
		}

		imports, err := env.GenerateImportObject(store, mod)
		if err != nil {
			imports = wasmer.NewImportObject() // for now, defaulting to creating non-WASI imports if there's a failure.
		}

		// mount the Runnable API host functions to the module's imports
		addHostFns(imports, store,
			returnResult(),
			returnError(),
			getFFIResult(),
			fetchURL(),
			graphQLQuery(),
			cacheSet(),
			cacheGet(),
			logMsg(),
			requestGetField(),
			respSetHeader(),
			getStaticFile(),
			abortHandler(),
		)

		w.module = mod
		w.store = store
		w.imports = imports
	}

	return w.module, w.store, w.imports, nil
}

// UseInternalLogger sets the logger to be used log internal wasm runtime messages
func UseInternalLogger(l *vlog.Logger) {
	internalLogger = l
}
