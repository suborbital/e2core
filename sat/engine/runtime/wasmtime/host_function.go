package runtimewasmtime

import (
	"fmt"

	"github.com/bytecodealliance/wasmtime-go"
	"github.com/pkg/errors"

	"github.com/suborbital/e2core/sat/engine/runtime"
)

var i32Type = wasmtime.NewValType(wasmtime.KindI32)

// addHostFns adds a list of host functions to an import object
func addHostFns(linker *wasmtime.Linker, fns ...runtime.HostFn) {
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

		// add swift params and mount swift variation
		params = append(params, i32Type, i32Type)
		swiftFnType := wasmtime.NewFuncType(params, returns)

		_ = linker.FuncNew("env", fmt.Sprintf("%s_swift", fn.Name), swiftFnType, wasmtimeFunc)
	}
}
