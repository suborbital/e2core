package runtimewasmer

import (
	"github.com/pkg/errors"
	"github.com/wasmerio/wasmer-go/wasmer"

	"github.com/suborbital/appspec/tenant"
	"github.com/suborbital/e2core/sat/api"
	"github.com/suborbital/e2core/sat/engine/runtime"
)

// WasmerBuilder is a Wasmer implementation of the instanceBuilder interface
type WasmerBuilder struct {
	ref     *tenant.WasmModuleRef
	hostFns []runtime.HostFn
	module  *wasmer.Module
	store   *wasmer.Store
	imports *wasmer.ImportObject
}

// NewBuilder creates a new WasmerBuilder
func NewBuilder(ref *tenant.WasmModuleRef, API api.HostAPI) runtime.RuntimeBuilder {
	w := &WasmerBuilder{
		ref:     ref,
		hostFns: API.HostFunctions(),
	}

	return w
}

func (w *WasmerBuilder) New() (runtime.RuntimeInstance, error) {
	module, _, imports, err := w.internals()
	if err != nil {
		return nil, errors.Wrap(err, "failed to ModuleBytes")
	}

	wasmerInst, err := wasmer.NewInstance(module, imports)
	if err != nil {
		return nil, errors.Wrap(err, "failed to NewInstance")
	}

	// if the module has exported a WASI start, call it
	wasiStart, err := wasmerInst.Exports.GetWasiStartFunction()
	if err == nil && wasiStart != nil {
		if _, err := wasiStart(); err != nil {
			return nil, errors.Wrap(err, "failed to wasiStart")
		}
	} else {
		// if the module has exported a _start function, call it
		_start, err := wasmerInst.Exports.GetFunction("_start")
		if err == nil && _start != nil {
			if _, err := _start(); err != nil {
				return nil, errors.Wrap(err, "failed to _start")
			}
		}
	}

	// if the module has exported an init function, call it
	init, err := wasmerInst.Exports.GetFunction("init")
	if err == nil && init != nil {
		if _, err := init(); err != nil {
			return nil, errors.Wrap(err, "failed to init")
		}
	}

	inst := &WasmerRuntime{
		inst: wasmerInst,
	}

	return inst, nil
}

func (w *WasmerBuilder) internals() (*wasmer.Module, *wasmer.Store, *wasmer.ImportObject, error) {
	if w.module == nil {
		engine := wasmer.NewEngine()
		store := wasmer.NewStore(engine)

		// Compiles the module
		mod, err := wasmer.NewModule(store, w.ref.Data)
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
		addHostFns(imports, store, w.hostFns...)

		w.module = mod
		w.store = store
		w.imports = imports
	}

	return w.module, w.store, w.imports, nil
}
