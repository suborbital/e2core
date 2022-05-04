package capabilities

import (
	"github.com/pkg/errors"

	"github.com/suborbital/reactr/rcap"
	"github.com/suborbital/vektor/vlog"
	"github.com/suborbital/velocity/server/appsource"
)

// ResolveFromSource takes the ident, namespace, and version, and looks up the capabilities for that trio from the
// AppSource applying the user overrides over the default configurations.
func ResolveFromSource(source appsource.AppSource, ident, namespace, version string, log *vlog.Logger) (rcap.CapabilityConfig, error) {
	defaultConfig := rcap.DefaultCapabilityConfig()

	userConfig := source.Capabilities(ident, namespace, version)
	if userConfig == nil {
		return defaultConfig, nil
	}

	connections := source.Connections(ident, version)

	if userConfig.Logger != nil {
		defaultConfig.Logger = userConfig.Logger
	}

	if userConfig.HTTP != nil {
		defaultConfig.HTTP = userConfig.HTTP
	}

	if userConfig.GraphQL != nil {
		defaultConfig.GraphQL = userConfig.GraphQL
	}

	if userConfig.Auth != nil {
		defaultConfig.Auth = userConfig.Auth
	}

	// defaultConfig for the cache can come from either the capabilities
	// and/or connections sections of the directive.
	if userConfig.Cache != nil {
		defaultConfig.Cache = userConfig.Cache
	}

	if connections.Redis != nil {
		redisConfig := &rcap.RedisConfig{
			ServerAddress: connections.Redis.ServerAddress,
			Username:      connections.Redis.Username,
			Password:      connections.Redis.Password,
		}

		defaultConfig.Cache.RedisConfig = redisConfig
	}

	if connections.Database != nil {
		queries := source.Queries(ident, version)

		dbConfig, err := connections.Database.ToRCAPConfig(queries)
		if err != nil {
			return defaultConfig, errors.Wrap(err, "failed to ToRCAPConfig")
		}

		defaultConfig.DB = dbConfig
	}

	if userConfig.File != nil {
		defaultConfig.File = userConfig.File
	}

	// Override the connections.Database struct
	if userConfig.DB != nil && userConfig.DB.Enabled {
		defaultConfig.DB = userConfig.DB
	}

	if userConfig.RequestHandler != nil {
		defaultConfig.RequestHandler = userConfig.RequestHandler
	}

	f := func(pathName string) ([]byte, error) {
		return source.File(ident, version, pathName)
	}

	defaultConfig.Logger.Logger = log
	defaultConfig.File.FileFunc = f

	return defaultConfig, nil
}
