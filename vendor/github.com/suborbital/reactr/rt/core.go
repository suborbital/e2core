package rt

import (
	"fmt"
	"sync"

	"github.com/pkg/errors"
	"github.com/suborbital/vektor/vlog"
)

// coreDoFunc is an internal version of DoFunc that takes a
// Job pointer instead of a Job value for the best memory usage
type coreDoFunc func(job *Job) *Result

// core is the 'core scheduler' for reactr, handling execution of
// Tasks, Jobs, and Schedules
type core struct {
	workers map[string]*worker
	watcher *watcher
	cache   Cache
	log     *vlog.Logger
	lock    sync.RWMutex
}

func newCore(log *vlog.Logger, cache Cache) *core {
	c := &core{
		workers: map[string]*worker{},
		cache:   cache,
		log:     log,
		lock:    sync.RWMutex{},
	}

	c.watcher = newWatcher(c.do)

	return c
}

func (c *core) do(job *Job) *Result {
	result := newResult(job.UUID())

	worker := c.findWorker(job.jobType)
	if worker == nil {
		result.sendErr(fmt.Errorf("failed to getWorker for jobType %q", job.jobType))
		return result
	}

	go func() {
		if !worker.isStarted() {
			// "recursively" pass this function as the runFunc for the runnable
			if err := worker.start(c.do); err != nil {
				result.sendErr(errors.Wrapf(err, "failed start worker for jobType %q", job.jobType))
				return
			}
		}

		job.result = result

		worker.schedule(job)
	}()

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

	w := newWorker(runnable, c.cache, opts)

	c.workers[jobType] = w

	if opts.preWarm {
		go func() {
			if err := w.start(c.do); err != nil {
				c.log.Error(errors.Wrapf(err, "failed to preWarm %s worker", jobType))
			}
		}()
	}
}

func (c *core) watch(sched Schedule) {
	c.watcher.watch(sched)
}

func (c *core) findWorker(jobType string) *worker {
	c.lock.RLock()
	defer c.lock.RUnlock()

	if c.workers == nil {
		return nil
	}

	if w, ok := c.workers[jobType]; ok {
		return w
	}

	return nil
}

func (c *core) hasWorker(jobType string) bool {
	w := c.findWorker(jobType)

	return w != nil
}
