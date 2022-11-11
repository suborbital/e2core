//go:build wasmedge
// +build wasmedge

package engine

import (
	"github.com/suborbital/appspec/tenant"
	"github.com/suborbital/e2core/sat/api"
	"github.com/suborbital/e2core/sat/engine/runtime"
	runtimewasmedge "github.com/suborbital/e2core/sat/engine/runtime/wasmedge"
)

func runtimeBuilder(ref *tenant.WasmModuleRef, api api.HostAPI) runtime.RuntimeBuilder {
	return runtimewasmedge.NewBuilder(ref, api)
}
