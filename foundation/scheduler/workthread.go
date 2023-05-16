package scheduler

import (
	"context"
	"time"

	"github.com/suborbital/e2core/foundation/tracing"
)

type workThread struct {
	runner         Runnable
	workChan       chan *Job
	doFunc         coreDoFunc
	timeoutSeconds int
	context        context.Context
	cancelFunc     context.CancelFunc
}

func newWorkThread(runner Runnable, doFunc coreDoFunc, workChan chan *Job, timeoutSeconds int) *workThread {
	ctx, cancelFunc := context.WithCancel(context.Background())

	wt := &workThread{
		runner:         runner,
		workChan:       workChan,
		doFunc:         doFunc,
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
			inJob := <-wt.workChan
			ctx, span := tracing.Tracer.Start(inJob.Context(), "workthread.run in scheduler")

			job := inJob.WithContext(ctx)

			var err error

			workCtx := newCtx(wt.doFunc)

			var result interface{}

			if wt.timeoutSeconds == 0 {
				// we pass in a dereferenced job so that the Runner cannot modify it
				result, err = wt.runner.Run(job, workCtx)
			} else {
				result, err = wt.runWithTimeout(&job, workCtx)
			}

			if err != nil {
				job.result.sendErr(err)
				continue
			}

			job.result.sendResult(result)

			span.End()
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
