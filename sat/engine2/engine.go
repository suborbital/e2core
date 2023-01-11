package engine2

import (
	"github.com/suborbital/e2core/foundation/scheduler"
	"github.com/suborbital/e2core/sat/engine2/api"
	"github.com/suborbital/systemspec/tenant"
)

// Engine is a Webassembly job scheduler with configurable host APIs
type Engine struct {
	*scheduler.Scheduler
}

// New creates a new Engine with the default API
func New(name string, ref *tenant.WasmModuleRef, api api.HostAPI) *Engine {
	e := &Engine{
		Scheduler: scheduler.New(),
	}

	runner := newRunnerFromRef(ref, api)

	e.Scheduler.Register(
		name,
		runner,
		scheduler.Autoscale(24),
		scheduler.MaxRetries(0),
		scheduler.RetrySeconds(0),
		scheduler.PreWarm(),
	)

	return e
}
