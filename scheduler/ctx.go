package scheduler

import (
	"context"

	"github.com/pkg/errors"
)

// Ctx is a Job context
type Ctx struct {
	Context   context.Context
	ffiResult *FFIResult
	ffiVars   []FFIVariable
	doFunc    coreDoFunc
}

// FFIResult is the results of an FFI host function call
type FFIResult struct {
	Result []byte
	Err    error
}

// FFIVariable is a variable that a Wasm Runnable can store host-side to be used in a host call
// such as a DB query. They are both ordered AND named, stored on the instance itself.
type FFIVariable struct {
	Name  string
	Value interface{}
}

func newCtx(doFunc coreDoFunc) *Ctx {
	c := &Ctx{
		Context: context.Background(),
		ffiVars: []FFIVariable{},
		doFunc:  doFunc,
	}

	return c
}

// Do runs a new job
func (c *Ctx) Do(job Job) *Result {
	return c.doFunc(&job)
}

func (c *Ctx) SetFFIResult(result []byte, err error) (*FFIResult, error) {
	if c.ffiResult != nil {
		return nil, errors.New("context's ffiResult is already set")
	}

	r := &FFIResult{
		Result: result,
		Err:    err,
	}

	c.ffiResult = r

	return r, nil
}

func (c *Ctx) UseFFIResult() (*FFIResult, error) {
	if c.ffiResult == nil {
		return nil, errors.New("context's ffiResult is not set")
	}

	defer func() {
		c.ffiResult = nil
	}()

	return c.ffiResult, nil
}

// HasFFIResult returns true if the Ctx has a current FFI result
func (c *Ctx) HasFFIResult() bool {
	return c.ffiResult != nil
}

// AddVar adds an FFI variable to the context
func (c *Ctx) AddVar(name, value string) {
	if c.ffiVars == nil {
		c.ffiVars = []FFIVariable{
			{name, value},
		}
		return
	}

	c.ffiVars = append(c.ffiVars, FFIVariable{name, value})
}

// UseVars returns the list of variables that the Wasm module has set on this Ctx. They are ordered and named.
// Since the variables can only be used by one host call, they are cleared after being returned.
func (c *Ctx) UseVars() ([]FFIVariable, error) {
	if c.ffiVars == nil {
		return nil, errors.New("context's ffiVars is not set")
	}

	defer func() {
		c.ffiVars = nil
	}()

	return c.ffiVars, nil
}

// FFISize returns the "size" of the result (positive int32 for a successful result, negative for error result)
func (r *FFIResult) FFISize() int32 {
	if r.Err != nil {
		return int32(len([]byte(r.Err.Error())) * -1)
	}

	return int32(len(r.Result))
}
