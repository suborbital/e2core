package sequence

import (
	"encoding/json"
	"sync"

	"github.com/pkg/errors"

	"github.com/suborbital/grav/grav"
	"github.com/suborbital/vektor/vk"
	"github.com/suborbital/vektor/vlog"
	"github.com/suborbital/velocity/directive/executable"
	"github.com/suborbital/velocity/scheduler"
	"github.com/suborbital/velocity/server/coordinator/executor"
	"github.com/suborbital/velocity/server/request"
)

type Sequence struct {
	steps []Step

	exec executor.Executor

	req *request.CoordinatedRequest

	ctx *vk.Ctx
	log *vlog.Logger

	lock sync.Mutex // need to ensure writes to req.State are kept serial.
}

// Step is a container over Executable that includes a 'Completed' field.
type Step struct {
	Exec      executable.Executable `json:"exec"`
	Completed bool                  `json:"completed"`
}

type FnResult struct {
	FQFN     string                       `json:"fqfn"`
	Key      string                       `json:"key"`
	Response *request.CoordinatedResponse `json:"response"`
	RunErr   scheduler.RunErr             `json:"runErr"`  // runErr is an error returned from a Runnable.
	ExecErr  string                       `json:"execErr"` // err is an annoying workaround that allows runGroup to propogate non-RunErrs out of its loop. Should be refactored when possible.
}

// FromJSON creates a sequence from a JSON-encoded set of steps.
func FromJSON(seqJSON []byte, req *request.CoordinatedRequest, ctx *vk.Ctx) (*Sequence, error) {
	steps := []Step{}
	if err := json.Unmarshal(seqJSON, &steps); err != nil {
		return nil, errors.Wrap(err, "failed to Unmarshal steps")
	}

	return newWithSteps(steps, req, ctx)
}

// New creates a new Sequence.
func New(execs []executable.Executable, req *request.CoordinatedRequest, ctx *vk.Ctx) (*Sequence, error) {
	steps := stepsFromExecutables(execs)

	return newWithSteps(steps, req, ctx)
}

func newWithSteps(steps []Step, req *request.CoordinatedRequest, ctx *vk.Ctx) (*Sequence, error) {
	stepsJSON, err := json.Marshal(steps)
	if err != nil {
		return nil, errors.Wrap(err, "failed to Marshal step")
	}

	req.SequenceJSON = stepsJSON

	s := &Sequence{
		steps: steps,
		req:   req,
		ctx:   ctx,
		lock:  sync.Mutex{},
	}

	if ctx != nil {
		s.log = ctx.Log
	}

	return s, nil
}

// Execute returns the "final state" of a Sequence. If the state's err is not nil, it means a runnable returned an error, and the Directive indicates the Sequence should return.
// if exec itself actually returns an error other than ErrSequenceRunErr, it means there was a problem executing the Sequence as described, and should be treated as such.
func (seq *Sequence) Execute(exec executor.Executor) error {
	seq.exec = exec

	for {
		// continue running steps until the sequence is complete.
		if err := seq.ExecuteNext(); err != nil {
			if err == executable.ErrSequenceCompleted {
				break
			}

			return err
		}
	}

	return nil
}

// ExecuteNext executes the "next" step (i.e. the first un-completed step) in the sequence.
func (seq *Sequence) ExecuteNext() error {
	step := seq.NextStep()

	if step == nil {
		return executable.ErrSequenceCompleted
	}

	return seq.executeStep(step)
}

// NextStep returns the first un-complete step, nil if the sequence is over.
func (seq *Sequence) NextStep() *Step {
	var step *Step

	for i := range seq.steps {
		// find the first incomplete step.
		if !seq.steps[i].Completed {
			step = &seq.steps[i]
			break
		}
	}

	return step
}

