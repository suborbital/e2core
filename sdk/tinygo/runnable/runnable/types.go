package runnable

// The Runnable interface is all that needs to be implemented by a Reactr runnable.
type Runnable interface {
	Run(input []byte) ([]byte, error)
}

// Deprecated: Please use "github.com/suborbital/reactr/api/tinygo/runnable/errors" instead.
type RunErr struct {
	error
	Code int
}

// Deprecated: Please use "github.com/suborbital/reactr/api/tinygo/runnable/errors" instead.
type HostErr error
