//go:build wasmedge
// +build wasmedge

// we compile-exclude wasmedge by default as tests will fail unless WasmEdge is installed

package runtimewasmedge

import (
	"fmt"

	"github.com/second-state/WasmEdge-go/wasmedge"

	"github.com/suborbital/e2core/sat/engine/runtime"
)

// toWasmEdgeHostFn creates a new host funcion from a generic host fn
func toWasmEdgeHostFn(hostFn runtime.HostFn) func(data interface{}, mem *wasmedge.Memory, params []interface{}) ([]interface{}, wasmedge.Result) {
	return func(data interface{}, mem *wasmedge.Memory, params []interface{}) ([]interface{}, wasmedge.Result) {
		hostResult, hostErr := hostFn.HostFn(params...)
		if hostErr != nil {
			return nil, wasmedge.Result_Fail
		}

		return []interface{}{hostResult}, wasmedge.Result_Success
	}
}

// addHostFns adds a list of host functions to an import object
func addHostFns(imports *wasmedge.ImportObject, fns ...runtime.HostFn) {
	for _, fn := range fns {
		wasmHostFn := toWasmEdgeHostFn(fn)

		argsType := make([]wasmedge.ValType, fn.ArgCount)
		for i := 0; i < fn.ArgCount; i++ {
			argsType[i] = wasmedge.ValType_I32
		}

		retType := []wasmedge.ValType{}
		if fn.Returns {
			retType = append(retType, wasmedge.ValType_I32)
		}
		funcType := wasmedge.NewFunctionType(argsType, retType)

		wasmEdgeHostFn := wasmedge.NewFunction(funcType, wasmHostFn, nil, 0)
		imports.AddFunction(fn.Name, wasmEdgeHostFn)

		swiftArgsType := append(argsType, wasmedge.ValType_I32, wasmedge.ValType_I32)
		swiftFuncType := wasmedge.NewFunctionType(swiftArgsType, retType)
		swiftWasmEdgeHostFn := wasmedge.NewFunction(swiftFuncType, wasmHostFn, nil, 0)
		swiftFuncName := fmt.Sprintf("%s_swift", fn.Name)
		imports.AddFunction(swiftFuncName, swiftWasmEdgeHostFn)
	}
}
