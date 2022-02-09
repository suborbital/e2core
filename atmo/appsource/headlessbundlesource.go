package appsource

import (
	"net/http"

	"github.com/pkg/errors"

	"github.com/suborbital/atmo/atmo/options"
	"github.com/suborbital/atmo/directive"
	"github.com/suborbital/atmo/directive/executable"
	"github.com/suborbital/atmo/fqfn"
	"github.com/suborbital/reactr/rcap"
)

// HeadlessBundleSource is an AppSource backed by a bundle file (but headless).
type HeadlessBundleSource struct {
	path         string
	opts         options.Options
	bundleSource *BundleSource
}

// NewHeadlessBundleSource creates a new HeadlessBundleSource that looks for a bundle at [path].
func NewHeadlessBundleSource(path string) AppSource {
	b := &HeadlessBundleSource{
		path: path,
		// re-use BundleSource's Directive-finding logic etc.
		bundleSource: NewBundleSource(path).(*BundleSource),
	}

	return b
}

// Start initializes the app source.
func (h *HeadlessBundleSource) Start(opts options.Options) error {
	h.opts = opts

	if err := h.bundleSource.Start(opts); err != nil {
		return errors.Wrap(err, "failed to bundleSource.Start")
	}

	return nil
}

// Runnables returns the Runnables for the app.
func (h *HeadlessBundleSource) Runnables(identifier, version string) []directive.Runnable {
	return h.bundleSource.Runnables(identifier, version)
}

// FindRunnable returns a nil error if a Runnable with the
// provided FQFN can be made available at the next sync,
// otherwise ErrRunnableNotFound is returned.
func (h *HeadlessBundleSource) FindRunnable(identifier, version, fqfn, auth string) (*directive.Runnable, error) {
	return h.bundleSource.FindRunnable(identifier, version, fqfn, auth)
}

// Handlers returns the handlers for the app.
func (h *HeadlessBundleSource) Handlers(identifier, version string) []directive.Handler {
	if h.bundleSource.bundle == nil {
		return []directive.Handler{}
	}

	handlers := make([]directive.Handler, 0)

	// for each Runnable, construct a handler that executes it
	// based on a POST request to its FQFN URL /identifier/namespace/fn/version.
	for _, runnable := range h.bundleSource.Runnables(identifier, version) {
		handler := directive.Handler{
			Input: directive.Input{
				Type:     directive.InputTypeRequest,
				Method:   http.MethodPost,
				Resource: fqfn.Parse(runnable.FQFN).HeadlessURLPath(),
			},
			Steps: []executable.Executable{
				{
					CallableFn: executable.CallableFn{
						Fn:   runnable.Name,
						With: map[string]string{},
						FQFN: runnable.FQFN,
					},
				},
			},
		}

		handlers = append(handlers, handler)
	}

	return handlers
}

// Schedules returns the schedules for the app.
func (h *HeadlessBundleSource) Schedules(_, _ string) []directive.Schedule {
	return nil
}

// Connections returns the Connections for the app.
func (h *HeadlessBundleSource) Connections(_, _ string) directive.Connections {
	return directive.Connections{}
}

// Authentication returns the Authentication for the app.
func (h *HeadlessBundleSource) Authentication(identifier, version string) directive.Authentication {
	return h.bundleSource.Authentication(identifier, version)
}

// Capabilities returns the Capabilities for the app.
func (h *HeadlessBundleSource) Capabilities(identifier, version string) *rcap.CapabilityConfig {
	return h.bundleSource.Capabilities(identifier, version)
}

// File returns a requested file.
func (h *HeadlessBundleSource) File(identifier, version, filename string) ([]byte, error) {
	return h.bundleSource.File(identifier, version, filename)
}

// Queries returns the Queries for the app.
func (h *HeadlessBundleSource) Queries(identifier, version string) []directive.DBQuery {
	return h.bundleSource.Queries(identifier, version)
}

// Applications returns the slice of Meta for the app.
func (h *HeadlessBundleSource) Applications() []Meta {
	return h.bundleSource.Applications()
}
