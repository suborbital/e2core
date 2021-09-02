package coordinator

import (
	"fmt"
	"net/http"

	"github.com/pkg/errors"
	"github.com/suborbital/atmo/atmo/coordinator/executor"
	"github.com/suborbital/atmo/directive"
	"github.com/suborbital/reactr/request"
	"github.com/suborbital/reactr/rt"
	"github.com/suborbital/vektor/vk"
	"github.com/suborbital/vektor/vlog"
)

// ErrSequenceRunErr is returned when the sequence returned due to a Runnable's RunErr
var ErrSequenceRunErr = errors.New("sequence resulted in a RunErr")

type sequence struct {
	steps []directive.Executable

	exec *executor.Executor

	ctx *vk.Ctx
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
	err      error      // err is an annoying workaround that allows runGroup to propogate non-RunErrs out of its loop. Should be refactored when possible.
}

func newSequence(steps []directive.Executable, exec *executor.Executor, ctx *vk.Ctx) *sequence {
	s := &sequence{
		steps: steps,
		exec:  exec,
		ctx:   ctx,
		log:   ctx.Log,
	}

	return s
}

// execute will return the "final state" of a sequence. If the state's err is not nil, it means a runnable returned an error, and the Directive indicates the sequence should return.
// if exec itself actually returns an error other than ErrSequenceRunErr, it means there was a problem executing the sequence as described, and should be treated as such.
func (seq *sequence) execute(req *request.CoordinatedRequest) (*sequenceState, error) {
	for _, step := range seq.steps {
		stateJSON, err := stateJSONForStep(req, step)
		if err != nil {
			seq.log.Error(errors.Wrap(err, "failed to stateJSONForStep"))
			return nil, err
		}

		stepResults := []fnResult{}

		if step.IsFn() {
			seq.log.Debug("running single fn", step.FQFN)

			singleResult, err := seq.runSingleFn(step.CallableFn, stateJSON)
			if err != nil {
				return nil, err
			} else if singleResult != nil {
				// in rare cases, this can be nil and so only append if not
				stepResults = append(stepResults, *singleResult)
			}

		} else if step.IsGroup() {
			seq.log.Debug("running group")

			// if the step is a group, run them all concurrently and collect the results
			groupResults, err := seq.runGroup(step.Group, stateJSON)
			if err != nil {
				return nil, err
			}

			stepResults = append(stepResults, groupResults...)
		} else if step.IsForEach() {
			seq.log.Debug("running foreach")

			// if the step is a forEach, run it and add the result to state
			// passing in the dereferenced request, as forEach needs to add temporary
			// state fields for each value it iterates over
			forEachResult, err := seq.runForEach(step.ForEach, *req)
			if err != nil {
				return nil, err
			} else if forEachResult != nil {
				// in rare cases, this can be nil and so only append if not
				stepResults = append(stepResults, *forEachResult)
			}
		}

		for _, result := range stepResults {
			val := result.response.Output

			// if the Runnable returned an error, handle that here
			if result.runErr != nil {

				// if the step is an Fn, use its onErr
				// if the step is a forEach, use that
				onErr := step.OnErr
				if onErr == nil && step.ForEach != nil {
					onErr = step.ForEach.OnErr
				}

				if onErr != nil {
					if state, err := seq.shouldReturn(onErr, result); err != nil {
						// shouldReturn returns ErrSequenceRunErr, so no need to wrap
						return state, err
					} else {
						seq.log.Info("continuing after error from", result.fqfn)
						val = []byte(result.runErr.Error())
					}
				} else {
					// the default if no onErr is set is to return, so put the error JSON in state
					seq.log.Info("returning after error from", result.fqfn)
					state := &sequenceState{err: result.runErr}

					return state, ErrSequenceRunErr
				}
			}

			req.State[result.key] = val

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

func (seq *sequence) shouldReturn(onErr *directive.FnOnErr, result fnResult) (*sequenceState, error) {
	shouldErr := true

	// if the error code is listed as return, or any/other indicates a return, then create an erroring state object and return it.

	if len(onErr.Code) > 0 {
		if val, ok := onErr.Code[result.runErr.Code]; ok && val == "continue" {
			shouldErr = false
		} else if !ok && onErr.Other == "continue" {
			shouldErr = false
		}
	} else if onErr.Any == "continue" {
		shouldErr = false
	}

	if shouldErr {
		seq.log.Error(errors.Wrapf(result.runErr, "returning with error from %s", result.fqfn))

		state := &sequenceState{
			err: result.runErr,
		}

		return state, ErrSequenceRunErr
	}

	return nil, nil
}

func stateJSONForStep(req *request.CoordinatedRequest, step directive.Executable) ([]byte, error) {
	// based on the step's `with` clause, build the state to pass into the function
	stepState, err := desiredState(step.With, req.State)
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

func desiredState(desired map[string]string, state map[string][]byte) (map[string][]byte, error) {
	if len(desired) == 0 {
		return state, nil
	}

	desiredState := map[string][]byte{}

	for alias, key := range desired {
		val, exists := state[key]
		if !exists {
			return nil, fmt.Errorf("failed to build desired state, %s does not exists in handler state", key)
		}

		desiredState[alias] = val
	}

	return desiredState, nil
}
