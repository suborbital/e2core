//go:build !wasmer && !wasmedge
// +build !wasmer,!wasmedge

package engine

import (
	"github.com/suborbital/e2core/sat/api"
	"github.com/suborbital/e2core/sat/engine/runtime"
	runtimewasmtime "github.com/suborbital/e2core/sat/engine/runtime/wasmtime"
	"github.com/suborbital/systemspec/tenant"
)

func runtimeBuilder(ref *tenant.WasmModuleRef, api api.HostAPI) runtime.RuntimeBuilder {
	return runtimewasmtime.NewBuilder(ref, api)
}
