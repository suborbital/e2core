package rt

import (
	"sync"
	"time"

	"golang.org/x/sync/singleflight"

	"github.com/pkg/errors"
)

const (
	defaultChanSize = 256
)

// ErrJobTimeout and others are errors related to workers
var (
	ErrJobTimeout = errors.New("job timeout")
)

type worker struct {
	runner   Runnable
	workChan chan *Job
	options  workerOpts

	defaultCaps Capabilities

	targetThreadCount int
	threads           []*workThread

	lock      *sync.RWMutex
	reconcile *singleflight.Group
	rate      *rateTracker
}

// newWorker creates a new goWorker
func newWorker(runner Runnable, caps Capabilities, opts workerOpts) *worker {
	w := &worker{
		runner:            runner,
		workChan:          make(chan *Job, defaultChanSize),
		options:           opts,
		defaultCaps:       caps,
		targetThreadCount: opts.poolSize,
		threads:           []*workThread{},
		lock:              &sync.RWMutex{},
		reconcile:         &singleflight.Group{},
		rate:              newRateTracker(),
	}

	return w
}

func (w *worker) schedule(job *Job) {
	if job.caps == nil {
		// make a copy so internals of the Capabilites aren't shared
		caps := w.defaultCaps
		job.caps = &caps
	}

	go func() {
		if err := w.reconcilePoolSize(); err != nil {
			job.result.sendErr(errors.Wrap(err, "failed to reconcilePoolSize"))
			return
		}

		w.workChan <- job
		w.rate.add()
	}()
}

// start ensures the worker is ready to receive jobs
func (w *worker) start() error {
	if w.options.preWarm {
		if err := w.reconcilePoolSize(); err != nil {
			return errors.Wrap(err, "failed to reconcilePoolSize")
		}
	}

	return nil
}

func (w *worker) stop() error {
	// set the poolsize to 0 and give the workers a chance to wind down
	return w.setThreadCount(0)
}

func (w *worker) setThreadCount(size int) error {
	w.targetThreadCount = size

	if err := w.reconcilePoolSize(); err != nil {
		return errors.Wrap(err, "failed to reconcilePoolSize")
	}

	return nil
}

// reconcilePoolSize starts and stops runners until `poolSize` are active
func (w *worker) reconcilePoolSize() error {
	attempts := 0

	shouldReturn := func() bool {
		if attempts > w.options.numRetries {
			return true
		} else {
			attempts++
			time.Sleep(time.Second * time.Duration(w.options.retrySecs))
			return false
		}
	}

	// this is wrapped in a singleFlight to ensure we're only attempting this
	// once at any given time, because we don't want a sudden influx of jobs
	// to wreak havoc on the Runnable (especially if it needs to provision resources)
	_, err, _ := w.reconcile.Do("reconcile", func() (interface{}, error) {
		for {
			w.lock.RLock()
			actualThreadCount := len(w.threads)
			w.lock.RUnlock()

			if actualThreadCount < w.targetThreadCount {
				if err := w.addThread(); err != nil {
					if shouldReturn() {
						return nil, errors.Wrap(err, "failed to addThread more than numRetries")
					}
				}
			} else if actualThreadCount > w.targetThreadCount {
				if err := w.removeThread(); err != nil {
					if shouldReturn() {
						return nil, errors.Wrap(err, "failed to removeThread more than numRetries")
					}
				}
			} else {
				break
			}
		}

		return nil, nil
	})

	if err != nil {
		return err
	}

	return nil
}

// addThread starts a new thread and adds it to the thread pool
func (w *worker) addThread() error {
	w.lock.Lock()
	defer w.lock.Unlock()

	wt := newWorkThread(w.runner, w.workChan, w.options.jobTimeoutSeconds)

	// give the runner opportunity to provision resources if needed
	if err := w.runner.OnChange(ChangeTypeStart); err != nil {
		return errors.Wrap(err, "runnable returned OnChange error")
	}

	wt.run()

	w.threads = append(w.threads, wt)

	return nil
}

// removeThread removes a thread and terminates it
func (w *worker) removeThread() error {
	w.lock.Lock()
	defer w.lock.Unlock()

	wt := w.threads[len(w.threads)-1]
	wt.cancelFunc()

	// give the runner opportunity to de-provision resources if needed
	if err := w.runner.OnChange(ChangeTypeStop); err != nil {
		return errors.Wrap(err, "runnable returned OnChange error")
	}

	w.threads = w.threads[:len(w.threads)-1]

	return nil
}

func (w *worker) metrics() WorkerMetrics {
	w.lock.RLock()
	defer w.lock.RUnlock()

	m := WorkerMetrics{
		TargetThreadCount: w.targetThreadCount,
		ThreadCount:       len(w.threads),
		JobCount:          len(w.workChan),
		JobRate:           w.rate.average(),
	}

	return m
}

type workerOpts struct {
	jobType           string
	poolSize          int
	autoscaleMax      int
	jobTimeoutSeconds int
	numRetries        int
	retrySecs         int
	preWarm           bool
}

func defaultOpts(jobType string) workerOpts {
	o := workerOpts{
		jobType:           jobType,
		poolSize:          1,
		autoscaleMax:      1,
		jobTimeoutSeconds: 0,
		numRetries:        5,
		retrySecs:         3,
		preWarm:           false,
	}

	return o
}
