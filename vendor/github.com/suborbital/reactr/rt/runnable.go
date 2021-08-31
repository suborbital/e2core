package rt

// DoFunc describes a function to schedule work
type DoFunc func(Job) *Result

// ChangeEvent represents a change relevant to a worker
type ChangeEvent int

// ChangeTypeStart and others represent types of changes
const (
	ChangeTypeStart ChangeEvent = iota
	ChangeTypeStop  ChangeEvent = iota
)

// Runnable describes something that is runnable
type Runnable interface {
	// Run is the entrypoint for jobs handled by a Runnable
	Run(Job, *Ctx) (interface{}, error)

	// OnChange is called when the worker using the Runnable instance is going to change.
	// OnChange will be called for things like startup and shutdown.
	OnChange(ChangeEvent) error
}
