package api

import (
	"github.com/pkg/errors"
	"github.com/rs/zerolog"

	"github.com/suborbital/systemspec/capabilities"
)

type HostAPI interface {
	HostFunctions() []HostFn
}

type defaultAPI struct {
	capabilities *capabilities.Capabilities
	logger       zerolog.Logger
}

// New returns the default engine API with the default config (everything enabled)
func New(log zerolog.Logger) HostAPI {
	config := capabilities.DefaultCapabilityConfig()

	// the default config will never cause this to error
	d, _ := NewWithConfig(log, config)

	return d
}

// NewWithConfig returns the default engine API with the given config
func NewWithConfig(log zerolog.Logger, config capabilities.CapabilityConfig) (HostAPI, error) {
	caps, err := capabilities.NewWithConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, "failed to capabilities.NewWithConfig")
	}

	d := &defaultAPI{
		capabilities: caps,
		logger:       log,
	}

	return d, nil
}

// HostFunctions returns the available host functions
func (d *defaultAPI) HostFunctions() []HostFn {
	fns := []HostFn{
		d.ReturnResultHandler(),
		d.ReturnErrorHandler(),
		d.GetFFIResultHandler(),
		d.AddFFIVariableHandler(),
		d.FetchURLHandler(),
		d.LogMsgHandler(),
		d.RequestGetFieldHandler(),
		d.RequestSetFieldHandler(),
		d.RespSetHeaderHandler(),
	}

	return fns
}
