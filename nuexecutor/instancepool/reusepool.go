package instancepool

import (
	"context"
	"sync"
	"time"

	"github.com/bytecodealliance/wasmtime-go/v7"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/suborbital/e2core/foundation/tracing"
	"github.com/suborbital/e2core/sat/engine2/api"
	"github.com/suborbital/e2core/sat/engine2/runtime/instance"
)

type Reuse struct {
	instances chan *instance.Instance

	lock   *sync.RWMutex
	logger zerolog.Logger
	module *wasmtime.Module
	engine *wasmtime.Engine
	linker *wasmtime.Linker
}

func NewReuse(moduleData []byte, hostAPI api.HostAPI, l zerolog.Logger) (Reuse, error) {
	engine := wasmtime.NewEngine()

	// Compiles the module
	mod, err := wasmtime.NewModule(engine, moduleData)
	if err != nil {
		return Reuse{}, errors.Wrap(err, "wasmtime.NewModule(engine, moduleData)")
	}

	// Create a linker with WASI functions defined within it
	linker := wasmtime.NewLinker(engine)
	if err := linker.DefineWasi(); err != nil {
		return Reuse{}, errors.Wrap(err, "linker.DefineWasi")
	}

	// mount the module API
	err = addHostFns(linker, hostAPI.HostFunctions())
	if err != nil {
		return Reuse{}, errors.Wrap(err, "addHostFns")
	}

	p := Reuse{
		instances: make(chan *instance.Instance, queueSize),
		lock:      new(sync.RWMutex),
		module:    mod,
		engine:    engine,
		linker:    linker,
		logger:    l.With().Str("module", "instancepool").Logger(),
	}

	wg := new(sync.WaitGroup)

	// Create an initial pool of instances ready to use.
	for i := 0; i < queueSize; i++ {
		wg.Add(1)
		go func(blo int) {
			inst, err := p.newInstance()
			if err != nil {
				// @todo nothing for now, but send back a messae to a shutdown channel if 5 errors happen to clean up
				// this e2core baby
				return
			}

			p.logger.Info().Int("initial-instance", blo).Msg("created instance into the pool")
			p.instances <- inst
			wg.Done()
		}(i)
	}

	wg.Wait()

	return p, nil
}

// newInstance generates a new instance based on the module, engine, linker that is ready to be sent an input to be
// processed.
func (p *Reuse) newInstance() (*instance.Instance, error) {
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
			return nil, errors.Wrap(err, "failed to call exported _start")
		}
	}

	return inst, nil
}

// GetInstance will pluck an instance from the available pool of instances from the channel, and immediately kicks off
// a goroutine to replenish the pool back to what it was so there's always a number of instances ready to go.
func (p *Reuse) GetInstance(ctx context.Context) *instance.Instance {
	_, span := tracing.Tracer.Start(ctx, "reuse getinstance")
	defer span.End()

	span.AddEvent("grabbing an instance from the queue", trace.WithAttributes(
		attribute.Int("instances len", len(p.instances)),
	))
	i := <-p.instances

	span.AddEvent("got the instance, did not block")

	return i
}

func (p *Reuse) GiveInstanceBack(ctx context.Context, i *instance.Instance) {
	_, span := tracing.Tracer.Start(ctx, "reuse give instance back")
	defer span.End()

	span.AddEvent("sending the received instance back into the queue", trace.WithAttributes(
		attribute.Int("instances len", len(p.instances)),
	))
	p.instances <- i

	span.AddEvent("instance put back onto the channel")
}

func (p *Reuse) Shutdown() {
	stopwg := new(sync.WaitGroup)
	ctx, cxl := context.WithTimeout(context.Background(), 15*time.Second)
	defer cxl()

	doneChan := make(chan struct{})

	stopwg.Add(1)

	go func() {
		for {
			i := <-p.instances

			i.Close()

			if len(p.instances) == 0 {
				break
			}
		}

		stopwg.Done()
	}()

	go func() {
		stopwg.Wait()
		doneChan <- struct{}{}
	}()

	select {
	case <-doneChan:
		p.logger.Info().Msg("shutdown complete correctly")
	case <-ctx.Done():
		p.logger.Info().Msg("reuse shutdown timeout reached, terminating")
	}

	close(p.instances)
}
