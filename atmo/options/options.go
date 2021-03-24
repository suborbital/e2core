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
	Logger       *vlog.Logger
	RunSchedules bool `env:"ATMO_RUN_SCHEDULES"`
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

// finalize "locks in" the options by overriding any existing options with the version from the environment, and setting the default logger if needed
func (o *Options) finalize(prefix string) {
	if o.Logger == nil {
		o.Logger = vlog.Default(vlog.EnvPrefix(prefix))
	}

	envOpts := Options{}
	if err := envconfig.ProcessWith(context.Background(), &envOpts, envconfig.OsLookuper()); err != nil {
		o.Logger.Error(errors.Wrap(err, "failed to ProcessWith environment config"))
		return
	}
}

func defaultOptions() *Options {
	o := &Options{
		RunSchedules: true,
	}

	return o
}
