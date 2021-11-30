package sequence

import (
	"fmt"
	"net/http"

	"github.com/pkg/errors"
	"github.com/suborbital/atmo/atmo/coordinator/executor"
	"github.com/suborbital/atmo/directive/executable"
	"github.com/suborbital/reactr/request"
	"github.com/suborbital/reactr/rt"
	"github.com/suborbital/vektor/vk"
	"github.com/suborbital/vektor/vlog"
)

var ErrSequenceCompleted = errors.New("sequence is complete, no steps to run")

type Sequence struct {
	steps []executable.Executable

	exec *executor.Executor

	ctx *vk.Ctx
	log *vlog.Logger
}

type SequenceState struct {
	State map[string][]byte
	Err   *rt.RunErr
}

type FnResult struct {
	FQFN     string
	Key      string
	Response *request.CoordinatedResponse
	RunErr   *rt.RunErr // runErr is an error returned from a Runnable
	Err      error      // err is an annoying workaround that allows runGroup to propogate non-RunErrs out of its loop. Should be refactored when possible.
}

func New(steps []executable.Executable, exec *executor.Executor, ctx *vk.Ctx) *Sequence {
	s := &Sequence{
		steps: steps,
		exec:  exec,
		ctx:   ctx,
		log:   ctx.Log,
	}

	return s
}

// Execute returns the "final state" of a Sequence. If the state's err is not nil, it means a runnable returned an error, and the Directive indicates the Sequence should return.
// if exec itself actually returns an error other than ErrSequenceRunErr, it means there was a problem executing the Sequence as described, and should be treated as such.
func (seq *Sequence) Execute(req *request.CoordinatedRequest) (*SequenceState, error) {
	for {
		// continue running steps until the sequence is complete
		if state, err := seq.ExecuteNext(req); err != nil {
			if err == ErrSequenceCompleted {
				break
			}

			return state, err
		}
	}

	state := &SequenceState{
		State: req.State,
	}

	return state, nil
}

// ExecuteNext executes the "next" step (i.e. the first un-completed step) in the sequence
func (seq *Sequence) ExecuteNext(req *request.CoordinatedRequest) (*SequenceState, error) {
	var step *executable.Executable

	for i, s := range seq.steps {
		// find the first "uncompleted" step
		if !s.Completed {
			step = &seq.steps[i]
			break
		}
	}

	if step == nil {
		return nil, ErrSequenceCompleted
	}

	return seq.executeStep(step, req)
}

// executeStep uses the configured Executor to run the provided handler step. The sequence state and any errors are returned.
// State is also loaded into the object pointed to by req, and the `Completed` field is set on the Executable pointed to by step.
func (seq *Sequence) executeStep(step *executable.Executable, req *request.CoordinatedRequest) (*SequenceState, error) {
	stateJSON, err := stateJSONForStep(req, *step)
	if err != nil {
		seq.log.Error(errors.Wrap(err, "failed to stateJSONForStep"))
		return nil, err
	}

	stepResults := []FnResult{}

	if step.IsFn() {
		seq.log.Debug("running single fn", step.FQFN)

		singleResult, err := seq.RunSingleFn(step.CallableFn, stateJSON)
		if err != nil {
			return nil, err
		} else if singleResult != nil {
			// in rare cases, this can be nil and so only append if not
			stepResults = append(stepResults, *singleResult)
		}

	} else if step.IsGroup() {
		seq.log.Debug("running group")

		// if the step is a group, run them all concurrently and collect the results
		groupResults, err := seq.RunGroup(step.Group, stateJSON)
		if err != nil {
			return nil, err
		}

		stepResults = append(stepResults, groupResults...)
	}

	// set the completed value as the functions have been executed
	step.Completed = true

	for _, result := range stepResults {
		val := result.Response.Output

		// if the Runnable returned an error, handle that here
		if result.RunErr != nil {
			if err := step.ShouldReturn(result.RunErr.Code); err != nil {
				seq.log.Error(errors.Wrapf(err, "returning after error from %s", result.FQFN))

				state := &SequenceState{
					Err: result.RunErr,
				}

				return state, err
			} else {
				seq.log.Info("continuing after error from", result.FQFN)
				val = []byte(result.RunErr.Error())
			}
		}

		req.State[result.Key] = val

		// check if the Runnable set any response headers
		if result.Response.RespHeaders != nil {
			for k, v := range result.Response.RespHeaders {
				req.RespHeaders[k] = v
			}
		}
	}

	return nil, nil
}

func stateJSONForStep(req *request.CoordinatedRequest, step executable.Executable) ([]byte, error) {
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
