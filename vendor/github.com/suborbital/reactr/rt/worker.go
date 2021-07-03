package rt

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

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

	threads    []*workThread
	threadLock sync.Mutex

	started atomic.Value
}

// newWorker creates a new goWorker
func newWorker(runner Runnable, caps Capabilities, opts workerOpts) *worker {
	w := &worker{
		runner:      runner,
		workChan:    make(chan *Job, defaultChanSize),
		options:     opts,
		defaultCaps: caps,
		threads:     make([]*workThread, opts.poolSize),
		threadLock:  sync.Mutex{},
		started:     atomic.Value{},
	}

	w.started.Store(false)

	return w
}

func (w *worker) schedule(job *Job) {
	if job.caps == nil {
		job.caps = &w.defaultCaps
	}

	go func() {
		w.workChan <- job
	}()
}

func (w *worker) start(doFunc coreDoFunc) error {
	// this should only be run once per worker, unless startup fails the first time
	if isStarted := w.started.Load().(bool); isStarted {
		return nil
	}

	w.started.Store(true)

	started := 0
	attempts := 0

	for {
		// fill the "pool" with workThreads
		for i := started; i < w.options.poolSize; i++ {
			wt := newWorkThread(w.runner, w.workChan, w.options.jobTimeoutSeconds)

			// give the runner opportunity to provision resources if needed
			if err := w.runner.OnChange(ChangeTypeStart); err != nil {
				fmt.Println(errors.Wrapf(err, "Runnable returned OnStart error, will retry in %ds", w.options.retrySecs))
				break
			} else {
				started++
			}

			wt.run()

			w.threads[i] = wt
		}

		if started == w.options.poolSize {
			break
		} else {
			if attempts >= w.options.numRetries {
				if started == 0 {
					// if no threads were able to start, ensure that
					// the next job causes another attempt
					w.started.Store(false)
				}

				return fmt.Errorf("attempted to start worker %d times, Runnable returned error each time", w.options.numRetries)
			}

			attempts++
			<-time.After(time.Duration(time.Second * time.Duration(w.options.retrySecs)))
		}
	}

	return nil
}

func (w *worker) isStarted() bool {
	return w.started.Load().(bool)
}

type workThread struct {
	runner         Runnable
	workChan       chan *Job
	timeoutSeconds int
	context        context.Context
	cancelFunc     context.CancelFunc
}

func newWorkThread(runner Runnable, workChan chan *Job, timeoutSeconds int) *workThread {
	ctx, cancelFunc := context.WithCancel(context.Background())

	wt := &workThread{
		runner:         runner,
		workChan:       workChan,
		timeoutSeconds: timeoutSeconds,
		context:        ctx,
		cancelFunc:     cancelFunc,
	}

	return wt
}

func (wt *workThread) run() {
	go func() {
		for {
			// die if the context has been cancelled
			if wt.context.Err() != nil {
				break
			}

			// wait for the next job
			job := <-wt.workChan
			var err error

			// TODO: check to see if the workThread pool is sufficient, and attempt to fill it if not

			ctx := newCtx(job.caps)

			var result interface{}

			if wt.timeoutSeconds == 0 {
				// we pass in a dereferenced job so that the Runner cannot modify it
				result, err = wt.runner.Run(*job, ctx)
			} else {
				result, err = wt.runWithTimeout(job, ctx)
			}

			if err != nil {
				job.result.sendErr(err)
				continue
			}

			job.result.sendResult(result)
		}
	}()
}

func (wt *workThread) runWithTimeout(job *Job, ctx *Ctx) (interface{}, error) {
	resultChan := make(chan interface{})
	errChan := make(chan error)

	go func() {
		// we pass in a dereferenced job so that the Runner cannot modify it
		result, err := wt.runner.Run(*job, ctx)
		if err != nil {
			errChan <- err
		} else {
			resultChan <- result
		}
	}()

	select {
	case result := <-resultChan:
		return result, nil
	case err := <-errChan:
		return nil, err
	case <-time.After(time.Duration(time.Second * time.Duration(wt.timeoutSeconds))):
		return nil, ErrJobTimeout
	}
}

func (wt *workThread) Stop() {
	wt.cancelFunc()
}

type workerOpts struct {
	jobType           string
	poolSize          int
	jobTimeoutSeconds int
	numRetries        int
	retrySecs         int
	preWarm           bool
}

func defaultOpts(jobType string) workerOpts {
	o := workerOpts{
		jobType:           jobType,
		poolSize:          1,
		jobTimeoutSeconds: 0,
		retrySecs:         3,
		numRetries:        5,
		preWarm:           false,
	}

	return o
}
