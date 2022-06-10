package config

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/sethvargo/go-envconfig"
)

const (
	DefaultControlPlane = "localhost:9090"
)

type Config struct {
	BundlePath   string `env:"bundle_path"`
	ExecMode     string `env:"VELOCITY_EXEC_MODE,default=metal"`
	SatTag       string `env:"VELOCITY_SAT_VERSION,default=latest"`
	ControlPlane string `env:"VELOCITY_CONTROL_PLANE,overwrite"`
	EnvToken     string `env:"VELOCITY_ENV_TOKEN"`
	UpstreamHost string `env:"VELOCITY_UPSTREAM_HOST"`
	Headless     bool   `env:"VELOCITY_HEADLESS,default=false"`
}

// Parse will return a resolved config struct configured by a combination of environment variables and command line
// arguments.
func Parse(bundlePath string, configLookuper envconfig.Lookuper) (Config, error) {
	c := Config{
		ControlPlane: DefaultControlPlane,
	}

	ctx, ctxCancel := context.WithTimeout(context.Background(), time.Second)
	defer ctxCancel()

	if err := envconfig.ProcessWith(ctx, &c, configLookuper); err != nil {
		return Config{}, errors.Wrap(err, "resolving config: envconfig.Process")
	}

	if c.ControlPlane == DefaultControlPlane && bundlePath == "" {
		return Config{}, errors.New("missing required argument: bundle path")
	} else if bundlePath != "" {
		c.BundlePath = bundlePath
	}

	return c, nil
}
