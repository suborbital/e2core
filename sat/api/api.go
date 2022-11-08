package api

import (
	"github.com/pkg/errors"

	"github.com/suborbital/appspec/capabilities"
	"github.com/suborbital/e2core/sat/engine/runtime"
)

type HostAPI interface {
	HostFunctions() []runtime.HostFn
}

type defaultAPI struct {
	capabilities *capabilities.Capabilities
}

// New returns the default engine API with the default config (everything enabled)
func New() HostAPI {
	config := capabilities.DefaultCapabilityConfig()

	// the default config will never cause this to error
	d, _ := NewWithConfig(config)

	return d
}

// NewWithConfig returns the default engine API with the given config
func NewWithConfig(config capabilities.CapabilityConfig) (HostAPI, error) {
	caps, err := capabilities.NewWithConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, "failed to capabilities.NewWithConfig")
	}

	d := &defaultAPI{
		capabilities: caps,
	}

	return d, nil
}

// HostFunctions returns the available host functions
func (d *defaultAPI) HostFunctions() []runtime.HostFn {
	fns := []runtime.HostFn{
		d.ReturnResultHandler(),
		d.ReturnErrorHandler(),
		d.GetFFIResultHandler(),
		d.AddFFIVariableHandler(),
		d.FetchURLHandler(),
		d.GraphQLQueryHandler(),
		d.CacheSetHandler(),
		d.CacheGetHandler(),
		d.LogMsgHandler(),
		d.RequestGetFieldHandler(),
		d.RequestSetFieldHandler(),
		d.RespSetHeaderHandler(),
		d.GetStaticFileHandler(),
		d.DBExecHandler(),
		d.AbortHandler(),
		d.GetSecretValueHandler(),
	}

	return fns
}
