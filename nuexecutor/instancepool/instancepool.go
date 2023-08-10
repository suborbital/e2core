package instancepool

import (
	"context"
	"sync"
	"time"

	"github.com/bytecodealliance/wasmtime-go/v7"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/suborbital/e2core/sat/engine2/api"
	"github.com/suborbital/e2core/sat/engine2/runtime/instance"
	"github.com/suborbital/e2core/sat/sat/metrics"
)

const (
	queueSize = 512
)

var i32Type = wasmtime.NewValType(wasmtime.KindI32)

type Pool struct {
	instances chan *instance.Instance
	logger    zerolog.Logger

	module *wasmtime.Module
	engine *wasmtime.Engine
	linker *wasmtime.Linker

	lock *sync.RWMutex
}

func New(moduleData []byte, hostAPI api.HostAPI, l zerolog.Logger) (Pool, error) {
	engine := wasmtime.NewEngine()

	// Compiles the module
	mod, err := wasmtime.NewModule(engine, moduleData)
	if err != nil {
		return Pool{}, errors.Wrap(err, "wasmtime.NewModule(engine, moduleData)")
	}

	// Create a linker with WASI functions defined within it
	linker := wasmtime.NewLinker(engine)
	if err := linker.DefineWasi(); err != nil {
		return Pool{}, errors.Wrap(err, "linker.DefineWasi")
	}

	// mount the module API
	err = addHostFns(linker, hostAPI.HostFunctions())
	if err != nil {
		return Pool{}, errors.Wrap(err, "addHostFns")
	}

	p := Pool{
		instances: make(chan *instance.Instance, queueSize),
		lock:      new(sync.RWMutex),
		module:    mod,
		engine:    engine,
		linker:    linker,
		logger:    l.With().Str("module", "instancepool").Logger(),
	}

	// Create an initial pool of instances ready to use.
	for i := 0; i < queueSize/5; i++ {
		go func(blo int) {
			inst, err := p.newInstance()
			if err != nil {
				// @todo nothing for now, but send back a messae to a shutdown channel if 5 errors happen to clean up
				// this e2core baby
				return
			}

			p.logger.Info().Int("initial-instance", blo).Msg("created instance into the pool")
			p.instances <- inst
		}(i)
	}

	return p, nil
}

// newInstance generates a new instance based on the module, engine, linker that is ready to be sent an input to be
// processed.
func (p *Pool) newInstance() (*instance.Instance, error) {
	tmr := metrics.NewTimer()

	store := wasmtime.NewStore(p.engine)

	wasiConfig := wasmtime.NewWasiConfig()
	store.SetWasi(wasiConfig)

	wasmTimeInst, err := p.linker.Instantiate(store, p.module)
	if err != nil {
		return nil, errors.Wrap(err, "failed to linker.Instantiate")
	}

	inst := instance.New(wasmTimeInst, store)

	if _, err := inst.Call("_start"); err != nil {
		if errors.Is(err, instance.ErrExportNotFound) {
			// that's ok, not all modules will have _start
		} else {
			metrics.Meter.InstantiateTime.Record(context.Background(), tmr.ObserveMicroS(), metric.WithAttributes(
				attribute.Bool("false", true),
				attribute.String("error", err.Error()),
			))

			return nil, errors.Wrap(err, "failed to call exported _start")
		}
	}

	metrics.Meter.InstantiateTime.Record(context.Background(), tmr.ObserveMicroS(), metric.WithAttributes(
		attribute.Bool("success", true),
	))

	return inst, nil
}

// GetInstance will pluck an instance from the available pool of instances from the channel, and immediately kicks off
// a goroutine to replenish the pool back to what it was so there's always a number of instances ready to go.
func (p *Pool) GetInstance() *instance.Instance {
	go func() {
		p.logger.Info().Msg("in GetInstance, creating a new instance to replenish the one being given")
		ni, err := p.newInstance()
		if err != nil {
			// do nothing for now
			return
		}

		p.instances <- ni
	}()

	p.logger.Info().Msg("fetching an instance from the pool to give back")

	i := <-p.instances

	return i
}

func (p *Pool) Shutdown() {
	ctx, cxl := context.WithTimeout(context.Background(), 5*time.Second)
	defer cxl()

	select {
	case i := <-p.instances:
		i.Close()

		if len(p.instances) == 0 {
			close(p.instances)
			return
		}

	case <-ctx.Done():
		return
	}
}

// addHostFns adds a list of host functions to an import object. This only happens once per pool.
func addHostFns(linker *wasmtime.Linker, fns []api.HostFn) error {
	for i := range fns {
		// we create a copy inside the loop otherwise things get overwritten
		fn := fns[i]

		// all function params are currently expressed as i32s, which will be improved upon
		// in the future with the introduction of witx-bindgen and/or interface types
		params := make([]*wasmtime.ValType, fn.ArgCount)
		for i := 0; i < fn.ArgCount; i++ {
			params[i] = i32Type
		}

		returns := make([]*wasmtime.ValType, 0)
		if fn.Returns {
			returns = append(returns, i32Type)
		}

		fnType := wasmtime.NewFuncType(params, returns)

		// this is reused across the normal and Swift variations of the function
		wasmtimeFunc := func(_ *wasmtime.Caller, args []wasmtime.Val) ([]wasmtime.Val, *wasmtime.Trap) {
			hostArgs := make([]any, fn.ArgCount)

			// args can be longer than hostArgs (swift, lame), so use hostArgs to control the loop
			for i := range hostArgs {
				ha := args[i].I32()
				hostArgs[i] = ha
			}

			result, err := fn.HostFn(hostArgs...)
			if err != nil {
				return nil, wasmtime.NewTrap(errors.Wrapf(err, "failed to HostFn for %s", fn.Name).Error())
			}

			// function may return nothing, so nil check before trying to convert it
			returnVals := make([]wasmtime.Val, 0)
			if result != nil {
				returnVals = append(returnVals, wasmtime.ValI32(result.(int32)))
			}

			return returnVals, nil
		}

		// this can return an error but there's nothing we can do about it
		err := linker.FuncNew("env", fn.Name, fnType, wasmtimeFunc)
		if err != nil {
			return errors.Wrapf(err, "linker.FuncNew: env, %s, %s", fn.Name, fnType)
		}
	}

	return nil
}
