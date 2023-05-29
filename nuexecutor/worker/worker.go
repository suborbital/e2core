package worker

import (
	"context"
	"sync"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"

	"github.com/suborbital/e2core/sat/engine2/api"
	"github.com/suborbital/e2core/sat/engine2/runtime/instance"
	"github.com/suborbital/systemspec/fqmn"
)

const (
	workersDefault uint8  = 16
	bufferDefault  uint16 = 10000
)

var (
	ErrContextDeadlineExceeded = errors.New("shutdown did not finish, context deadline exceeded, bailing earlier")
)

type Wasm struct {
	// source is to get the compiled wasm module in byte slice from somewhere.
	// source ModSource
	inst *instance.Instance

	// workers holds info on how many go routines to launch that handle incoming jobs.
	workers uint8

	// incoming is the channel that the module returns so handlers can use it to send jobs to process.
	incoming chan Job

	// shutdown channel receives a struct signal to terminate individual workers.
	shutdown chan struct{}

	// wg is used to coordinate graceful shutdown of all workers.
	wg *sync.WaitGroup

	// logger is a scoped logger for the executor.
	logger zerolog.Logger
}

type Config struct {
	Workers uint8
	Buffer  uint16
}

type ModSource interface {
	Get(context.Context, fqmn.FQMN) ([]byte, error)
}

func New(c Config, l zerolog.Logger, wasmBytes []byte) (*Wasm, error) {
	workers := workersDefault
	buffer := bufferDefault

	if c.Workers > uint8(0) {
		workers = c.Workers
	}

	if c.Buffer > uint16(0) {
		buffer = c.Buffer
	}

	inst, err := buildModule(wasmBytes, api.New(l).HostFunctions())
	if err != nil {
		return nil, errors.Wrap(err, "buildModule")
	}

	return &Wasm{
		inst: inst,
		// source:   source,
		workers:  workers,
		incoming: make(chan Job, buffer),
		wg:       new(sync.WaitGroup),
		logger:   l.With().Str("component", "nuexecutor").Logger(),
	}, nil
}

// Start launches workers in a goroutine. Number of workers is governed by the Config.Workers configuration property. It
// returns a unidirectional channel that consuming code can send individual jobs to which will be picked up by the work
// method.
func (w *Wasm) Start() chan<- Job {
	for i := uint8(0); i < w.workers; i++ {
		go w.work(i)
		w.wg.Add(1)
	}

	return w.incoming
}

// Shutdown receives a context, ideally with a timeout or a deadline, and starts tearing down the workers gracefully.
func (w *Wasm) Shutdown(ctx context.Context) error {
	// Create a new local channel to signal when all goroutines have stopped.
	wgDone := make(chan struct{})

	// Close the shutdown channel, which all goroutines are waiting on, which will trigger all of them to terminate and
	// call wg.Done.
	close(w.shutdown)

	// Wait until all workers actually stopped. Then we send a signal to the wgDone channel, which the below select
	// block picks up, ideally before the ctx.Done() sends a message.
	go func() {
		w.wg.Wait()

		wgDone <- struct{}{}
	}()

	// Once we've set up the structured teardown, wait on one of the two events:
	// - either context deadline exceeds, or context gets manually cancelled, which means shutdown is improper, and we
	//   return an error, or
	// - we receive a signal on the wgDone channel, which means everything stopped correctly, and we return nil.
	select {
	case <-ctx.Done():
		return ErrContextDeadlineExceeded
	case <-wgDone:
		return nil
	}
}

// work is the workhorse of the router. This picks up the jobs sent to the incoming channel.
func (w *Wasm) work(n uint8) {
	ll := w.logger.With().Int("worker", int(n)).Logger()
	for {
		select {
		case j := <-w.incoming:
			jb := j.Input()

			inPointer, writeErr := w.inst.WriteMemory(jb)
			if writeErr != nil {
				j.errChan <- errors.Wrap(writeErr, "w.inst.WriteMemory")
				return
			}

			ident, err := instance.Store(w.inst)
			if err != nil {
				j.errChan <- errors.Wrap(err, "instance.Store")
			}

			// execute the module's Run function, passing the input data and ident
			// set runErr but don't return because the ExecutionResult error should also be grabbed
			_, callErr := w.inst.Call("run_e", inPointer, int32(len(jb)), ident)
			if callErr != nil {
				j.errChan <- errors.Wrap(callErr, "w.inst.Call")
				continue
			}

			// get the results from the instance
			output, runErr := w.inst.ExecutionResult()
			if runErr != nil {
				j.errChan <- errors.Wrap(runErr, "w.inst.ExecutionResult")
				continue
			}

			ll.Info().Bytes("bla", j.Input()).Msg("received message")
			j.responseChan <- Result{content: output}
			ll.Info().Msg("sent message back to job's response channel")
		case <-w.shutdown:
			ll.Info().Msg("signal received on shutdown channel, returning")
			w.wg.Done()
			return
		}
	}
}
