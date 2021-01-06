package hive

import (
	"fmt"
	"sync"

	"github.com/pkg/errors"
	"github.com/suborbital/vektor/vlog"
)

type scheduler struct {
	workers map[string]*worker
	store   Storage
	cache   Cache
	logger  *vlog.Logger
	sync.Mutex
}

func newScheduler(logger *vlog.Logger, cache Cache) *scheduler {
	s := &scheduler{
		workers: map[string]*worker{},
		store:   newMemoryStorage(),
		cache:   cache,
		logger:  logger,
		Mutex:   sync.Mutex{},
	}

	return s
}

func (s *scheduler) schedule(job Job) *Result {
	result := newResult(job.UUID(), func(uuid string) {
		if err := s.store.Remove(uuid); err != nil {
			s.logger.Error(errors.Wrap(err, "scheduler failed to Remove Job from storage"))
		}
	})

	worker := s.getWorker(job.jobType)
	if worker == nil {
		result.sendErr(fmt.Errorf("failed to getRunnable for jobType %q", job.jobType))
		return result
	}

	go func() {
		if !worker.isStarted() {
			// "recursively" pass this function as the runFunc for the runnable
			if err := worker.start(s.schedule); err != nil {
				result.sendErr(errors.Wrapf(err, "failed start worker for jobType %q", job.jobType))
				return
			}
		}

		job.result = result
		s.store.Add(job)

		worker.schedule(job.Reference())
	}()

	return result
}

// handle adds a handler
func (s *scheduler) handle(jobType string, runnable Runnable, options ...Option) {
	s.Lock()
	defer s.Unlock()

	// apply the provided options
	opts := defaultOpts(jobType)
	for _, o := range options {
		opts = o(opts)
	}

	w := newWorker(runnable, s.store, s.cache, opts)
	if s.workers == nil {
		s.workers = map[string]*worker{jobType: w}
	} else {
		s.workers[jobType] = w
	}
}

func (s *scheduler) getWorker(jobType string) *worker {
	s.Lock()
	defer s.Unlock()

	if s.workers == nil {
		return nil
	}

	if w, ok := s.workers[jobType]; ok {
		return w
	}

	return nil
}
