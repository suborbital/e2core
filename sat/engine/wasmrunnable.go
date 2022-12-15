package engine

import (
	"encoding/json"
	"io"
	"os"

	"github.com/pkg/errors"

	"github.com/suborbital/e2core/e2core/sequence"
	"github.com/suborbital/e2core/foundation/scheduler"
	"github.com/suborbital/e2core/sat/engine/runtime"
	"github.com/suborbital/e2core/sat/engine/runtime/api"
	"github.com/suborbital/e2core/sat/engine/runtime/wasmtime"
	"github.com/suborbital/systemspec/request"
	"github.com/suborbital/systemspec/tenant"
)

var (
	ErrDesiredStateNotGenerated = errors.New("desired state was not generated")
)

// wasmRunner represents a Wasm module
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
	builder := wasmtime.NewBuilder(ref, api)

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
			return nil, errors.Wrap(bytesErr, "failed to parse job for Wasm module")
		}

		jobBytes = bytes
	}

	if req != nil {
		if req.SequenceJSON != nil && len(req.SequenceJSON) > 0 {
			seq, err := sequence.FromJSON(req.SequenceJSON, req)
			if err != nil {
				return nil, errors.Wrap(err, "failed to sequence.FromJSON")
			}

			// figure out where we are in the sequence
			step := seq.NextStep()
			if step == nil {
				return nil, errors.New("got nil NextStep")
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

		// execute the module's Run function, passing the input data and ident
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
		// if the module didn't return an explicit runErr, still check to see if there was an
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

// OnChange runs when a worker starts using this module
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
