package sequence

import (
	"time"

	"github.com/pkg/errors"

	"github.com/suborbital/velocity/directive/executable"
	"github.com/suborbital/velocity/scheduler"
	"github.com/suborbital/velocity/server/request"
)

var ErrMissingFQFN = errors.New("callableFn missing FQFN")

func (seq *Sequence) ExecSingleFn(fn executable.CallableFn) (*FnResult, error) {
	start := time.Now()
	defer func() {
		seq.log.Debug("fn", fn.Fn, "executed in", time.Since(start).Milliseconds(), "ms")
	}()

	if fn.FQFN == "" {
		return nil, ErrMissingFQFN
	}

	var runErr scheduler.RunErr

	// Do will execute the job locally if possible or find a remote peer to execute it.
	res, err := seq.exec.Do(fn.FQFN, seq.req, seq.ctx, seq.handleMessage)
	if err != nil {
		// check if the error type is scheduler.RunErr, because those are handled differently.
		if returnedErr, isRunErr := err.(scheduler.RunErr); isRunErr {
			runErr = returnedErr
		} else {
			return nil, errors.Wrap(err, "failed to exec.Do")
		}
	} else if res == nil {
		seq.log.Debug("fn", fn.Fn, "returned a nil result")

		return nil, nil
	}

	// runErr would be an actual error returned from a function
	// should find a better way to determine if a RunErr is "non-nil".
	if runErr.Code != 0 || runErr.Message != "" {
		seq.log.Debug("fn", fn.Fn, "returned an error")
	}

	cResponse := &request.CoordinatedResponse{}

	if res != nil {
		cResponse = res.(*request.CoordinatedResponse)
	}

	result := &FnResult{
		FQFN:     fn.FQFN,
		Key:      fn.Key(),
		Response: cResponse,
		RunErr:   runErr,
	}

	return result, nil
}
