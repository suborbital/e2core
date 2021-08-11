package rcap

import (
	"github.com/pkg/errors"
	"github.com/suborbital/vektor/vlog"
)

var ErrCapabilityNotEnabled = errors.New("capability is not enabled")

// CapabilityConfig is configuration for a Runnable's capabilities
type CapabilityConfig struct {
	Logger         LoggerConfig         `json:"logger" yaml:"logger"`
	HTTP           HTTPConfig           `json:"http" yaml:"http"`
	GraphQL        GraphQLConfig        `json:"graphql" yaml:"graphql"`
	Auth           AuthConfig           `json:"auth" yaml:"auth"`
	Cache          CacheConfig          `json:"cache" yaml:"cache"`
	File           FileConfig           `json:"file" yaml:"file"`
	RequestHandler RequestHandlerConfig `json:"requestHandler" yaml:"requestHandler"`
}

// DefaultCapabilityConfig returns the default all-enabled config (with a default logger)
func DefaultCapabilityConfig() CapabilityConfig {
	return DefaultConfigWithLogger(vlog.Default())
}

func DefaultConfigWithLogger(logger *vlog.Logger) CapabilityConfig {
	c := CapabilityConfig{
		Logger: LoggerConfig{
			Enabled: true,
			Logger:  logger,
		},
		HTTP: HTTPConfig{
			Enabled: true,
		},
		GraphQL: GraphQLConfig{
			Enabled: true,
		},
		Auth: AuthConfig{
			Enabled: true,
		},
		Cache: CacheConfig{
			Enabled: true,
		},
		File: FileConfig{
			Enabled: true,
		},
		RequestHandler: RequestHandlerConfig{
			Enabled: true,
		},
	}

	return c
}
