package worker

import (
	"context"
	"sync"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"

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
	source ModSource

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

func New(c Config, l zerolog.Logger, source ModSource) *Wasm {
	workers := workersDefault
	buffer := bufferDefault

	if c.Workers > uint8(0) {
		workers = c.Workers
	}

	if c.Buffer > uint16(0) {
		buffer = c.Buffer
	}

	return &Wasm{
		source:   source,
		workers:  workers,
		incoming: make(chan Job, buffer),
		wg:       new(sync.WaitGroup),
		logger:   l.With().Str("component", "nuexecutor").Logger(),
	}
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
			ll.Info().Bytes("bla", j.Input()).Msg("received message")
		case <-w.shutdown:
			ll.Info().Msg("signal received on shutdown channel, returning")
			w.wg.Done()
			return
		}
	}
}
