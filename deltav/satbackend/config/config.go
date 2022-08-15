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
	BundlePath   string `env:"DELTAV_BUNDLE_PATH,default=./modules.wasm.zip"`
	ExecMode     string `env:"DELTAV_EXEC_MODE,default=metal"`
	SatTag       string `env:"DELTAV_SAT_VERSION,default=latest"`
	ControlPlane string `env:"DELTAV_CONTROL_PLANE,overwrite"`
	EnvToken     string `env:"DELTAV_ENV_TOKEN"`
	UpstreamHost string `env:"DELTAV_UPSTREAM_HOST"`
	Headless     bool   `env:"DELTAV_HEADLESS,default=false"`
}

// Parse will return a resolved config struct configured by a combination of environment variables and command line
// arguments.
func Parse(bundlePath string, configLookuper envconfig.Lookuper) (Config, error) {
	ctx, ctxCancel := context.WithTimeout(context.Background(), time.Second)
	defer ctxCancel()

	c := Config{}
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