// executeStep uses the configured Executor to run the provided handler step. The sequence state and any errors are returned.
// State is also loaded into the object pointed to by req, and the `Completed` field is set on the Executable pointed to by step.
func (seq *Sequence) executeStep(step *Step) error {
	var reqState map[string][]byte

	desiredState, err := seq.exec.DesiredStepState(step.Exec, seq.req)
	if err != nil {
		if err == executor.ErrDesiredStateNotGenerated {
			// that's fine, do nothing.
		} else {
			return errors.Wrap(err, "failed to calculate DesiredStepState")
		}
	} else {
		// save the request's 'real' state.
		reqState = seq.req.State

		// swap in the desired state while we execute.
		seq.req.State = desiredState
	}

	// collect the results from all executed functions.
	stepResults := []FnResult{}

	if step.Exec.IsFn() {
		seq.log.Debug("running single fn", step.Exec.FQFN)

		singleResult, err := seq.ExecSingleFn(step.Exec.CallableFn)
		if err != nil {
			return err
		} else if singleResult != nil {
			// in rare cases, this can be nil and so only append if not.
			stepResults = append(stepResults, *singleResult)
		}

	} else if step.Exec.IsGroup() {
		seq.log.Debug("running group")

		// if the step is a group, run them all concurrently and collect the results.
		groupResults, err := seq.ExecGroup(step.Exec.Group)
		if err != nil {
			return err
		}

		stepResults = append(stepResults, groupResults...)
	}

	// set the completed value as the functions have been executed.
	step.Completed = true

	if reqState != nil {
		// restore the 'real' state.
		seq.req.State = reqState
	}

	seq.lock.Lock()
	defer seq.lock.Unlock()

	// determine if error handling results in a return.
	if err := seq.HandleStepErrs(stepResults, step.Exec); err != nil {
		return err
	}

	// set correct state based on the step's results.
	seq.HandleStepResults(stepResults)

	return nil
}

func (seq *Sequence) HandleStepResults(stepResults []FnResult) {
	for _, result := range stepResults {
		if result.Response == nil {
			seq.log.ErrorString("recieved nil response for", result.Key)
			continue
		}

		seq.req.State[result.Key] = result.Response.Output

		// check if the Runnable set any response headers.
		if result.Response.RespHeaders != nil {
			for k, v := range result.Response.RespHeaders {
				seq.req.RespHeaders[k] = v
			}
		}
	}
}

func (seq *Sequence) HandleStepErrs(results []FnResult, step executable.Executable) error {
	for _, result := range results {
		if result.RunErr.Code == 0 && result.RunErr.Message == "" {
			continue
		}

		if err := step.ShouldReturn(result.RunErr.Code); err != nil {
			seq.log.Error(errors.Wrapf(err, "returning after error from %s", result.FQFN))

			return result.RunErr
		} else {
			seq.log.Debug("continuing after error from", result.FQFN)
			seq.req.State[result.Key] = []byte(result.RunErr.Error())
		}
	}

	return nil
}

// handleMessage is called by the executor when in proxy mode,
// and it is responsible for collecting the fnResults from the proxied peers:
//
// sequence.Execute -> exec.do -> handleMessage (n times) -> .do returns to .Execute.
func (seq *Sequence) handleMessage(msg grav.Message) error {
	seq.lock.Lock()
	defer seq.lock.Unlock()

	result := FnResult{}
	if err := json.Unmarshal(msg.Data(), &result); err != nil {
		return errors.Wrap(err, "failed to Unmarshal FnResult")
	}

	seq.log.Debug("handleMessage recieved", msg.UUID(), "(", msg.ParentID(), ")")

	step := seq.NextStep()
	if step == nil {
		seq.log.ErrorString("handleMessage got nil NextStep")
		return executable.ErrSequenceCompleted
	} else if step.Exec.IsFn() {
		seq.log.Debug("handling result of", step.Exec.FQFN)
		step.Completed = true
	} else {
		seq.log.Warn("cannot handle message from group step")
		return nil
	}

	stepResults := []FnResult{result}

	// determine if error handling results in a return.
	if err := seq.HandleStepErrs(stepResults, step.Exec); err != nil {
		return err
	}

	// set correct state based on the step's results.
	seq.HandleStepResults(stepResults)

	// check nextstep again.
	step = seq.NextStep()
	if step == nil {
		return executable.ErrSequenceCompleted
	}

	return nil
}

// StepsJSON returns the JSON of the steps it is working on.
func (seq *Sequence) StepsJSON() ([]byte, error) {
	return json.Marshal(seq.steps)
}

func stepsFromExecutables(execs []executable.Executable) []Step {
	steps := make([]Step, len(execs))

	for i := range execs {
		steps[i] = Step{execs[i], false}
	}

	return steps
}
