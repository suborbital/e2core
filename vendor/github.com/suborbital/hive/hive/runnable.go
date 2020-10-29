package hive

// DoFunc describes a function to schedule work
type DoFunc func(Job) *Result

// Runnable describes something that is runnable
type Runnable interface {
	// Run is the entrypoint for jobs handled by a Runnable
	Run(Job, DoFunc) (interface{}, error)

	// OnStart is called by the scheduler when a worker is started that will use the Runnable
	// OnStart will be called once for each worker in a pool
	OnStart() error
}
