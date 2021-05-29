package appsource

import (
	"errors"

	"github.com/suborbital/atmo/atmo/options"
	"github.com/suborbital/atmo/directive"
)

var ErrRunnableNotFound = errors.New("failed to find requested Runnable")

// Meta describes the metadata for an App
type Meta struct {
	Identifier string
	AppVersion string
}

type AppSource interface {
	// Start indicates to the AppSource that it should prepare for app startup
	Start(options.Options) error
	// Runnables returns all of the available Runnables
	Runnables() []directive.Runnable
	// FindRunnable directs the AppSource to attempt to find
	// a particular Runnable and make it available at next
	// AppSource state sync via a call to Runnables().
	// ErrRunnableNotFound should be returned if the Runnable cannot be found.
	FindRunnable(string) error
	// Handlers returns the handlers for the app
	Handlers() []directive.Handler
	// Schedules returns the requested schedules for the app
	Schedules() []directive.Schedule
	// File is a source of files for the Runnables
	// TODO: refactor this into a set of capabilities / profiles
	File(string) ([]byte, error)
	// Meta returns metadata about the app
	Meta() Meta
}
