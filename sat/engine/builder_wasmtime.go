//go:build !wasmer && !wasmedge
// +build !wasmer,!wasmedge

package engine

import (
	"github.com/suborbital/appspec/tenant"

	"github.com/suborbital/e2core/sat/api"
	"github.com/suborbital/e2core/sat/engine/runtime"
	runtimewasmtime "github.com/suborbital/e2core/sat/engine/runtime/wasmtime"
)

func runtimeBuilder(ref *tenant.WasmModuleRef, api api.HostAPI) runtime.RuntimeBuilder {
	return runtimewasmtime.NewBuilder(ref, api)
}
