package appsource

import (
	"net/http"
	"os"

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
func (h *HeadlessBundleSource) Runnables() []directive.Runnable {
	if h.bundleSource.bundle == nil {
		return []directive.Runnable{}
	}

	return h.bundleSource.Runnables()
}

// FindRunnable returns a nil error if a Runnable with the
// provided FQFN can be made available at the next sync,
// otherwise ErrRunnableNotFound is returned.
func (h *HeadlessBundleSource) FindRunnable(fqfn, auth string) (*directive.Runnable, error) {
	if h.bundleSource.bundle == nil {
		return nil, ErrRunnableNotFound
	}

	return h.bundleSource.FindRunnable(fqfn, auth)
}

// Handlers returns the handlers for the app.
func (h *HeadlessBundleSource) Handlers() []directive.Handler {
	if h.bundleSource.bundle == nil {
		return []directive.Handler{}
	}

	handlers := []directive.Handler{}

	// for each Runnable, construct a handler that executes it
	// based on a POST request to its FQFN URL /identifier/namespace/fn/version.
	for _, runnable := range h.bundleSource.Runnables() {
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
func (h *HeadlessBundleSource) Schedules() []directive.Schedule {
	return []directive.Schedule{}
}

// Connections returns the Connections for the app.
func (h *HeadlessBundleSource) Connections() directive.Connections {
	return directive.Connections{}
}

// Authentication returns the Authentication for the app.
func (b *HeadlessBundleSource) Authentication() directive.Authentication {
	if b.bundleSource.bundle == nil {
		return directive.Authentication{}
	}

	return b.bundleSource.Authentication()
}

// Capabilities returns the Capabilities for the app.
func (b *HeadlessBundleSource) Capabilities() *rcap.CapabilityConfig {
	if b.bundleSource.bundle == nil {
		config := rcap.DefaultCapabilityConfig()
		return &config
	}

	return b.bundleSource.Capabilities()
}

// File returns a requested file.
func (h *HeadlessBundleSource) File(filename string) ([]byte, error) {
	if h.bundleSource.bundle == nil {
		return nil, os.ErrNotExist
	}

	return h.bundleSource.bundle.StaticFile(filename)
}

// Queries returns the Queries for the app.
func (b *HeadlessBundleSource) Queries() []directive.DBQuery {
	if b.bundleSource.bundle == nil {
		return []directive.DBQuery{}
	}

	return b.bundleSource.Queries()
}

func (h *HeadlessBundleSource) Meta() Meta {
	if h.bundleSource.bundle == nil {
		return Meta{}
	}

	m := Meta{
		Identifier: h.bundleSource.bundle.Directive.Identifier,
		AppVersion: h.bundleSource.bundle.Directive.AppVersion,
	}

	return m
}
