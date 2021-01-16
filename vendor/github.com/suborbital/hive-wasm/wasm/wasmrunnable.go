package wasm

import (
	"encoding/json"
	"fmt"

	"github.com/suborbital/hive-wasm/bundle"
	"github.com/suborbital/hive-wasm/request"
	"github.com/suborbital/hive/hive"
	"github.com/suborbital/vektor/vlog"

	"github.com/pkg/errors"
)

//Runner represents a wasm-based runnable
type Runner struct {
	env *wasmEnvironment
}

// UseLogger sets the logger to be used by Wasm Runnables
func UseLogger(l *vlog.Logger) {
	logger = l
}

// NewRunner returns a new *Runner
func NewRunner(filepath string) *Runner {
	ref := &bundle.WasmModuleRef{
		Filepath: filepath,
	}

	return newRunnerWithRef(ref)
}

func newRunnerWithRef(ref *bundle.WasmModuleRef) *Runner {
	environment := newEnvironment(ref)

	r := &Runner{
		env: environment,
	}

	return r
}

func newRunnerWithEnvironment(env *wasmEnvironment) *Runner {
	w := &Runner{
		env: env,
	}

	return w
}

// Run runs a Runner
func (w *Runner) Run(job hive.Job, ctx *hive.Ctx) (interface{}, error) {
	var jobBytes []byte

	// check if the job is a CoordinatedRequest, and set up the WasmInstance if so
	req, err := request.FromJSON(job.Bytes())
	if err != nil {
		logger.Debug("job is not a coordinated request:", err.Error())

		// if it's not a request, treat it as normal data
		bytes, bytesErr := interfaceToBytes(job.Data())
		if bytesErr != nil {
			return nil, errors.Wrap(bytesErr, "failed to parse job for Wasm Runnable")
		}

		jobBytes = bytes
	} else {
		// if the job is a request, the input to the Runnable is the URL
		input := fmt.Sprintf("%s %s %s", req.Method, req.URL, req.ID)
		jobBytes = []byte(input)
	}

	var output []byte
	var runErr error

	if err := w.env.useInstance(req, ctx, func(instance *wasmInstance, ident int32) {
		inPointer, writeErr := instance.writeMemory(jobBytes)
		if writeErr != nil {
			runErr = errors.Wrap(writeErr, "failed to instance.writeMemory")
			return
		}

		wasmRun := instance.wasmerInst.Exports["run_e"]
		if wasmRun == nil {
			runErr = errors.New("missing required FFI function: run_e")
			return
		}

		// ident is a random identifier for this job run that allows for "easy" FFI function calls in both directions
		if _, wasmErr := wasmRun(inPointer, len(jobBytes), ident); wasmErr != nil {
			runErr = errors.Wrap(wasmErr, "failed to wasmRun")
			return
		}

		output = <-instance.resultChan

		// deallocate the memory used for the input
		instance.deallocate(inPointer, len(jobBytes))
	}); err != nil {
		return nil, errors.Wrap(err, "failed to useInstance")
	}

	if runErr != nil {
		return nil, errors.Wrap(err, "failed to execute Wasm Runnable")
	}

	return output, nil
}

// OnChange evt ChangeEventruns when a worker starts using this Runnable
func (w *Runner) OnChange(evt hive.ChangeEvent) error {
	if evt == hive.ChangeTypeStart {
		if err := w.env.addInstance(); err != nil {
			return errors.Wrap(err, "failed to addInstance")
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
