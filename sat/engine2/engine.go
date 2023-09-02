package engine2

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"

	"github.com/suborbital/e2core/foundation/scheduler"
	"github.com/suborbital/e2core/sat/engine2/api"
	"github.com/suborbital/systemspec/tenant"
)

// Engine is a Webassembly job scheduler with configurable host APIs
type Engine struct {
	*scheduler.Scheduler
}

// New creates a new Engine with the default API
func New(name string, ref *tenant.WasmModuleRef, api api.HostAPI, logger zerolog.Logger) *Engine {
	e := &Engine{
		Scheduler: scheduler.NewWithLogger(logger.With().Str("component", "engine.scheduler").Logger()),
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

func WasmRefFromFile(filename string) (*tenant.WasmModuleRef, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, errors.Wrap(err, "failed to Open")
	}

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, errors.Wrap(err, "failed to ReadAll")
	}

	name := strings.TrimSuffix(filepath.Base(filename), ".wasm")
	ref := &tenant.WasmModuleRef{Name: name, Data: data}

	return ref, nil
}
