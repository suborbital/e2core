package engine

import (
	"github.com/pkg/errors"

	"github.com/suborbital/e2core/foundation/scheduler"
	"github.com/suborbital/e2core/sat/engine/runtime/api"
	"github.com/suborbital/systemspec/tenant"
)

// Engine is a Webassembly job scheduler with configurable host APIs
type Engine struct {
	*scheduler.Scheduler
	api api.HostAPI
}

// New creates a new Engine with the default API
func New() *Engine {
	return NewWithAPI(api.New())
}

// NewWithAPI creates a new Engine with the given API
func NewWithAPI(api api.HostAPI) *Engine {
	e := &Engine{
		Scheduler: scheduler.New(),
		api:       api,
	}

	return e
}

// Register registers a Wasm module by reference
func (e *Engine) Register(name string, ref *tenant.WasmModuleRef, opts ...scheduler.Option) scheduler.JobFunc {
	runner := newRunnerFromRef(ref, e.api)

	return e.Scheduler.Register(name, runner, opts...)
}

// RegisterFromFile registers a Wasm module by reference
func (e *Engine) RegisterFromFile(name, filename string, opts ...scheduler.Option) (scheduler.JobFunc, error) {
	runner, err := newRunnerFromFile(filename, e.api)
	if err != nil {
		return nil, errors.Wrap(err, "failed to newRunnerFromFile")
	}

	jobFunc := e.Scheduler.Register(name, runner, opts...)

	return jobFunc, nil
}
