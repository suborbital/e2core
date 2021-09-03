package coordinator

import (
	"encoding/json"
	"time"

	"github.com/pkg/errors"
	"github.com/suborbital/atmo/directive"
	"github.com/suborbital/reactr/request"
	"github.com/suborbital/reactr/rt"
)

var ErrMissingFQFN = errors.New("callableFn missing FQFN")

func (seq sequence) runSingleFn(fn directive.CallableFn, reqJSON []byte) (*fnResult, error) {
	start := time.Now()
	defer func() {
		seq.log.Debug("fn", fn.Fn, "executed in", time.Since(start).Milliseconds(), "ms")
	}()

	if fn.FQFN == "" {
		return nil, ErrMissingFQFN
	}

	var jobResult []byte
	var runErr *rt.RunErr

	// Do will execute the job locally if possible or find a remote peer to execute it
	res, err := seq.exec.Do(fn.FQFN, reqJSON, seq.ctx)
	if err != nil {
		// check if the error type is rt.RunErr, because those are handled differently
		returnedErr := &rt.RunErr{}
		if errors.As(err, returnedErr) {
			runErr = returnedErr
		} else {
			return nil, errors.Wrap(err, "failed to exec.Do")
		}
	} else {
		jobResult = res.([]byte)
	}

	// runErr would be an actual error returned from a function
	if runErr != nil {
		seq.log.Debug("fn", fn.Fn, "returned an error")
	} else if jobResult == nil {
		seq.log.Debug("fn", fn.Fn, "returned a nil result")
	}

	key := key(fn)
	cResponse := &request.CoordinatedResponse{}

	if jobResult != nil {
		if err := json.Unmarshal(jobResult, cResponse); err != nil {
			// handle backwards-compat
			cResponse.Output = jobResult
		}
	}

	result := &fnResult{
		fqfn:     fn.FQFN,
		key:      key,
		response: cResponse,
		runErr:   runErr,
	}

	return result, nil
}

func key(fn directive.CallableFn) string {
	key := fn.Fn

	if fn.As != "" {
		key = fn.As
	}

	return key
}
