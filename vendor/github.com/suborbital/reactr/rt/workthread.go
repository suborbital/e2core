package rt

import (
	"context"
	"time"
)

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
