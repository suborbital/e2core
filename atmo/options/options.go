package options

import (
	"context"

	"github.com/pkg/errors"
	"github.com/sethvargo/go-envconfig"
	"github.com/suborbital/vektor/vlog"
)

const atmoEnvPrefix = "ATMO"

// Options defines options for Atmo
type Options struct {
	Logger       *vlog.Logger `env:"-"`
	BundlePath   string       `env:"ATMO_BUNDLE_PATH"`
	RunSchedules string       `env:"ATMO_RUN_SCHEDULES"`
	Wait         bool         `env:"ATMO_WAIT"`
}

// Modifier defines options for Atmo
type Modifier func(*Options)

func NewWithModifiers(mods ...Modifier) *Options {
	opts := defaultOptions()

	for _, mod := range mods {
		mod(opts)
	}

	opts.finalize(atmoEnvPrefix)

	return opts
}

// UseLogger sets the logger to be used
func UseLogger(logger *vlog.Logger) Modifier {
	return func(opts *Options) {
		opts.Logger = logger
	}
}

// UseBundlePath sets the bundle path to be used
func UseBundlePath(path string) Modifier {
	return func(opts *Options) {
		opts.BundlePath = path
	}
}

// ShouldWait sets wether Atmo should wait for a bundle to become available on disk
func ShouldWait(wait bool) Modifier {
	return func(opts *Options) {
		opts.Wait = wait
	}
}

// finalize "locks in" the options by overriding any existing options with the version from the environment, and setting the default logger if needed
func (o *Options) finalize(prefix string) {
	if o.Logger == nil {
		o.Logger = vlog.Default(vlog.EnvPrefix(prefix))
	}

	envOpts := Options{}
	if err := envconfig.Process(context.Background(), &envOpts); err != nil {
		o.Logger.Error(errors.Wrap(err, "failed to Process environment config"))
		return
	}

	if envOpts.RunSchedules != "" {
		o.RunSchedules = envOpts.RunSchedules
	}
}

func defaultOptions() *Options {
	o := &Options{
		RunSchedules: "true",
	}

	return o
}
