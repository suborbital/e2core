//go:build wasmer
// +build wasmer

package engine

import (
	"github.com/suborbital/appspec/tenant"
	"github.com/suborbital/e2core/sat/api"
	"github.com/suborbital/e2core/sat/engine/runtime"
	runtimewasmer "github.com/suborbital/e2core/sat/engine/runtime/wasmer"
)

func runtimeBuilder(ref *tenant.WasmModuleRef, api api.HostAPI) runtime.RuntimeBuilder {
	return runtimewasmer.NewBuilder(ref, api)
}
