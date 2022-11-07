package engine

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/pkg/errors"

	"github.com/suborbital/appspec/fqmn"
	"github.com/suborbital/appspec/request"
	"github.com/suborbital/appspec/tenant"
	"github.com/suborbital/appspec/tenant/executable"
	"github.com/suborbital/e2core/scheduler"
	"github.com/suborbital/e2core/server/coordinator/sequence"

	"github.com/suborbital/e2core/sat/api"
	"github.com/suborbital/e2core/sat/engine/runtime"
)

var (
	ErrDesiredStateNotGenerated = errors.New("desired state was not generated")
)

// wasmRunner represents a wasm-based runnable
type wasmRunner struct {
	env *runtime.WasmEnvironment
}

// newRunnerFromFile returns a new *wasmRunner
func newRunnerFromFile(filepath string, api api.HostAPI) (*wasmRunner, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to Open")
	}

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, errors.Wrap(err, "failed to ReadAll")
	}

	ref := tenant.NewWasmModuleRef("", "", data)

	runner := newRunnerFromRef(ref, api)

	return runner, nil
}

// newRunnerFromRef creates a wasmRunner from a moduleRef
func newRunnerFromRef(ref *tenant.WasmModuleRef, api api.HostAPI) *wasmRunner {
	builder := runtimeBuilder(ref, api)

	environment := runtime.NewEnvironment(builder)

	r := &wasmRunner{
		env: environment,
	}

	return r
}

// Run runs a wasmRunner
func (w *wasmRunner) Run(job scheduler.Job, ctx *scheduler.Ctx) (interface{}, error) {
	var jobBytes []byte
	var req *request.CoordinatedRequest

	// check if the job is a CoordinatedRequest (pointer or bytes), and set up the WasmInstance if so
	if jobReq, ok := job.Data().(*request.CoordinatedRequest); ok {
		req = jobReq

	} else if jobReq, err := request.FromJSON(job.Bytes()); err == nil {
		req = jobReq

	} else {
		// if it's not a request, treat it as normal data
		bytes, bytesErr := interfaceToBytes(job.Data())
		if bytesErr != nil {
			return nil, errors.Wrap(bytesErr, "failed to parse job for Wasm Runnable")
		}

		jobBytes = bytes
	}

	if req != nil {
		if req.SequenceJSON != nil && len(req.SequenceJSON) > 0 {
			seq, err := sequence.FromJSON(req.SequenceJSON, req, nil)
			if err != nil {
				return nil, errors.Wrap(err, "failed to sequence.FromJSON")
			}

			// figure out where we are in the sequence
			step := seq.NextStep()
			if step == nil {
				return nil, errors.New("got nil NextStep")
			}

			state, err := desiredStepState(step.Exec, req)
			if err != nil {
				if err != ErrDesiredStateNotGenerated {
					return nil, errors.Wrap(err, "failed to desiredStepState")
				}
			} else {
				req.State = state
			}

		}

		// save the coordinated request into the
		// job context for use by the API package
		ctx.Context = api.ContextWithRequest(ctx.Context, req)

		jobBytes = req.Body
	}

	var output []byte
	var runErr error
	var callErr error

	if err := w.env.UseInstance(ctx, func(instance *runtime.WasmInstance, ident int32) {
		inPointer, writeErr := instance.WriteMemory(jobBytes)
		if writeErr != nil {
			runErr = errors.Wrap(writeErr, "failed to instance.writeMemory")
			return
		}

		// execute the Runnable's Run function, passing the input data and ident
		// set runErr but don't return because the ExecutionResult error should also be grabbed
		_, callErr = instance.Call("run_e", inPointer, int32(len(jobBytes)), ident)

		// get the results from the instance
		output, runErr = instance.ExecutionResult()

		// deallocate the memory used for the input
		instance.Deallocate(inPointer, len(jobBytes))
	}); err != nil {
		return nil, errors.Wrap(err, "failed to useInstance")
	}

	if runErr != nil {
		// we do not wrap the error here as we want to
		// propogate its exact type to the caller (specifically scheduler.RunErr)
		return nil, runErr
	}

	if callErr != nil {
		// if the runnable didn't return an explicit runErr, still check to see if there was an
		// error executing the module in the first place. It's posslble for both to be non-nil
		// in which case returning the runErr takes precedence, which is why it's checked first.
		return nil, errors.Wrap(callErr, "wasm execution error")
	}

	if req != nil {
		resp := &request.CoordinatedResponse{
			Output:      output,
			RespHeaders: req.RespHeaders,
		}

		return resp, nil
	}

	return output, nil
}

// OnChange runs when a worker starts using this Runnable
func (w *wasmRunner) OnChange(evt scheduler.ChangeEvent) error {
	switch evt {
	case scheduler.ChangeTypeStart:
		if err := w.env.AddInstance(); err != nil {
			return errors.Wrap(err, "failed to addInstance")
		}
	case scheduler.ChangeTypeStop:
		if err := w.env.RemoveInstance(); err != nil {
			return errors.Wrap(err, "failed to removeInstance")
		}
	}

	return nil
}

func interfaceToBytes(data interface{}) ([]byte, error) {
	// if data is []byte or string, return it as-is
	if b, ok := data.([]byte); ok {
		return b, nil
	} else if s, ok := data.(string); ok {
		return []byte(s), nil
	}

	// otherwise, assume it's a struct of some kind,
	// so JSON marshal it and return it
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return nil, errors.Wrap(err, "failed to Marshal job data")
	}

	return dataJSON, nil
}

// desiredStepState calculates the state as it should be for a particular step's 'with' clause.
func desiredStepState(step executable.Executable, req *request.CoordinatedRequest) (map[string][]byte, error) {
	if len(step.With) == 0 {
		return nil, ErrDesiredStateNotGenerated
	}

	desiredState := map[string][]byte{}
	aliased := map[string]bool{}

	// first go through the 'with' clause and load all of the appropriate aliased values.
	for alias, key := range step.With {
		val, exists := req.State[key]
		if !exists {
			// if the literal key is not in state,
			// iterate through all the state keys and
			// parse them as FQMNs, and match with any
			// that have matching names.

			found := false

			for stateKey := range req.State {
				stateFQMN, err := fqmn.Parse(stateKey)
				if err != nil {
					// if the state key isn't an FQMN, that's fine, move along
					continue
				}

				if stateFQMN.Name == key {
					found = true

					val = req.State[stateKey]

					desiredState[alias] = val
					aliased[stateKey] = true

					break
				}
			}

			if !found {
				return nil, fmt.Errorf("failed to build desired state, %s does not exists in handler state", key)
			}
		} else {
			desiredState[alias] = val
			aliased[key] = true
		}

	}

	// next, go through the rest of the original state and load the non-aliased values.
	for key, val := range req.State {
		_, skip := aliased[key]
		if !skip {
			desiredState[key] = val
		}
	}

	return desiredState, nil
}
