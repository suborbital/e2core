package rwasm

import (
	"fmt"

	"github.com/wasmerio/wasmer-go/wasmer"
)

// HostFn describes a host function callable from within a Runnable module
type HostFn struct {
	name   string
	args   []wasmer.ValueKind
	ret    []wasmer.ValueKind
	hostFn func(...wasmer.Value) (interface{}, error)
}

// newHostFn creates a new host funcion
func newHostFn(name string, argLen int, returns bool, fn func(...wasmer.Value) (interface{}, error)) *HostFn {
	retVals := []wasmer.ValueKind{}
	if returns {
		retVals = append(retVals, wasmer.I32)
	}

	args := make([]wasmer.ValueKind, argLen)
	for i := 0; i < argLen; i++ {
		args[i] = wasmer.I32
	}

	hfn := &HostFn{
		name:   name,
		args:   args,
		ret:    retVals,
		hostFn: fn,
	}

	return hfn
}

// addHostFns adds a list of host functions to an import object
func addHostFns(imports *wasmer.ImportObject, store *wasmer.Store, fns ...*HostFn) {
	externMap := map[string]wasmer.IntoExtern{}

	for _, fn := range fns {
		externMap[fn.fnName()] = fn.toWasmerFn(store)
		externMap[fn.fnSwiftName()] = fn.toWasmerSwiftFn(store)
	}

	imports.Register("env", externMap)
}

func (h *HostFn) toWasmerFn(store *wasmer.Store) *wasmer.Function {
	wasmerFn := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(h.fnArgs(), h.fnReturns()),
		h.fn(),
	)

	return wasmerFn
}

func (h *HostFn) toWasmerSwiftFn(store *wasmer.Store) *wasmer.Function {
	wasmerFn := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(h.fnSwiftArgs(), h.fnReturns()),
		h.fn(),
	)

	return wasmerFn
}

// Name returns the fn name
func (h *HostFn) fnName() string {
	return h.name
}

// SwiftName returns the Swift variant of the function name
func (h *HostFn) fnSwiftName() string {
	return fmt.Sprintf("%s_swift", h.name)
}

// Args returns the argument types for the function
func (h *HostFn) fnArgs() []*wasmer.ValueType {
	return wasmer.NewValueTypes(h.args...)
}

// SwiftArgs returns the argument types for the function's Swift variant
func (h *HostFn) fnSwiftArgs() []*wasmer.ValueType {
	swiftArgs := append(h.args, wasmer.I32, wasmer.I32)

	return wasmer.NewValueTypes(swiftArgs...)
}

// Returns returns the return value types for the function
func (h *HostFn) fnReturns() []*wasmer.ValueType {
	return wasmer.NewValueTypes(h.ret...)
}

// Fn translates wraps the host fn in a Wasmer fn
func (h *HostFn) fn() func([]wasmer.Value) ([]wasmer.Value, error) {
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
