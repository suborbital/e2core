package coordinator

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/pkg/errors"
	"github.com/suborbital/grav/grav"
	"github.com/suborbital/reactr/directive"
	"github.com/suborbital/reactr/request"
	"github.com/suborbital/reactr/rt"
	"github.com/suborbital/vektor/vk"
	"github.com/suborbital/vektor/vlog"
)

type fqfnFunc func(string) (string, error)
type connectFunc func() *grav.Pod

// ErrSequenceRunErr is returned when the sequence returned due to a Runnable's RunErr
var ErrSequenceRunErr = errors.New("sequence resulted in a RunErr")

type sequence struct {
	steps []directive.Executable

	connectFunc connectFunc
	fqfn        fqfnFunc

	log *vlog.Logger
}

type sequenceState struct {
	state map[string][]byte
	err   *rt.RunErr
}

type fnResult struct {
	fqfn     string
	key      string
	response *request.CoordinatedResponse
	runErr   *rt.RunErr // runErr is an error returned from a Runnable
	err      error      // err is an annoying "hack" that allows runGroup to propogate errors out of its loop. Should be refactored when possible.
}

func newSequence(steps []directive.Executable, connect connectFunc, fqfn fqfnFunc, log *vlog.Logger) *sequence {
	s := &sequence{
		steps:       steps,
		connectFunc: connect,
		fqfn:        fqfn,
		log:         log,
	}

	return s
}

// exec will return the "final state" of a sequence. If the state's err is not nil, it means a runnable returned an error, and the Directive indicates the sequence should return.
// if exec itself actually returns an error other than ErrSequenceRunErr, it means there was a problem executing the sequence as described, and should be treated as such.
func (seq *sequence) exec(req *request.CoordinatedRequest) (*sequenceState, error) {
	for _, step := range seq.steps {
		stateJSON, err := stateJSONForStep(req, step)
		if err != nil {
			seq.log.Error(errors.Wrap(err, "failed to stateJSONForStep"))
			return nil, err
		}

		stepResults := []fnResult{}

		if step.IsFn() {
			singleResult, err := seq.runSingleFn(step.CallableFn, stateJSON)
			if err != nil {
				return nil, err
			}

			stepResults = append(stepResults, *singleResult)
		} else {
			// if the step is a group, run them all concurrently and collect the results
			groupResults, err := seq.runGroup(step.Group, stateJSON)
			if err != nil {
				return nil, err
			}

			stepResults = append(stepResults, groupResults...)
		}

		for _, result := range stepResults {
			if result.runErr != nil {
				if step.OnErr != nil {
					shouldErr := false

					// if the error code is listed as return, or any/other indicates a return, then create an erroring state object and return it.

					if len(step.OnErr.Code) > 0 {
						if val, ok := step.OnErr.Code[result.runErr.Code]; ok && val == "return" {
							shouldErr = true
						} else if !ok && step.OnErr.Other == "return" {
							shouldErr = true
						}
					} else if step.OnErr.Any == "return" {
						shouldErr = true
					}

					if shouldErr {
						seq.log.Error(errors.Wrapf(result.runErr, "returning with error from %s", result.fqfn))

						state := &sequenceState{
							err: result.runErr,
						}

						return state, ErrSequenceRunErr
					} else {
						seq.log.Info("continuing after error from", result.fqfn)
					}
				} else {
					// set the error's JSON as the state value
					req.State[result.key] = []byte(result.runErr.Error())
				}
			} else {
				req.State[result.key] = result.response.Output
			}

			// check if the Runnable set any response headers
			if result.response.RespHeaders != nil {
				for k, v := range result.response.RespHeaders {
					req.RespHeaders[k] = v
				}
			}
		}
	}

	state := &sequenceState{
		state: req.State,
	}

	return state, nil
}

