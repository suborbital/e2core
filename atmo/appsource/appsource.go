package appsource

import (
	"errors"

	"github.com/suborbital/atmo/atmo/options"
	"github.com/suborbital/atmo/directive"
	"github.com/suborbital/reactr/rcap"
)

var (
	ErrRunnableNotFound     = errors.New("failed to find requested Runnable")
	ErrAuthenticationFailed = errors.New("failed to authenticate")
)

// Meta describes the metadata for an App.
type Meta struct {
	Identifier string `json:"identifier"`
	AppVersion string `json:"appVersion"`
	Domain     string `json:"domain"`
}

type AppSource interface {
	// Start indicates to the AppSource that it should prepare for app startup.
	Start(opts options.Options) error

	// Runnables returns all of the available Runnables.
	Runnables(ident, version string) []directive.Runnable

	// FindRunnable attempts to find the given Runnable by its fqfn, and returns ErrRunnableNotFound if it cannot.
	FindRunnable(fqfn, authHeader string) (*directive.Runnable, error)

	// Handlers returns the handlers for the app.
	Handlers(ident, version string) []directive.Handler

	// Schedules returns the requested schedules for the app.
	Schedules(ident, version string) []directive.Schedule

	// Connections returns the connections needed for the app.
	Connections(ident, version string) directive.Connections

	// Authentication provides any auth headers or metadata for the app.
	Authentication(ident, version string) directive.Authentication

	// Capabilities provides the application's configured capabilities.
	Capabilities(ident, namespace, version string) *rcap.CapabilityConfig

	// File is a source of files for the Runnables
	// TODO: refactor this into a set of capabilities / profiles.
	File(identifier, version, path string) ([]byte, error)

	// Queries returns the database queries that should be made available.
	Queries(ident, version string) []directive.DBQuery

	// Applications returns a slice of Meta, metadata about the apps in that app source.
	Applications() []Meta
}
