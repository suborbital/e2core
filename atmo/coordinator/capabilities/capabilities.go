package capabilities

import (
	"github.com/pkg/errors"
	"github.com/suborbital/atmo/atmo/appsource"
	"github.com/suborbital/reactr/rcap"
	"github.com/suborbital/vektor/vlog"
)

// Render "renders" capabilities by layering any user-defined
// capabilities onto the provided set, thus allowing the user to omit any
// individual capability (or all of them) to receive the defaults.
func Render(config rcap.CapabilityConfig, source appsource.AppSource, log *vlog.Logger) (rcap.CapabilityConfig, error) {
	userConfig := source.Capabilities()
	if userConfig == nil {
		return config, nil
	}

	connections := source.Connections()

	if userConfig.Logger != nil {
		config.Logger = userConfig.Logger
	}

	if userConfig.HTTP != nil {
		config.HTTP = userConfig.HTTP
	}

	if userConfig.GraphQL != nil {
		config.GraphQL = userConfig.GraphQL
	}

	if userConfig.Auth != nil {
		config.Auth = userConfig.Auth
	}

	// config for the cache can come from either the capabilities
	// and/or connections sections of the directive
	if userConfig.Cache != nil {
		config.Cache = userConfig.Cache
	}

	if connections.Redis != nil {
		config.Cache.RedisConfig = connections.Redis
	}

	if connections.Database != nil {
		queries := source.Queries()

		dbConfig, err := connections.Database.ToRCAPConfig(queries)
		if err != nil {
			return config, errors.Wrap(err, "failed to ToRCAPConfig")
		}

		config.DB = dbConfig
	}

	if userConfig.File != nil {
		config.File = userConfig.File
	}

	if userConfig.RequestHandler != nil {
		config.RequestHandler = userConfig.RequestHandler
	}

	// The following are extra inputs needed to make things work

	config.Logger.Logger = log
	config.File.FileFunc = source.File
	config.Auth.Headers = source.Authentication().Domains

	return config, nil
}
