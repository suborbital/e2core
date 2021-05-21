package coordinator

import (
	"encoding/json"
	"time"

	"github.com/pkg/errors"
	"github.com/suborbital/atmo/directive"
	"github.com/suborbital/grav/grav"
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

	pod := seq.connectFunc()
	defer pod.Disconnect()

	// compose a message containing the serialized request state, and send it via Grav
	// for the appropriate meshed Reactr to handle. It may be handled by self if appropriate.
	jobMsg := grav.NewMsg(fn.FQFN, reqJSON)

	var jobResult []byte
	var runErr *rt.RunErr

	podErr := pod.Send(jobMsg).WaitUntil(grav.Timeout(30), func(msg grav.Message) error {
		switch msg.Type() {
		case rt.MsgTypeReactrResult:
			// if the Runnable returned a result
			jobResult = msg.Data()
		case rt.MsgTypeReactrRunErr:
			// if the Runnable itself returned an error
			runErr = &rt.RunErr{}
			if err := json.Unmarshal(msg.Data(), runErr); err != nil {
				return errors.Wrap(err, "failed to Unmarshal RunErr")
			}
		case rt.MsgTypeReactrJobErr:
			// if something else caused an error while running this fn
			return errors.New(string(msg.Data()))
		case rt.MsgTypeReactrNilResult:
			// if the Runnable returned nil, do nothing
		}

		return nil
	})

	// podErr would be something that happened whily trying to run a function, not an error returned from a function
	if podErr != nil {
		if podErr == grav.ErrWaitTimeout {
			return nil, errors.Wrapf(podErr, "fn %s timed out", fn.Fn)
		}

		return nil, errors.Wrapf(podErr, "failed to execute fn %s", fn.Fn)
	}

	// runErr would be an actual error returned from a function
	if runErr != nil {
		seq.log.Debug("fn", fn.Fn, "returned an error")
	} else if jobResult == nil {
		seq.log.Debug("fn", fn.Fn, "returned a nil result")
	}

	key := key(fn)

	cResponse := &request.CoordinatedResponse{}
	if err := json.Unmarshal(jobResult, cResponse); err != nil {
		// handle backwards-compat
		cResponse.Output = jobResult
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
