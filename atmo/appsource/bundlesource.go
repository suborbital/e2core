package appsource

import (
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/suborbital/atmo/atmo/options"
	"github.com/suborbital/atmo/bundle"
	"github.com/suborbital/atmo/directive"
)

// BundleSource is an AppSource backed by a bundle file
type BundleSource struct {
	path   string
	opts   options.Options
	bundle *bundle.Bundle
}

// NewBundleSource creates a new BundleSource that looks for a bundle at [path]
func NewBundleSource(path string) AppSource {
	b := &BundleSource{
		path: path,
	}

	return b
}

// Start initializes the app source
func (b *BundleSource) Start(opts options.Options) error {
	b.opts = opts

	if err := b.findBundle(); err != nil {
		return errors.Wrap(err, "failed to findBundle")
	}

	if err := b.bundle.Directive.Validate(); err != nil {
		return errors.Wrap(err, "failed to Validate Directive")
	}

	return nil
}

// Runnables returns the Runnables for the app
func (b *BundleSource) Runnables() []directive.Runnable {
	if b.bundle == nil {
		return []directive.Runnable{}
	}

	return b.bundle.Directive.Runnables
}

// Handlers returns the handlers for the app
func (b *BundleSource) Handlers() []directive.Handler {
	if b.bundle == nil {
		return []directive.Handler{}
	}

	return b.bundle.Directive.Handlers
}

// Schedules returns the schedules for the app
func (b *BundleSource) Schedules() []directive.Schedule {
	if b.bundle == nil {
		return []directive.Schedule{}
	}

	return b.bundle.Directive.Schedules
}

// File returns a requested file
func (b *BundleSource) File(filename string) ([]byte, error) {
	if b.bundle == nil {
		return nil, os.ErrNotExist
	}

	return b.bundle.StaticFile(filename)
}

func (b *BundleSource) Meta() Meta {
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

		b.opts.Logger.Info("found bundle at", b.path)

		b.bundle = bdl
		break
	}

	return nil
}
