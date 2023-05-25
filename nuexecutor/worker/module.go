package worker

import (
	"github.com/bytecodealliance/wasmtime-go/v7"
	"github.com/pkg/errors"

	"github.com/suborbital/e2core/sat/engine2/api"
	"github.com/suborbital/e2core/sat/engine2/runtime/instance"
)

var i32Type = wasmtime.NewValType(wasmtime.KindI32)

func buildModule(wasmBytes []byte, hostFns []api.HostFn) (*instance.Instance, error) {
	engine := wasmtime.NewEngine()

	mod, err := wasmtime.NewModule(engine, wasmBytes)
	if err != nil {
		return nil, errors.Wrap(err, "failed to NewModule")
	}

	// Create a linker with WASI functions defined within it
	linker := wasmtime.NewLinker(engine)
	if err := linker.DefineWasi(); err != nil {
		return nil, errors.Wrap(err, "failed to DefineWasi")
	}

	err = addHostFns(linker, hostFns)
	if err != nil {
		return nil, errors.Wrap(err, "addHostFns")
	}

	store := wasmtime.NewStore(engine)
	wasiConfig := wasmtime.NewWasiConfig()
	store.SetWasi(wasiConfig)

	wasmTimeInst, err := linker.Instantiate(store, mod)
	if err != nil {
		return nil, errors.Wrap(err, "linker.Instantiate")
	}

	inst := instance.New(wasmTimeInst, store)

	if _, err := inst.Call("_start"); err != nil {
		if !errors.Is(err, instance.ErrExportNotFound) {
			return nil, errors.Wrap(err, "failed to call exported _start")
		}

		// that's ok, not all modules will have _start
	}

	return inst, nil
}

// addHostFns adds a list of host functions to an import object
func addHostFns(linker *wasmtime.Linker, fns []api.HostFn) error {
	for i := range fns {
		// we create a copy inside the loop otherwise things get overwritten
		fn := fns[i]

		// all function params are currently expressed as i32s, which will be improved upon
		// in the future with the introduction of witx-bindgen and/or interface types
		params := make([]*wasmtime.ValType, fn.ArgCount)
		for i := 0; i < fn.ArgCount; i++ {
			params[i] = i32Type
		}

		returns := make([]*wasmtime.ValType, 0)
		if fn.Returns {
			returns = append(returns, i32Type)
		}

		fnType := wasmtime.NewFuncType(params, returns)

		// this is reused across the normal and Swift variations of the function
		wasmtimeFunc := func(_ *wasmtime.Caller, args []wasmtime.Val) ([]wasmtime.Val, *wasmtime.Trap) {
			hostArgs := make([]any, fn.ArgCount)

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
			returnVals := make([]wasmtime.Val, 0)
			if result != nil {
				returnVals = append(returnVals, wasmtime.ValI32(result.(int32)))
			}

			return returnVals, nil
		}

		// this can return an error but there's nothing we can do about it
		err := linker.FuncNew("env", fn.Name, fnType, wasmtimeFunc)
		if err != nil {
			return errors.Wrapf(err, "linker.FuncNew: env, %s, %s", fn.Name, fnType)
		}
	}

	return nil
}
