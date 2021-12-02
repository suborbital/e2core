package sequence

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/pkg/errors"
	"github.com/suborbital/atmo/atmo/coordinator/executor"
	"github.com/suborbital/atmo/directive/executable"
	"github.com/suborbital/grav/grav"
	"github.com/suborbital/reactr/request"
	"github.com/suborbital/reactr/rt"
	"github.com/suborbital/vektor/vk"
	"github.com/suborbital/vektor/vlog"
)

type Sequence struct {
	steps []executable.Executable

	exec *executor.Executor

	req *request.CoordinatedRequest

	ctx *vk.Ctx
	log *vlog.Logger
}

type FnResult struct {
	FQFN     string                       `json:"fqfn"`
	Key      string                       `json:"key"`
	Response *request.CoordinatedResponse `json:"response"`
	RunErr   *rt.RunErr                   `json:"runErr"`  // runErr is an error returned from a Runnable
	ExecErr  error                        `json:"execErr"` // err is an annoying workaround that allows runGroup to propogate non-RunErrs out of its loop. Should be refactored when possible.
}

func New(steps []executable.Executable, exec *executor.Executor, ctx *vk.Ctx) *Sequence {
	s := &Sequence{
		steps: steps,
		exec:  exec,
		ctx:   ctx,
		log:   ctx.Log,
	}

	if exec != nil {
		// set messages received by the executor to be handled by the sequence
		exec.UseCallback(s.handleMessage)
	}

	return s
}

// Execute returns the "final state" of a Sequence. If the state's err is not nil, it means a runnable returned an error, and the Directive indicates the Sequence should return.
// if exec itself actually returns an error other than ErrSequenceRunErr, it means there was a problem executing the Sequence as described, and should be treated as such.
func (seq *Sequence) Execute(req *request.CoordinatedRequest) error {
	stepsJSON, err := json.Marshal(seq.steps)
	if err != nil {
		return errors.Wrap(err, "failed to Marshal step")
	}

	req.SequenceJSON = stepsJSON

	seq.req = req

	for {
		// continue running steps until the sequence is complete
		if err := seq.ExecuteNext(req); err != nil {
			if err == executable.ErrSequenceCompleted {
				break
			}

			return err
		}
	}

	return nil
}

// ExecuteNext executes the "next" step (i.e. the first un-completed step) in the sequence
func (seq *Sequence) ExecuteNext(req *request.CoordinatedRequest) error {
	step := seq.NextStep()

	if step == nil {
		return executable.ErrSequenceCompleted
	}

	return seq.executeStep(step, req)
}

// NextStep returns the first un-complete step, nil if the sequence is over
func (seq *Sequence) NextStep() *executable.Executable {
	var step *executable.Executable

	for i, s := range seq.steps {
		// find the first "uncompleted" step
		if !s.Completed {
			step = &seq.steps[i]
			break
		}
	}

	return step
}

// executeStep uses the configured Executor to run the provided handler step. The sequence state and any errors are returned.
// State is also loaded into the object pointed to by req, and the `Completed` field is set on the Executable pointed to by step.
func (seq *Sequence) executeStep(step *executable.Executable, req *request.CoordinatedRequest) error {
	stateJSON, err := stateJSONForStep(req, *step)
	if err != nil {
		return errors.Wrap(err, "failed to stateJSONForStep")
	}

	stepResults := []FnResult{}

	if step.IsFn() {
		seq.log.Debug("running single fn", step.FQFN)

		singleResult, err := seq.ExecSingleFn(step.CallableFn, stateJSON)
		if err != nil {
			return err
		} else if singleResult != nil {
			// in rare cases, this can be nil and so only append if not
			stepResults = append(stepResults, *singleResult)
		}

	} else if step.IsGroup() {
		seq.log.Debug("running group")

		// if the step is a group, run them all concurrently and collect the results
		groupResults, err := seq.ExecGroup(step.Group, stateJSON)
		if err != nil {
			return err
		}

		stepResults = append(stepResults, groupResults...)
	}

	// set the completed value as the functions have been executed
	step.Completed = true

	// determine if error handling results in a return
	if err := seq.handleStepErrs(stepResults, step); err != nil {
		return err
	}

	// set correct state based on the step's results
	seq.handleStepResults(stepResults)

	return nil
}

func (seq *Sequence) handleStepResults(stepResults []FnResult) {
	for _, result := range stepResults {

		seq.req.State[result.Key] = result.Response.Output

		// check if the Runnable set any response headers
		if result.Response.RespHeaders != nil {
			for k, v := range result.Response.RespHeaders {
				seq.req.RespHeaders[k] = v
			}
		}
	}
}

func (seq *Sequence) handleStepErrs(results []FnResult, step *executable.Executable) error {
	for _, result := range results {
		if result.RunErr == nil {
			continue
		}

		if err := step.ShouldReturn(result.RunErr.Code); err != nil {
			seq.log.Error(errors.Wrapf(err, "returning after error from %s", result.FQFN))

			return result.RunErr
		} else {
			seq.log.Info("continuing after error from", result.FQFN)
			seq.req.State[result.Key] = []byte(result.RunErr.Error())
		}
	}

	return nil
}

// this is called by the executor:
// sequence -> exec.do -> handleMessage (n times) -> .do returns to sequence
func (seq *Sequence) handleMessage(msg grav.Message) error {
	result := FnResult{}
	if err := json.Unmarshal(msg.Data(), &result); err != nil {
		return errors.Wrap(err, "failed to Unmarshal FnResult")
	}

	step := seq.NextStep()
	if step == nil {
		return executable.ErrSequenceCompleted
	} else if step.IsFn() {
		seq.log.Info("handling nextStep from handleMessage")
		step.SetCompleted(true)
	} else {
		seq.log.Warn("cannot handle message from group step")
		return nil
	}

	stepResults := []FnResult{result}

	// determine if error handling results in a return
	if err := seq.handleStepErrs(stepResults, step); err != nil {
		return err
	}

	// set correct state based on the step's results
	seq.handleStepResults(stepResults)

	// check nextstep again
	step = seq.NextStep()
	if step == nil {
		return executable.ErrSequenceCompleted
	}

	return nil
}

func stateJSONForStep(req *request.CoordinatedRequest, step executable.Executable) ([]byte, error) {
	// based on the step's `with` clause, build the state to pass into the function
	stepState, err := desiredState(step.With, req.State)
	if err != nil {
		return nil, vk.Wrap(http.StatusInternalServerError, errors.Wrap(err, "failed to build desiredState"))
	}

	stepReq := request.CoordinatedRequest{
		Method:       req.Method,
		URL:          req.URL,
		ID:           req.ID,
		Body:         req.Body,
		Headers:      req.Headers,
		RespHeaders:  req.RespHeaders,
		Params:       req.Params,
		State:        stepState,
		SequenceJSON: req.SequenceJSON,
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
