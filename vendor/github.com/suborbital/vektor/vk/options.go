package vk

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/sethvargo/go-envconfig"
	"github.com/suborbital/vektor/vlog"
)

// Options are the available options for Server
type Options struct {
	AppName   string `env:"_APP_NAME"`
	Domain    string `env:"_DOMAIN"`
	HTTPPort  int    `env:"_HTTP_PORT"`
	EnvPrefix string `env:"-"`
	Logger    *vlog.Logger
}

func newOptsWithModifiers(mods ...OptionsModifier) *Options {
	options := &Options{}
	// loop through the provided options and apply the
	// modifier function to the options object
	for _, mod := range mods {
		mod(options)
	}

	envPrefix := defaultEnvPrefix
	if options.EnvPrefix != "" {
		envPrefix = options.EnvPrefix
	}

	options.finalize(envPrefix)

	return options
}

// ShouldUseHTTP returns true and a port string if the option is enabled
func (o *Options) ShouldUseHTTP() (bool, string) {
	if o.HTTPPort != 0 {
		return true, fmt.Sprintf(":%d", o.HTTPPort)
	}

	return false, ""
}

// finalize "locks in" the options by overriding any existing options with the version from the environment, and setting the default logger if needed
func (o *Options) finalize(prefix string) {
	if o.Logger == nil {
		o.Logger = vlog.Default(vlog.EnvPrefix(prefix))
	}

	envOpts := Options{}
	if err := envconfig.ProcessWith(context.Background(), &envOpts, envconfig.PrefixLookuper(prefix, envconfig.OsLookuper())); err != nil {
		o.Logger.Error(errors.Wrap(err, "[vk] failed to ProcessWith environment config"))
		return
	}

	o.replaceFieldsIfNeeded(&envOpts)
}

func (o *Options) replaceFieldsIfNeeded(replacement *Options) {
	if replacement.AppName != "" {
		o.AppName = replacement.AppName
	}

	if replacement.Domain != "" {
		o.Domain = replacement.Domain
	}

	if replacement.HTTPPort != 0 {
		o.HTTPPort = replacement.HTTPPort
	}
}
