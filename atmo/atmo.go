package atmo

import (
	"github.com/pkg/errors"
	"github.com/suborbital/atmo/atmo/coordinator"
	"github.com/suborbital/atmo/atmo/options"
	"github.com/suborbital/atmo/atmo/release"
	"github.com/suborbital/reactr/bundle"
	"github.com/suborbital/reactr/rwasm"
	"github.com/suborbital/vektor/vk"
	"github.com/suborbital/vektor/vlog"
)

// Atmo is an Atmo server
type Atmo struct {
	coordinator *coordinator.Coordinator
	options     options.Options

	server *vk.Server
}

type atmoInfo struct {
	AtmoVersion string `json:"atmo_version"`
}

// New creates a new Atmo instance
func New() *Atmo {
	logger := vlog.Default(
		vlog.AppMeta(atmoInfo{AtmoVersion: release.AtmoDotVersion}),
		vlog.Level(vlog.LogLevelInfo),
		vlog.EnvPrefix("ATMO"),
	)

	rwasm.UseLogger(logger)

	server := vk.New(
		vk.UseEnvPrefix("ATMO"),
		vk.UseAppName("Atmo"),
		vk.UseLogger(logger),
	)

	opts := options.NewWithModifiers(
		options.UseLogger(logger),
	)

	a := &Atmo{
		coordinator: coordinator.New(opts),
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