func (seq sequence) runSingleFn(fn directive.CallableFn, body []byte) (*fnResult, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		seq.log.Debug("fn", fn.Fn, fmt.Sprintf("executed in %d ms", duration.Milliseconds()))
	}()

	// calculate the FQFN
	fqfn, err := seq.fqfn(fn.Fn)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to FQFN for fn %s", fn.Fn)
	}

	pod := seq.connectFunc()
	defer pod.Disconnect()

	// compose a message containing the serialized request state, and send it via Grav
	// for the appropriate meshed Reactr to handle. It may be handled by self if appropriate.
	jobMsg := grav.NewMsg(fqfn, body)

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
			return nil, errors.Wrapf(err, "fn %s timed out", fn.Fn)
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
		fqfn:     fqfn,
		key:      key,
		response: cResponse,
		runErr:   runErr,
	}

	return result, nil
}

// runGroup runs a group of functions
// this is all more complicated than it needs to be, Grav should be doing more of the work for us here
func (seq *sequence) runGroup(fns []directive.CallableFn, body []byte) ([]fnResult, error) {
	start := time.Now()
	defer func() {
		seq.log.Debug("group", fmt.Sprintf("executed in %d ms", time.Since(start).Milliseconds()))
	}()

	resultChan := make(chan fnResult, len(fns))

	// for now we'll use a bit of a kludgy means of running all of the group fns concurrently
	// in the future, we should send out all of the messages first, then have some new Grav
	// functionality to collect all the responses, probably using the parent ID.
	for i := range fns {
		fn := fns[i]
		seq.log.Debug("running fn", fn.Fn, "from group")

		go func() {
			res, err := seq.runSingleFn(fn, body)
			if err != nil {
				seq.log.Error(errors.Wrap(err, "failed to runSingleFn"))
				resultChan <- fnResult{err: err}
			} else {
				resultChan <- *res
			}
		}()
	}

	results := []fnResult{}
	respCount := 0
	timeoutChan := time.After(30 * time.Second)

	for respCount < len(fns) {
		select {
		case result := <-resultChan:
			if result.err != nil {
				// if there was an error running the funciton, return that error
				return nil, result.err
			}

			results = append(results, result)
		case <-timeoutChan:
			return nil, errors.New("fn group timed out")
		}

		respCount++
	}

	return results, nil
}

func stateJSONForStep(req *request.CoordinatedRequest, step directive.Executable) ([]byte, error) {
	// the desired state is cached, so after the first call this is very efficient
	desired, err := step.ParseWith()
	if err != nil {
		return nil, vk.Wrap(http.StatusInternalServerError, errors.Wrap(err, "failed to ParseWith"))
	}

	// based on the step's `with` clause, build the state to pass into the function
	stepState, err := desiredState(desired, req.State)
	if err != nil {
		return nil, vk.Wrap(http.StatusInternalServerError, errors.Wrap(err, "failed to build desiredState"))
	}

	stepReq := request.CoordinatedRequest{
		Method:  req.Method,
		URL:     req.URL,
		ID:      req.ID,
		Body:    req.Body,
		Headers: req.Headers,
		Params:  req.Params,
		State:   stepState,
	}

	stateJSON, err := stepReq.ToJSON()
	if err != nil {
		return nil, vk.Wrap(http.StatusInternalServerError, errors.Wrap(err, "failed to ToJSON Request State"))
	}

	return stateJSON, nil
}

func desiredState(desired []directive.Alias, state map[string][]byte) (map[string][]byte, error) {
	if desired == nil || len(desired) == 0 {
		return state, nil
	}

	desiredState := map[string][]byte{}

	for _, a := range desired {
		val, exists := state[a.Key]
		if !exists {
			return nil, fmt.Errorf("failed to build desired state, %s does not exists in handler state", a.Key)
		}

		desiredState[a.Alias] = val
	}

	return desiredState, nil
}

func key(fn directive.CallableFn) string {
	key := fn.Fn

	if fn.As != "" {
		key = fn.As
	}

	return key
}
