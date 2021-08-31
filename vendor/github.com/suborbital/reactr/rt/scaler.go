package rt

import (
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/suborbital/vektor/vlog"
)

// ScalerMetrics is internal metrics about the scaler
type ScalerMetrics struct {
	TotalThreadCount int                      `json:"totalThreadCount"`
	TotalJobCount    int                      `json:"totalJobCount"`
	Workers          map[string]WorkerMetrics `json:"workers"`
}

// WorkerMetrics is metrics about a worker
type WorkerMetrics struct {
	TargetThreadCount int     `json:"targetThreadCount"`
	ThreadCount       int     `json:"threadCount"`
	JobCount          int     `json:"jobCount"`
	JobRate           float64 `json:"jobRate"`
}

type scaler struct {
	workers map[string]*worker

	log       *vlog.Logger
	lock      *sync.RWMutex
	startOnce *sync.Once
}

func newScaler(log *vlog.Logger) *scaler {
	s := &scaler{
		workers:   map[string]*worker{},
		log:       log,
		lock:      &sync.RWMutex{},
		startOnce: &sync.Once{},
	}

	return s
}

func (s *scaler) startAutoscaler() {
	// if this is called (once or many times), start the autoscaler loop

	s.startOnce.Do(func() {
		go func() {
			for {
				s.lock.RLock()

				for _, worker := range s.workers {
					m := worker.metrics()

					// if job queue is double thread pool size, double the thread count
					// until it reaches autoscaleMax, and reverse when job queue is half
					if m.JobCount > m.ThreadCount*2 || m.JobRate > float64(m.ThreadCount*2) {
						if m.ThreadCount*2 <= worker.options.autoscaleMax {
							worker.setThreadCount(m.ThreadCount * 2)
						} else {
							worker.setThreadCount(worker.options.autoscaleMax)
						}
					} else if m.JobCount < m.ThreadCount/2 && m.JobRate < float64(m.ThreadCount/2) {
						if m.ThreadCount/2 > worker.options.poolSize {
							worker.setThreadCount(m.ThreadCount / 2)
						} else {
							worker.setThreadCount(worker.options.poolSize)
						}
					}
				}

				s.lock.RUnlock()
				time.Sleep(time.Millisecond * 500)
			}
		}()
	})
}

func (s *scaler) addWorker(jobType string, wk *worker) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.workers[jobType] = wk

	go func() {
		if err := wk.start(); err != nil {
			s.log.Error(errors.Wrapf(err, "failed to start %s worker", jobType))
		}
	}()
}

func (s *scaler) removeWorker(jobType string) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	worker, exists := s.workers[jobType]
	if !exists {
		// make this a no-op
		return nil
	}

	delete(s.workers, jobType)

	if err := worker.stop(); err != nil {
		return errors.Wrap(err, "failed to worker.stop")
	}

	return nil
}

func (s *scaler) findWorker(jobType string) *worker {
	s.lock.RLock()
	defer s.lock.RUnlock()

	if s.workers == nil {
		return nil
	}

	if w, ok := s.workers[jobType]; ok {
		return w
	}

	return nil
}

func (s *scaler) metrics() ScalerMetrics {
	s.lock.RLock()
	defer s.lock.RUnlock()

	m := ScalerMetrics{
		TotalThreadCount: 0,
		TotalJobCount:    0,
		Workers:          map[string]WorkerMetrics{},
	}

	for name, w := range s.workers {
		metrics := w.metrics()

		m.TotalThreadCount += metrics.ThreadCount
		m.TotalJobCount += metrics.JobCount
		m.Workers[name] = metrics
	}

	return m
}
