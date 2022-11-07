package runtime

type innerFunc func(args ...interface{}) (interface{}, error)

// HostFn describes a host function callable from within a Runnable module
type HostFn struct {
	Name     string
	ArgCount int
	Returns  bool
	HostFn   innerFunc
}

// NewHostFn creates a new host function
func NewHostFn(name string, argCount int, returns bool, fn innerFunc) HostFn {
	h := HostFn{
		Name:     name,
		ArgCount: argCount,
		Returns:  returns,
		HostFn:   fn,
	}

	return h
}
