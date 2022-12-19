//go:build tinygo.wasm

package errors

import (
	"errors"
)

// RunErr adds a status code for use in FFI calls to return_result()
type RunErr struct {
	error
	Code int
}

// New creates a new RunErr
func New(message string, code int) RunErr {
	return RunErr{errors.New(message), code}
}

// WithCode creates a new RunErr from an existing error and a status code
func WithCode(err error, code int) RunErr {
	return RunErr{err, code}
}
