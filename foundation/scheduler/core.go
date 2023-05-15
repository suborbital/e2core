package scheduler

import (
	"fmt"
	"sync"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

// coreDoFunc is an internal version of DoFunc that takes a
// Job pointer instead of a Job value for the best memory usage
type coreDoFunc func(job *Job) *Result

// core is the 'core scheduler' for reactr, handling execution of
// Tasks, Jobs, and Schedules
type core struct {
	// scaler holds references to workers and autoscales their workThreads
	scaler *scaler
	// watcher holds onto active Schedules and ensures they get executed
	watcher *watcher

	log  zerolog.Logger
	lock sync.RWMutex
}

func newCore(log zerolog.Logger) *core {
	c := &core{
		scaler: newScaler(log),
		log:    log,
		lock:   sync.RWMutex{},
	}

	c.watcher = newWatcher(c.do)

	return c
}

func (c *core) do(job *Job) *Result {
	result := newResult(job.UUID())

	rid := "no-request"
	if job.Req() != nil {
		rid = job.Req().ID
	}

	ll := c.log.With().Str("requestID", rid).Logger()

	ll.Info().Msg("core.do function got called")

	jobWorker := c.scaler.findWorker(job.jobType)
	if jobWorker == nil {
		result.sendErr(fmt.Errorf("failed to getWorker for jobType %q", job.jobType))
		return result
	}

	go func() {
		job.result = result
		ll.Info().Msg("jobworker got a job scheduled")

		jobWorker.schedule(job)
	}()

	ll.Info().Msg("returning result from core.do func")
	return result
}

// register adds a handler
func (c *core) register(jobType string, runnable Runnable, options ...Option) {
	c.lock.Lock()
	defer c.lock.Unlock()

	// apply the provided options
	opts := defaultOpts(jobType)
	for _, o := range options {
		opts = o(opts)
	}

	if opts.autoscaleMax > opts.poolSize {
		// only start the autoscaler if one of the Runnables needs it
		c.scaler.startAutoscaler()
	}

	w := newWorker(runnable, c.do, opts)

	c.scaler.addWorker(jobType, w)
}

func (c *core) deRegister(jobType string) error {
	if err := c.scaler.removeWorker(jobType); err != nil {
		return errors.Wrap(err, "failed to removeWorker")
	}

	return nil
}

func (c *core) hasWorker(jobType string) bool {
	w := c.scaler.findWorker(jobType)

	return w != nil
}

func (c *core) watch(sched Schedule) {
	c.watcher.watch(sched)
}

func (c *core) metrics() ScalerMetrics {
	return c.scaler.metrics()
}
