package worker

import (
	"context"
	"sync"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/suborbital/e2core/foundation/tracing"
	"github.com/suborbital/e2core/nuexecutor/instancepool"
	"github.com/suborbital/e2core/sat/engine2/runtime/instance"
	"github.com/suborbital/systemspec/fqmn"
)

const (
	workersDefault uint8  = 255
	bufferDefault  uint16 = 10000
)

var (
	ErrContextDeadlineExceeded = errors.New("shutdown did not finish, context deadline exceeded, bailing earlier")
)

type Wasm struct {
	// source is to get the compiled wasm module in byte slice from somewhere.
	// source ModSource
	provider instancepool.Pool

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

func New(c Config, l zerolog.Logger, pool instancepool.Pool) (*Wasm, error) {
	workers := workersDefault
	buffer := bufferDefault

	if c.Workers > uint8(0) {
		workers = c.Workers
	}

	if c.Buffer > uint16(0) {
		buffer = c.Buffer
	}

	return &Wasm{
		provider: pool,
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
		go w.work()
		w.wg.Add(1)
	}

	// this is to keep track of the pool when shutting it down.
	w.wg.Add(1)

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

	// Shut down the instance provider. The Shutdown method blocks until either all instances from the channel are
	// drained and closed, or 5 seconds has passed. Then decrement the waitgroup by one. In the Start method of the
	// worker we added an extra increment to the waitgroup to account for the pool.
	go func() {
		w.provider.Shutdown()

		w.wg.Done()
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
func (w *Wasm) work() {
	for {
		select {
		case incomingJob := <-w.incoming:
			_, span := tracing.Tracer.Start(incomingJob.ctx, "work")

			jb := incomingJob.Input()

			span.AddEvent("provider.GetInstance")
			readyInstance := w.provider.GetInstance()

			inPointer, writeErr := readyInstance.WriteMemory(jb)
			if writeErr != nil {
				incomingJob.errChan <- errors.Wrap(writeErr, "w.inst.WriteMemory")

				span.AddEvent("w.inst.WriteMemory failed", trace.WithAttributes(
					attribute.String("error", writeErr.Error()),
				))
				span.End()

				continue
			}

			ident, err := instance.Store(readyInstance)
			if err != nil {
				incomingJob.errChan <- errors.Wrap(err, "instance.Store")

				span.AddEvent("instance.Store failed", trace.WithAttributes(
					attribute.String("error", err.Error()),
				))
				span.End()

				continue
			}

			// execute the module's Run function, passing the input data and ident
			// set runErr but don't return because the ExecutionResult error should also be grabbed
			_, callErr := readyInstance.Call("run_e", inPointer, int32(len(jb)), ident)
			if callErr != nil {
				incomingJob.errChan <- errors.Wrap(callErr, "w.inst.Call")

				span.AddEvent("w.inst.Call run_e failed", trace.WithAttributes(
					attribute.String("error", callErr.Error()),
				))
				span.End()

				continue
			}

			// get the results from the instance
			output, runErr := readyInstance.ExecutionResult()
			if runErr != nil {
				incomingJob.errChan <- errors.Wrap(runErr, "w.inst.ExecutionResult")

				span.AddEvent("w.inst.ExecutionResult failed", trace.WithAttributes(
					attribute.String("error", runErr.Error()),
				))
				span.End()

				continue
			}

			incomingJob.responseChan <- Result{content: output}
			span.AddEvent("result returned successfully")
			span.End()
		case <-w.shutdown:
			w.wg.Done()
			return
		}
	}
}
