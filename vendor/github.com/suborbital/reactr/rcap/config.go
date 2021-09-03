package rcap

import (
	"github.com/pkg/errors"
	"github.com/suborbital/vektor/vlog"
)

var ErrCapabilityNotEnabled = errors.New("capability is not enabled")

// CapabilityConfig is configuration for a Runnable's capabilities
// NOTE: if any of the individual configs are nil, it will cause a crash,
// but we need to be able to determine if they're set or not, hence the pointers
// we are going to leave capabilities undocumented until we come up with a more elegant solution
type CapabilityConfig struct {
	Logger         *LoggerConfig         `json:"logger,omitempty" yaml:"logger,omitempty"`
	HTTP           *HTTPConfig           `json:"http,omitempty" yaml:"http,omitempty"`
	GraphQL        *GraphQLConfig        `json:"graphql,omitempty" yaml:"graphql,omitempty"`
	Auth           *AuthConfig           `json:"auth,omitempty" yaml:"auth,omitempty"`
	Cache          *CacheConfig          `json:"cache,omitempty" yaml:"cache,omitempty"`
	File           *FileConfig           `json:"file,omitempty" yaml:"file,omitempty"`
	RequestHandler *RequestHandlerConfig `json:"requestHandler,omitempty" yaml:"requestHandler,omitempty"`
}

// DefaultCapabilityConfig returns the default all-enabled config (with a default logger)
func DefaultCapabilityConfig() CapabilityConfig {
	return DefaultConfigWithLogger(vlog.Default())
}

func DefaultConfigWithLogger(logger *vlog.Logger) CapabilityConfig {
	c := CapabilityConfig{
		Logger: &LoggerConfig{
			Enabled: true,
			Logger:  logger,
		},
		HTTP: &HTTPConfig{
			Enabled: true,
			Rules:   defaultHTTPRules(),
		},
		GraphQL: &GraphQLConfig{
			Enabled: true,
			Rules:   defaultHTTPRules(),
		},
		Auth: &AuthConfig{
			Enabled: true,
		},
		Cache: &CacheConfig{
			Enabled: true,
			Rules:   defaultCacheRules(),
		},
		File: &FileConfig{
			Enabled: true,
		},
		RequestHandler: &RequestHandlerConfig{
			Enabled: true,
		},
	}

	return c
}
