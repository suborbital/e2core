package coordinator

import (
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

type sequence struct {
	steps []directive.Executable

	connect connectFunc
	fqfn    fqfnFunc

	log *vlog.Logger
}

type sequenceState map[string][]byte

type fnResult struct {
	name   string
	result []byte
	err    error
}

func newSequence(steps []directive.Executable, connect connectFunc, fqfn fqfnFunc, log *vlog.Logger) *sequence {
	s := &sequence{
		steps:   steps,
		connect: connect,
		fqfn:    fqfn,
		log:     log,
	}

	return s
}

func (seq *sequence) exec(req *request.CoordinatedRequest) (sequenceState, error) {
	for _, step := range seq.steps {
		stateJSON, err := stateJSONForStep(req, step)
		if err != nil {
			seq.log.Error(errors.Wrap(err, "failed to stateJSONForStep"))
			return nil, err
		}

		if step.IsFn() {
			entry, err := seq.runSingleFn(step.CallableFn, stateJSON)
			if err != nil {
				return nil, err
			}

			if entry != nil {
				// reactr issue #45
				key := key(step.CallableFn)

				req.State[key] = entry
			}
		} else {
			// if the step is a group, run them all concurrently and collect the results
			entries, err := seq.runGroup(step.Group, stateJSON)
			if err != nil {
				return nil, err
			}

			for k, v := range entries {
				req.State[k] = v
			}
		}
	}

	return req.State, nil
}

func (seq sequence) runSingleFn(fn directive.CallableFn, body []byte) ([]byte, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		seq.log.Debug("fn", fn.Fn, fmt.Sprintf("executed in %d ms", duration.Milliseconds()))
	}()

	// calculate the FQFN
	fqfn, err := seq.fqfn(fn.Fn)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to FQFN for group fn %s", fn.Fn)
	}

	pod := seq.connect()
	defer pod.Disconnect()

	// compose a message containing the serialized request state, and send it via Grav
	// for the appropriate meshed Reactr to handle. It may be handled by self if appropriate.
	jobMsg := grav.NewMsg(fqfn, body)

	var jobResult []byte
	var jobErr error

	podErr := pod.Send(jobMsg).WaitUntil(grav.Timeout(30), func(msg grav.Message) error {
		switch msg.Type() {
		case rt.MsgTypeReactrResult:
			jobResult = msg.Data()
		case rt.MsgTypeReactrJobErr:
			jobErr = errors.New(string(msg.Data()))
		case rt.MsgTypeReactrNilResult:
			// do nothing
		}

		return nil
	})

	// check for errors and results, convert to something useful, and return
	// this should probably be refactored as it looks pretty goofy

	if podErr != nil {
		if podErr == grav.ErrWaitTimeout {
			jobErr = errors.Wrapf(err, "fn %s timed out", fn.Fn)
		} else {
			jobErr = errors.Wrap(podErr, "failed to receive fn result")
		}
	}

	if jobErr != nil {
		return nil, errors.Wrapf(jobErr, "group fn %s failed", fn.Fn)
	}

	if jobResult == nil {
		seq.log.Debug("fn", fn.Fn, "returned a nil result")
		return nil, nil
	}

	return jobResult, nil
}

// runGroup runs a group of functions
// this is all more complicated than it needs to be, Grav should be doing more of the work for us here
func (seq *sequence) runGroup(fns []directive.CallableFn, body []byte) (map[string][]byte, error) {
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

		key := key(fn)

		go func() {
			res, err := seq.runSingleFn(fn, body)

			result := fnResult{
				name:   key,
				result: res,
				err:    err,
			}

			resultChan <- result
		}()
	}

	entries := map[string][]byte{}
	respCount := 0
	timeoutChan := time.After(30 * time.Second)

	for respCount < len(fns) {
		select {
		case resp := <-resultChan:
			if resp.err != nil {
				return nil, errors.Wrapf(resp.err, "%s produced error", resp.name)
			}

			if resp.result != nil {
				entries[resp.name] = resp.result
			}
		case <-timeoutChan:
			return nil, errors.New("fn group timed out")
		}

		respCount++
	}

	return entries, nil
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
