package runtime

import (
	"sync"

	"github.com/bytecodealliance/wasmtime-go/v9"
	"github.com/pkg/errors"

	"github.com/suborbital/e2core/foundation/scheduler"
	"github.com/suborbital/e2core/sat/engine2/api"
	"github.com/suborbital/e2core/sat/engine2/runtime/instance"
	"github.com/suborbital/systemspec/tenant"
)

var i32Type = wasmtime.NewValType(wasmtime.KindI32)

// InstancePool is a factory for Wasm instances
type InstancePool struct {
	availableInstances chan *instance.Instance

	ref     *tenant.WasmModuleRef
	hostFns []api.HostFn
	module  *wasmtime.Module
	engine  *wasmtime.Engine
	linker  *wasmtime.Linker
	lock    sync.RWMutex
}

// NewInstancePool creates a new InstancePool
func NewInstancePool(ref *tenant.WasmModuleRef, api api.HostAPI) *InstancePool {
	b := &InstancePool{
		availableInstances: make(chan *instance.Instance, 64),
		ref:                ref,
		hostFns:            api.HostFunctions(),
	}

	return b
}

// AddInstance adds a new Wasm instance to the environment's pool
func (ip *InstancePool) AddInstance() error {
	ip.lock.Lock()
	defer ip.lock.Unlock()

	inst, err := ip.new()
	if err != nil {
		return errors.Wrap(err, "failed to builder.New")
	}

	ip.availableInstances <- inst

	return nil
}

// RemoveInstance removes one of the active instances from rotation and destroys it
func (ip *InstancePool) RemoveInstance() error {
	// grab an instance from the available queue
	// and we won't give it back becuase it's being destroyed
	inst := <-ip.availableInstances

	inst.Close()
	inst = nil

	return nil
}

// UseInstance provides an instance from the environment's pool to be used by a callback function
func (ip *InstancePool) UseInstance(ctx *scheduler.Ctx, instFunc func(*instance.Instance, int32)) error {
	go func() {
		// prepare a new instance
		if err := ip.AddInstance(); err != nil {
			panic(err)
		}
	}()

	// grab an instance from the available queue
	inst := <-ip.availableInstances

	defer func(it *instance.Instance) {
		it.Close()
		it = nil
	}(inst)

	// generate a random identifier as a reference to the instance in use to
	// easily allow the Wasm module to reference itself when calling back over the FFI
	ident, err := instance.Store(inst)
	if err != nil {
		return errors.Wrap(err, "failed to setupNewIdentifier")
	}

	// setup the instance's temporary state
	inst.UseCtx(ctx)

	// do the actual call into the Wasm module
	instFunc(inst, ident)

	// clear the instance's temporary state
	inst.UseCtx(nil)

	// remove the instance from global state
	instance.Remove(ident)

	return nil
}

func (ip *InstancePool) new() (*instance.Instance, error) {
	module, engine, linker, err := ip.internals()
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

	inst := instance.New(wasmTimeInst, store)

	if _, err := inst.Call("_start"); err != nil {
		if errors.Is(err, instance.ErrExportNotFound) {
			// that's ok, not all modules will have _start
		} else {
			return nil, errors.Wrap(err, "failed to call exported _start")
		}
	}

	return inst, nil
}

func (ip *InstancePool) internals() (*wasmtime.Module, *wasmtime.Engine, *wasmtime.Linker, error) {
	if ip.module == nil {
		engine := wasmtime.NewEngine()

		// Compiles the module
		mod, err := wasmtime.NewModule(engine, ip.ref.Data)
		if err != nil {
			return nil, nil, nil, errors.Wrap(err, "failed to NewModule")
		}

		// Create a linker with WASI functions defined within it
		linker := wasmtime.NewLinker(engine)
		if err := linker.DefineWasi(); err != nil {
			return nil, nil, nil, errors.Wrap(err, "failed to DefineWasi")
		}

		// mount the module API
		addHostFns(linker, ip.hostFns...)

		ip.module = mod
		ip.engine = engine
		ip.linker = linker
	}

	return ip.module, ip.engine, ip.linker, nil
}

// addHostFns adds a list of host functions to an import object
func addHostFns(linker *wasmtime.Linker, fns ...api.HostFn) {
	for i := range fns {
		// we create a copy inside the loop otherwise things get overwritten
		fn := fns[i]

		// all function params are currently expressed as i32s, which will be improved upon
		// in the future with the introduction of witx-bindgen and/or interface types
		params := make([]*wasmtime.ValType, fn.ArgCount)
		for i := 0; i < fn.ArgCount; i++ {
			params[i] = i32Type
		}

		returns := []*wasmtime.ValType{}
		if fn.Returns {
			returns = append(returns, i32Type)
		}

		fnType := wasmtime.NewFuncType(params, returns)

		// this is reused across the normal and Swift variations of the function
		wasmtimeFunc := func(_ *wasmtime.Caller, args []wasmtime.Val) ([]wasmtime.Val, *wasmtime.Trap) {
			hostArgs := make([]interface{}, fn.ArgCount)

			// args can be longer than hostArgs (swift, lame), so use hostArgs to control the loop
			for i := range hostArgs {
				ha := args[i].I32()
				hostArgs[i] = ha
			}

			result, err := fn.HostFn(hostArgs...)
			if err != nil {
				return nil, wasmtime.NewTrap(errors.Wrapf(err, "failed to HostFn for %s", fn.Name).Error())
			}

			// function may return nothing, so nil check before trying to convert it
			returnVals := []wasmtime.Val{}
			if result != nil {
				returnVals = append(returnVals, wasmtime.ValI32(result.(int32)))
			}

			return returnVals, nil
		}

		// this can return an error but there's nothing we can do about it
		_ = linker.FuncNew("env", fn.Name, fnType, wasmtimeFunc)
	}
}
