//go:build wasmedge
// +build wasmedge

// we compile-exclude wasmedge by default as tests will fail unless WasmEdge is installed

package runtimewasmedge

import (
	"github.com/pkg/errors"
	"github.com/second-state/WasmEdge-go/wasmedge"

	"github.com/suborbital/e2core/sat/api"
	"github.com/suborbital/e2core/sat/engine/runtime"
	"github.com/suborbital/systemspec/tenant"
)

// WasmEdgeBuilder is a WasmEdge implementation of the instanceBuilder interface
type WasmEdgeBuilder struct {
	ref     *tenant.WasmModuleRef
	hostFns []runtime.HostFn
}

// NewBuilder create a new WasmEdgeBuilder
func NewBuilder(ref *tenant.WasmModuleRef, hostAPI api.HostAPI) runtime.RuntimeBuilder {
	w := &WasmEdgeBuilder{
		ref:     ref,
		hostFns: hostAPI.HostFunctions(),
	}
	return w
}

func (w *WasmEdgeBuilder) New() (runtime.RuntimeInstance, error) {
	imports, ast, err := w.setupAST()
	if err != nil {
		return nil, err
	}

	// Create store
	store := wasmedge.NewStore()

	// Create executor
	executor := wasmedge.NewExecutor()

	// Register import object
	executor.RegisterImport(store, imports)

	wasiImports := wasmedge.NewWasiImportObject(nil, nil, nil)
	executor.RegisterImport(store, wasiImports)

	// Instantiate store
	executor.Instantiate(store, ast)
	ast.Release()

	wasiStart := store.FindFunction("_start")
	if wasiStart != nil {
		if _, err := executor.Invoke(store, "_start"); err != nil {
			return nil, errors.Wrap(err, "failed to _start")
		}
	}
	init := store.FindFunction("init")
	if init != nil {
		if _, err := executor.Invoke(store, "init"); err != nil {
			return nil, errors.Wrap(err, "failed to init")
		}
	}

	inst := &WasmEdgeRuntime{
		imports:  imports,
		store:    store,
		executor: executor,
	}

	return inst, nil
}

func (w *WasmEdgeBuilder) setupAST() (*wasmedge.ImportObject, *wasmedge.AST, error) {
	// Set not to print debug info
	wasmedge.SetLogErrorLevel()

	moduleBytes, err := w.ref.Bytes()
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to get ref ModuleBytes")
	}

	// Create Loader
	loader := wasmedge.NewLoader()

	// Create AST
	ast, err := loader.LoadBuffer(moduleBytes)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to create ast")
	}
	loader.Release()

	// Validate the ast
	val := wasmedge.NewValidator()
	err = val.Validate(ast)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to validate ast")
	}
	val.Release()

	// Create import object
	imports := wasmedge.NewImportObject("env")

	// mount the Runnable API host functions to the module's imports
	addHostFns(imports, w.hostFns...)

	return imports, ast, nil
}
