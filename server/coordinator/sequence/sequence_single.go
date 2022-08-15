package sequence

import (
	"time"

	"github.com/pkg/errors"

	"github.com/suborbital/appspec/tenant/executable"
	"github.com/suborbital/deltav/scheduler"
	"github.com/suborbital/deltav/server/request"
)

var ErrMissingFQFN = errors.New("executableMod missing FQFN")

func (seq *Sequence) ExecSingleMod(mod executable.ExecutableMod) (*FnResult, error) {
	start := time.Now()
	defer func() {
		seq.log.Debug("mod", mod.FQMN, "executed in", time.Since(start).Milliseconds(), "ms")
	}()

	if mod.FQMN == "" {
		return nil, ErrMissingFQFN
	}

	var runErr scheduler.RunErr

	// Do will execute the job locally if possible or find a remote peer to execute it.
	res, err := seq.exec.Do(mod.FQMN, seq.req, seq.ctx, seq.handleMessage)
	if err != nil {
		// check if the error type is scheduler.RunErr, because those are handled differently.
		if returnedErr, isRunErr := err.(scheduler.RunErr); isRunErr {
			runErr = returnedErr
		} else {
			return nil, errors.Wrap(err, "failed to exec.Do")
		}
	} else if res == nil {
		seq.log.Debug("fn", mod.FQMN, "returned a nil result")

		return nil, nil
	}

	// runErr would be an actual error returned from a function
	// should find a better way to determine if a RunErr is "non-nil".
	if runErr.Code != 0 || runErr.Message != "" {
		seq.log.Debug("fn", mod.FQMN, "returned an error")
	}

	cResponse := &request.CoordinatedResponse{}

	if res != nil {
		cResponse = res.(*request.CoordinatedResponse)
	}

	result := &FnResult{
		FQFN:     mod.FQMN,
		Key:      mod.Key(),
		Response: cResponse,
		RunErr:   runErr,
	}

	return result, nil
}
