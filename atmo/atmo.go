package atmo

import (
	"time"

	"github.com/pkg/errors"
	"github.com/suborbital/atmo/atmo/appsource"
	"github.com/suborbital/atmo/atmo/coordinator"
	"github.com/suborbital/atmo/atmo/options"
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

	appSource := appsource.NewBundleSource(atmoOpts.BundlePath)

	a := &Atmo{
		coordinator: coordinator.New(appSource, atmoOpts),
		server:      server,
		options:     atmoOpts,
	}

	return a
}

// Start starts the Atmo server
func (a *Atmo) Start() error {
	if err := a.coordinator.App.Start(*a.options); err != nil {
		return errors.Wrap(err, "failed to App.Start")
	}

	// wait until the AppSource claims to be ready
	for {
		if a.coordinator.App.Ready() {
			break
		} else {
			time.Sleep(time.Second)
			continue
		}
	}

	routes := a.coordinator.GenerateRouter()
	a.server.AddGroup(routes)

	if err := a.server.Start(); err != nil {
		return errors.Wrap(err, "failed to Start server")
	}

	return nil
}
