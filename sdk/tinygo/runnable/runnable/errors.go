package runnable

import "errors"

// Deprecated: Please use "github.com/suborbital/reactr/api/tinygo/runnable/errors" instead.
func NewError(code int, message string) RunErr {
	return RunErr{errors.New(message), code}
}

// Deprecated: Please use "github.com/suborbital/reactr/api/tinygo/runnable/errors" instead.
func NewHostError(message string) HostErr {
	return errors.New(message).(HostErr)
}
