package atmo

import (
	"github.com/pkg/errors"
	"github.com/suborbital/atmo/atmo/coordinator"
	"github.com/suborbital/atmo/bundle"
	"github.com/suborbital/reactr/rwasm"
	"github.com/suborbital/vektor/vk"
	"github.com/suborbital/vektor/vlog"
)

// Atmo is an Atmo server
type Atmo struct {
	coordinator *coordinator.Coordinator
	options     Options

	server *vk.Server
}

// New creates a new Atmo instance
func New(mods ...OptionModifier) *Atmo {
	logger := vlog.Default(
		vlog.Level(vlog.LogLevelDebug),
	)

	rwasm.UseLogger(logger)

	server := vk.New(
		vk.UseEnvPrefix("ATMO"),
		vk.UseAppName("Atmo"),
		vk.UseLogger(logger),
	)

	a := &Atmo{
		coordinator: coordinator.New(logger),
		server:      server,
	}

	return a
}

// Start starts the Atmo server
func (a *Atmo) Start(bundlePath string) error {
	bundle, err := bundle.Read(bundlePath)
	if err != nil {
		return errors.Wrap(err, "failed to ReadBundle")
	}

	routes := a.coordinator.UseBundle(bundle)
	a.server.AddGroup(routes)

	if err := a.server.Start(); err != nil {
		return errors.Wrap(err, "failed to Start server")
	}

	return nil
}
