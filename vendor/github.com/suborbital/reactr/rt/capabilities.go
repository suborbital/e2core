package rt

import (
	"github.com/pkg/errors"
	"github.com/suborbital/reactr/rcap"
	"github.com/suborbital/vektor/vlog"
)

var ErrCapabilityNotAvailable = errors.New("capability not available")

// Capabilities define the capabilities available to a Runnable
type Capabilities struct {
	config rcap.CapabilityConfig

	Auth          rcap.AuthCapability
	LoggerSource  rcap.LoggerCapability
	HTTPClient    rcap.HTTPCapability
	GraphQLClient rcap.GraphQLCapability
	FileSource    rcap.FileCapability
	Cache         rcap.CacheCapability

	// RequestHandler and doFunc are special because they are more
	// sensitive; they could cause memory leaks or expose internal state,
	// so they cannot be swapped out for a different implementation.
	RequestHandler rcap.RequestHandlerCapability
	doFunc         coreDoFunc
}

// DefaultCapabilities returns the default capabilities with the provided Logger
func DefaultCapabilities(logger *vlog.Logger) Capabilities {
	return CapabilitiesFromConfig(rcap.DefaultConfigWithLogger(logger))
}

func CapabilitiesFromConfig(config rcap.CapabilityConfig) Capabilities {
	caps := Capabilities{
		config:        config,
		Auth:          rcap.DefaultAuthProvider(*config.Auth),
		LoggerSource:  rcap.DefaultLoggerSource(*config.Logger),
		HTTPClient:    rcap.DefaultHTTPClient(*config.HTTP),
		GraphQLClient: rcap.DefaultGraphQLClient(*config.GraphQL),
		FileSource:    rcap.DefaultFileSource(*config.File),
		Cache:         rcap.SetupCache(*config.Cache),

		// RequestHandler and doFunc don't get set here since they are set by
		// the rt and rwasm internals; a better solution for this should probably be found
	}

	return caps
}

// Config returns the configuration that was used to create the Capabilities
// the config cannot be changed, but it can be used to determine what was
// previously set so that the orginal config (like enabled settings) can be respected
func (c Capabilities) Config() rcap.CapabilityConfig {
	return c.config
}
