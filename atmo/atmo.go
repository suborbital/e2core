package atmo

import (
	"time"

	"github.com/pkg/errors"
	"github.com/suborbital/atmo/atmo/coordinator"
	"github.com/suborbital/atmo/atmo/options"
	"github.com/suborbital/reactr/bundle"
	"github.com/suborbital/reactr/rwasm"
	"github.com/suborbital/vektor/vk"
)

// Atmo is an Atmo server
type Atmo struct {
	coordinator *coordinator.Coordinator
	server      *vk.Server

	options *options.Options
}

// New creates a new Atmo instance
func New(opts ...options.Modifier) *Atmo {
	atmoOpts := options.NewWithModifiers(opts...)

	rwasm.UseLogger(atmoOpts.Logger)

	server := vk.New(
		vk.UseEnvPrefix("ATMO"),
		vk.UseAppName("Atmo"),
		vk.UseLogger(atmoOpts.Logger),
	)

	a := &Atmo{
		coordinator: coordinator.New(atmoOpts),
		server:      server,
		options:     atmoOpts,
	}

	return a
}

// Start starts the Atmo server
func (a *Atmo) Start(bundlePath string) error {
	var bdl *bundle.Bundle

	for {
		b, err := bundle.Read(bundlePath)
		if err != nil {
			// if there was a problem, but the 'wait' option is set,
			// then try again after a second
			if a.options.Wait {
				a.options.Logger.Warn("failed to Read bundle, will try again:", err.Error())
				time.Sleep(time.Second)
				continue
			}

			return errors.Wrap(err, "failed to ReadBundle")
		}

		a.options.Logger.Info("found bundle at", bundlePath)

		bdl = b
		break
	}

	routes := a.coordinator.UseBundle(bdl)
	a.server.AddGroup(routes)

	if err := a.server.Start(); err != nil {
		return errors.Wrap(err, "failed to Start server")
	}

	return nil
}
