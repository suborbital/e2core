package capabilities

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
	Logger  *LoggerConfig         `json:"logger,omitempty" yaml:"logger,omitempty"`
	HTTP    *HTTPConfig           `json:"http,omitempty" yaml:"http,omitempty"`
	GraphQL *GraphQLConfig        `json:"graphql,omitempty" yaml:"graphql,omitempty"`
	Auth    *AuthConfig           `json:"auth,omitempty" yaml:"auth,omitempty"`
	Cache   *CacheConfig          `json:"cache,omitempty" yaml:"cache,omitempty"`
	File    *FileConfig           `json:"file,omitempty" yaml:"file,omitempty"`
	DB      *DatabaseConfig       `json:"db" yaml:"db"`
	Request *RequestHandlerConfig `json:"requestHandler,omitempty" yaml:"requestHandler,omitempty"`
	Secrets *SecretsConfig        `json:"secrets,omitempty" yaml:"secrets,omitempty"`
}

// DefaultCapabilityConfig returns the default all-enabled config (with a default logger)
func DefaultCapabilityConfig() CapabilityConfig {
	return DefaultConfigWithLogger(vlog.Default())
}

// DefaultConfigWithLogger returns a capability config with a custom logger
func DefaultConfigWithLogger(logger *vlog.Logger) CapabilityConfig {
	return NewConfig(logger, "", "", nil)
}

// DefaultConfigWithDB returns a capability config with a custom logger and database configured
func DefaultConfigWithDB(logger *vlog.Logger, dbType, dbConnString string, queries []Query) CapabilityConfig {
	return NewConfig(logger, dbType, dbConnString, queries)
}

func NewConfig(logger *vlog.Logger, dbType, dbConnString string, queries []Query) CapabilityConfig {
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
		DB: &DatabaseConfig{
			Enabled:          dbConnString != "",
			DBType:           dbType,
			ConnectionString: dbConnString,
			Queries:          queries,
		},
		Request: &RequestHandlerConfig{
			Enabled:       true,
			AllowGetField: true,
			AllowSetField: true,
		},
		Secrets: &SecretsConfig{
			Enabled: true,
			Env:     nil,
		},
	}

	return c
}
