package appsource

import (
	"os"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/suborbital/atmo/atmo/options"
	"github.com/suborbital/atmo/bundle"
	"github.com/suborbital/atmo/directive"
	"github.com/suborbital/reactr/rcap"
)

// BundleSource is an AppSource backed by a bundle file
type BundleSource struct {
	path   string
	opts   options.Options
	bundle *bundle.Bundle

	lock sync.RWMutex
}

// NewBundleSource creates a new BundleSource that looks for a bundle at [path]
func NewBundleSource(path string) AppSource {
	b := &BundleSource{
		path: path,
		lock: sync.RWMutex{},
	}

	return b
}

// Start initializes the app source
func (b *BundleSource) Start(opts options.Options) error {
	b.opts = opts

	if err := b.findBundle(); err != nil {
		return errors.Wrap(err, "failed to findBundle")
	}

	return nil
}

// Runnables returns the Runnables for the app
func (b *BundleSource) Runnables() []directive.Runnable {
	// refresh the bundle since it's possible it was updated underneath you
	// this full-locks, so we call it before the RLock
	if err := b.findBundle(); err != nil {
		return []directive.Runnable{}
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	if b.bundle == nil {
		return []directive.Runnable{}
	}

	return b.bundle.Directive.Runnables
}

// FindRunnable searches for and returns the requested runnable
// otherwise ErrRunnableNotFound
func (b *BundleSource) FindRunnable(fqfn string) (*directive.Runnable, error) {
	// refresh the bundle since it's possible it was updated underneath you
	// this full-locks, so we call it before the RLock
	if err := b.findBundle(); err != nil {
		return nil, ErrRunnableNotFound
	}

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

// Handlers returns the handlers for the app
func (b *BundleSource) Handlers() []directive.Handler {
	b.lock.RLock()
	defer b.lock.RUnlock()

	if b.bundle == nil {
		return []directive.Handler{}
	}

	return b.bundle.Directive.Handlers
}

// Schedules returns the schedules for the app
func (b *BundleSource) Schedules() []directive.Schedule {
	b.lock.RLock()
	defer b.lock.RUnlock()

	if b.bundle == nil {
		return []directive.Schedule{}
	}

	return b.bundle.Directive.Schedules
}

// Connections returns the Connections for the app
func (b *BundleSource) Connections() directive.Connections {
	b.lock.RLock()
	defer b.lock.RUnlock()

	if b.bundle == nil || b.bundle.Directive.Connections == nil {
		return directive.Connections{}
	}

	return *b.bundle.Directive.Connections
}

// Authentication returns the Authentication for the app
func (b *BundleSource) Authentication() directive.Authentication {
	b.lock.RLock()
	defer b.lock.RUnlock()

	if b.bundle == nil || b.bundle.Directive.Authentication == nil {
		return directive.Authentication{}
	}

	return *b.bundle.Directive.Authentication
}

// Capabilities returns the configuration for the app's capabilities
func (b *BundleSource) Capabilities() *rcap.CapabilityConfig {
	b.lock.RLock()
	defer b.lock.RUnlock()

	if b.bundle == nil || b.bundle.Directive.Capabilities == nil {
		config := rcap.DefaultCapabilityConfig()
		return &config
	}

	return b.bundle.Directive.Capabilities
}

// File returns a requested file
func (b *BundleSource) File(filename string) ([]byte, error) {
	b.lock.RLock()
	defer b.lock.RUnlock()

	if b.bundle == nil {
		return nil, os.ErrNotExist
	}

	return b.bundle.StaticFile(filename)
}

func (b *BundleSource) Meta() Meta {
	b.lock.RLock()
	defer b.lock.RUnlock()

	if b.bundle == nil {
		return Meta{}
	}

	m := Meta{
		Identifier: b.bundle.Directive.Identifier,
		AppVersion: b.bundle.Directive.AppVersion,
	}

	return m
}

// findBundle loops forever until it finds a bundle at the configured path
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
