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
	RunSchedules *bool        `env:"ATMO_RUN_SCHEDULES,default=true"`
	Headless     *bool        `env:"ATMO_HEADLESS,default=false"`
	Wait         *bool        `env:"ATMO_WAIT,default=false"`
	ControlPlane string       `env:"ATMO_CONTROL_PLANE"`
}

// Modifier defines options for Atmo
type Modifier func(*Options)

func NewWithModifiers(mods ...Modifier) *Options {
	opts := &Options{}

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

// ShouldRunHeadless sets wether Atmo should operate in 'headless' mode
func ShouldRunHeadless(headless bool) Modifier {
	return func(opts *Options) {
		// only set the pointer if the value is true
		if headless {
			opts.Headless = &headless
		}
	}
}

// ShouldWait sets wether Atmo should wait for a bundle to become available on disk
func ShouldWait(wait bool) Modifier {
	return func(opts *Options) {
		// only set the pointer if the value is true
		if wait {
			opts.Wait = &wait
		}
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

	o.ControlPlane = envOpts.ControlPlane

	// set RunSchedules if it was not passed as a flag
	if o.RunSchedules == nil {
		if envOpts.RunSchedules != nil {
			o.RunSchedules = envOpts.RunSchedules
		}
	}

	// set Wait if it was not passed as a flag
	// if Wait is unset but ControlPlane IS set,
	// Wait is implied to be true.
	if o.Wait == nil {
		if o.ControlPlane != "" {
			wait := true
			o.Wait = &wait
		} else if envOpts.Wait != nil {
			o.Wait = envOpts.Wait
		}
	}

	// set Headless if it was not passed as a flag
	if o.Headless == nil {
		if envOpts.Headless != nil {
			o.Headless = envOpts.Headless
		}
	}
}
