package sequence

import (
	"encoding/json"
	"sync"

	"github.com/pkg/errors"

	"github.com/suborbital/e2core/foundation/scheduler"
	"github.com/suborbital/systemspec/request"
	"github.com/suborbital/systemspec/tenant"
)

type Sequence struct {
	steps []Step

	req *request.CoordinatedRequest

	lock sync.Mutex // need to ensure writes to req.State are kept serial.
}

// Step is a container over WorkflowStep that includes a 'Completed' field.
type Step struct {
	tenant.WorkflowStep `json:"inline"`
	Completed           bool `json:"completed"`
}

type ExecResult struct {
	FQMN     string                       `json:"fqmn"`
	Response *request.CoordinatedResponse `json:"response"`
	RunErr   scheduler.RunErr             `json:"runErr"`  // runErr is an error returned from a Runnable.
	ExecErr  string                       `json:"execErr"` // err is an annoying workaround that allows runGroup to propogate non-RunErrs out of its loop. Should be refactored when possible.
}

// FromJSON creates a sequence from a JSON-encoded set of steps.
func FromJSON(seqJSON []byte, req *request.CoordinatedRequest) (*Sequence, error) {
	steps := []Step{}
	if err := json.Unmarshal(seqJSON, &steps); err != nil {
		return nil, errors.Wrap(err, "failed to Unmarshal steps")
	}

	return newWithSteps(steps, req)
}

// New creates a new Sequence.
func New(workflowSteps []tenant.WorkflowStep, req *request.CoordinatedRequest) (*Sequence, error) {
	steps := createUncompletedSteps(workflowSteps)

	return newWithSteps(steps, req)
}

func newWithSteps(steps []Step, req *request.CoordinatedRequest) (*Sequence, error) {
	stepsJSON, err := json.Marshal(steps)
	if err != nil {
		return nil, errors.Wrap(err, "failed to Marshal step")
	}

	req.SequenceJSON = stepsJSON

	s := &Sequence{
		steps: steps,
		req:   req,
		lock:  sync.Mutex{},
	}

	return s, nil
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

// Request returns the request for this sequence
func (seq *Sequence) Request() *request.CoordinatedRequest {
	return seq.req
}

// ParentID returns the parent ID for this sequence
func (seq *Sequence) ParentID() string {
	return seq.req.ID
}

func (seq *Sequence) HandleStepResults(results []ExecResult) error {
	seq.lock.Lock()
	defer seq.lock.Unlock()

	for _, result := range results {
		if result.RunErr.Code != 0 || result.RunErr.Message != "" {
			return result.RunErr
		} else if result.ExecErr != "" {
			return errors.New(result.ExecErr)
		}

		seq.req.State[result.FQMN] = result.Response.Output

		// check if the Runnable set any response headers.
		if result.Response.RespHeaders != nil {
			for k, v := range result.Response.RespHeaders {
				seq.req.RespHeaders[k] = v
			}
		}
	}

	step := seq.NextStep()
	step.Completed = true

	return nil
}

// StepsJSON returns the JSON of the steps it is working on.
func (seq *Sequence) StepsJSON() ([]byte, error) {
	return json.Marshal(seq.steps)
}

func createUncompletedSteps(execs []tenant.WorkflowStep) []Step {
	steps := make([]Step, len(execs))

	for i := range execs {
		steps[i] = Step{execs[i], false}
	}

	return steps
}
