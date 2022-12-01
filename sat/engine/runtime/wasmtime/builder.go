package wasmtime

import (
	"github.com/bytecodealliance/wasmtime-go"
	"github.com/pkg/errors"

	"github.com/suborbital/e2core/sat/engine/runtime"
	"github.com/suborbital/e2core/sat/engine/runtime/api"
	"github.com/suborbital/systemspec/tenant"
)

// InstanceBuilder is a Wasmer implementation of the instanceBuilder interface
type InstanceBuilder struct {
	ref     *tenant.WasmModuleRef
	hostFns []runtime.HostFn
	module  *wasmtime.Module
	engine  *wasmtime.Engine
	linker  *wasmtime.Linker
}

// NewBuilder creates a new InstanceBuilder
func NewBuilder(ref *tenant.WasmModuleRef, api api.HostAPI) *InstanceBuilder {
	b := &InstanceBuilder{
		ref:     ref,
		hostFns: api.HostFunctions(),
	}

	return b
}

func (w *InstanceBuilder) New() (runtime.RuntimeInstance, error) {
	module, engine, linker, err := w.internals()
	if err != nil {
		return nil, errors.Wrap(err, "failed to internals")
	}

	store := wasmtime.NewStore(engine)

	wasiConfig := wasmtime.NewWasiConfig()
	store.SetWasi(wasiConfig)

	wasmTimeInst, err := linker.Instantiate(store, module)
	if err != nil {
		return nil, errors.Wrap(err, "failed to linker.Instantiate")
	}

	inst := &Instance{
		inst:  *wasmTimeInst,
		store: store,
	}

	if _, err := inst.Call("_start"); err != nil {
		if errors.Is(err, runtime.ErrExportNotFound) {
			// that's ok, not all modules will have _start
		} else {
			return nil, errors.Wrap(err, "failed to call exported _start")
		}
	}

	// the deprecated `init` is not used in the Wasmtime runtime

	return inst, nil
}

func (w *InstanceBuilder) internals() (*wasmtime.Module, *wasmtime.Engine, *wasmtime.Linker, error) {
	if w.module == nil {
		engine := wasmtime.NewEngine()

		// Compiles the module
		mod, err := wasmtime.NewModule(engine, w.ref.Data)
		if err != nil {
			return nil, nil, nil, errors.Wrap(err, "failed to NewModule")
		}

		// Create a linker with WASI functions defined within it
		linker := wasmtime.NewLinker(engine)
		if err := linker.DefineWasi(); err != nil {
			return nil, nil, nil, errors.Wrap(err, "failed to DefineWasi")
		}

		// mount the module API
		addHostFns(linker, w.hostFns...)

		w.module = mod
		w.engine = engine
		w.linker = linker
	}

	return w.module, w.engine, w.linker, nil
}
