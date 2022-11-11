package runtimewasmer

import (
	"fmt"

	"github.com/wasmerio/wasmer-go/wasmer"

	"github.com/suborbital/e2core/sat/engine/runtime"
)

// WasmerHostFn describes a host function callable from within a Runnable module
type WasmerHostFn struct {
	name   string
	args   []wasmer.ValueKind
	ret    []wasmer.ValueKind
	hostFn func(...wasmer.Value) (interface{}, error)
}

// toWasmerHostFn creates a new host funcion from a generic host fn
func toWasmerHostFn(hostFn runtime.HostFn) *WasmerHostFn {
	retVals := []wasmer.ValueKind{}
	if hostFn.Returns {
		retVals = append(retVals, wasmer.I32)
	}

	args := make([]wasmer.ValueKind, hostFn.ArgCount)
	for i := 0; i < hostFn.ArgCount; i++ {
		args[i] = wasmer.I32
	}

	// create a wasmer-specific representation of the generic host function
	hfn := &WasmerHostFn{
		name: hostFn.Name,
		args: args,
		ret:  retVals,
		hostFn: func(wasmerArgs ...wasmer.Value) (interface{}, error) {
			funcArgs := make([]interface{}, len(wasmerArgs))
			for i, a := range wasmerArgs {
				funcArgs[i] = a.I32()
			}

			return hostFn.HostFn(funcArgs...)
		},
	}

	return hfn
}

// addHostFns adds a list of host functions to an import object
func addHostFns(imports *wasmer.ImportObject, store *wasmer.Store, fns ...runtime.HostFn) {
	externMap := map[string]wasmer.IntoExtern{}

	for _, fn := range fns {
		wasmerHostFn := toWasmerHostFn(fn)

		externMap[wasmerHostFn.fnName()] = wasmerHostFn.toWasmerFn(store)
		externMap[wasmerHostFn.fnSwiftName()] = wasmerHostFn.toWasmerSwiftFn(store)
	}

	imports.Register("env", externMap)
}

func (h *WasmerHostFn) toWasmerFn(store *wasmer.Store) *wasmer.Function {
	wasmerFn := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(h.fnArgs(), h.fnReturns()),
		h.innerFn(),
	)

	return wasmerFn
}

func (h *WasmerHostFn) toWasmerSwiftFn(store *wasmer.Store) *wasmer.Function {
	wasmerFn := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(h.fnSwiftArgs(), h.fnReturns()),
		h.innerFn(),
	)

	return wasmerFn
}

// Name returns the fn name
func (h *WasmerHostFn) fnName() string {
	return h.name
}

// SwiftName returns the Swift variant of the function name
func (h *WasmerHostFn) fnSwiftName() string {
	return fmt.Sprintf("%s_swift", h.name)
}

// Args returns the argument types for the function
func (h *WasmerHostFn) fnArgs() []*wasmer.ValueType {
	return wasmer.NewValueTypes(h.args...)
}

// SwiftArgs returns the argument types for the function's Swift variant
func (h *WasmerHostFn) fnSwiftArgs() []*wasmer.ValueType {
	swiftArgs := append(h.args, wasmer.I32, wasmer.I32)

	return wasmer.NewValueTypes(swiftArgs...)
}

// Returns returns the return value types for the function
func (h *WasmerHostFn) fnReturns() []*wasmer.ValueType {
	return wasmer.NewValueTypes(h.ret...)
}

// innerFn translates wraps the host fn in a Wasmer fn
func (h *WasmerHostFn) innerFn() func([]wasmer.Value) ([]wasmer.Value, error) {
	return func(argL []wasmer.Value) ([]wasmer.Value, error) {
		result, err := h.hostFn(argL...)
		if err != nil {
			return nil, err
		}

		retVals := []wasmer.Value{}
		if result != nil {
			retVals = append(retVals, wasmer.NewValue(result, wasmer.I32))
		}

		return retVals, nil
	}
}
