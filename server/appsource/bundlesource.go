package appsource

import (
	"os"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/suborbital/reactr/rcap"
	"github.com/suborbital/velocity/bundle"
	"github.com/suborbital/velocity/directive"
	"github.com/suborbital/velocity/server/options"
)

// BundleSource is an AppSource backed by a bundle file.
type BundleSource struct {
	path   string
	opts   options.Options
	bundle *bundle.Bundle

	lock sync.RWMutex
}

// NewBundleSource creates a new BundleSource that looks for a bundle at [path].
func NewBundleSource(path string) AppSource {
	b := &BundleSource{
		path: path,
		lock: sync.RWMutex{},
	}

	return b
}

// Start initializes the app source.
func (b *BundleSource) Start(opts options.Options) error {
	b.opts = opts

	if err := b.findBundle(); err != nil {
		return errors.Wrap(err, "failed to findBundle")
	}

	return nil
}

// Runnables returns the Runnables for the app.
func (b *BundleSource) Runnables(identifier, version string) []directive.Runnable {
	if !b.checkIdentifierVersion(identifier, version) {
		return nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	if b.bundle == nil {
		return nil
	}

	return b.bundle.Directive.Runnables
}

// FindRunnable searches for and returns the requested runnable
// otherwise ErrRunnableNotFound.
func (b *BundleSource) FindRunnable(fqfn, _ string) (*directive.Runnable, error) {
	b.lock.RLock()
	defer b.lock.RUnlock()

	if b.bundle == nil {
		return nil, ErrRunnableNotFound
	}

	for i, r := range b.bundle.Directive.Runnables {
		if r.FQFN == fqfn {
			return &b.bundle.Directive.Runnables[i], nil
		}
	}

	return nil, ErrRunnableNotFound
}

// Handlers returns the handlers for the app.
func (b *BundleSource) Handlers(identifier, version string) []directive.Handler {
	if !b.checkIdentifierVersion(identifier, version) {
		return nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	if b.bundle == nil {
		return []directive.Handler{}
	}

	return b.bundle.Directive.Handlers
}

// Schedules returns the schedules for the app.
func (b *BundleSource) Schedules(identifier, version string) []directive.Schedule {
	if !b.checkIdentifierVersion(identifier, version) {
		return nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	if b.bundle == nil {
		return []directive.Schedule{}
	}

	return b.bundle.Directive.Schedules
}

// Connections returns the Connections for the app.
func (b *BundleSource) Connections(identifier, version string) directive.Connections {
	if !b.checkIdentifierVersion(identifier, version) {
		return directive.Connections{}
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	if b.bundle == nil || b.bundle.Directive.Connections == nil {
		return directive.Connections{}
	}

	return *b.bundle.Directive.Connections
}

// Authentication returns the Authentication for the app.
func (b *BundleSource) Authentication(identifier, version string) directive.Authentication {
	if !b.checkIdentifierVersion(identifier, version) {
		return directive.Authentication{}
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	if b.bundle == nil || b.bundle.Directive.Authentication == nil {
		return directive.Authentication{}
	}

	return *b.bundle.Directive.Authentication
}

// Capabilities returns the configuration for the app's capabilities.

func (b *BundleSource) Capabilities(identifier, namespace, version string) *rcap.CapabilityConfig {
	defaultConfig := rcap.DefaultCapabilityConfig()

	if !b.checkIdentifierVersion(identifier, version) {
		return &defaultConfig
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	if b.bundle == nil || b.bundle.Directive.Capabilities == nil {
		return &defaultConfig
	}

	return b.bundle.Directive.Capabilities
}

// File returns a requested file.
func (b *BundleSource) File(identifier, version, filename string) ([]byte, error) {
	if !b.checkIdentifierVersion(identifier, version) {
		return nil, os.ErrNotExist
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	if b.bundle == nil {
		return nil, os.ErrNotExist
	}

	return b.bundle.StaticFile(filename)
}

// Queries returns the Queries available to the app.
func (b *BundleSource) Queries(identifier, version string) []directive.DBQuery {
	if !b.checkIdentifierVersion(identifier, version) {
		return nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	if b.bundle == nil || b.bundle.Directive.Queries == nil {
		return nil
	}

	return b.bundle.Directive.Queries
}

func (b *BundleSource) Applications() []Meta {
	b.lock.RLock()
	defer b.lock.RUnlock()

	if b.bundle == nil {
		return nil
	}

	return []Meta{
		{
			Identifier: b.bundle.Directive.Identifier,
			AppVersion: b.bundle.Directive.AppVersion,
		},
	}
}

// findBundle loops forever until it finds a bundle at the configured path.
func (b *BundleSource) findBundle() error {
	for {
		bdl, err := bundle.Read(b.path)
		if err != nil {
			if !*b.opts.Wait {
				return errors.Wrap(err, "failed to Read bundle")
			}

			b.opts.Logger.Warn("failed to Read bundle, will try again:", err.Error())
			time.Sleep(time.Second)

			continue
		}

		b.opts.Logger.Info("loaded bundle from", b.path)

		b.lock.Lock()
		defer b.lock.Unlock()

		b.bundle = bdl

		if err := b.bundle.Directive.Validate(); err != nil {
			return errors.Wrap(err, "failed to Validate Directive")
		}

		break
	}

	return nil
}

// checkIdentifierVersion checks whether the passed in identifier and version are for the current app running in the
// bundle or not. Returns true only if both match.
func (b *BundleSource) checkIdentifierVersion(identifier, version string) bool {
	return b.bundle.Directive.Identifier == identifier &&
		b.bundle.Directive.AppVersion == version
}
