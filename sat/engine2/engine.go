package engine2

import (
	"github.com/suborbital/e2core/foundation/scheduler"
	"github.com/suborbital/e2core/sat/engine2/api"
	"github.com/suborbital/e2core/sat/engine2/runtime"
	"github.com/suborbital/systemspec/tenant"
)

// Engine is a Webassembly job scheduler with configurable host APIs
type Engine struct {
	*scheduler.Scheduler
	pool runtime.InstancePool
}

// New creates a new Engine with the default API
func New(name string, ref *tenant.WasmModuleRef, api api.HostAPI, opts ...scheduler.Option) *Engine {
	e := &Engine{
		Scheduler: scheduler.New(),
		pool:      *runtime.NewInstancePool(ref, api),
	}

	runner := newRunnerFromRef(ref, api)

	e.Scheduler.Register(name, runner, opts...)

	return e
}
