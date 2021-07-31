package rt

import (
	"github.com/pkg/errors"
	"github.com/suborbital/reactr/rcap"
	"github.com/suborbital/vektor/vlog"
)

var ErrCapabilityNotAvailable = errors.New("capability not available")

// Capabilities define the capabilities available to a Runnable
type Capabilities struct {
	Auth          rcap.AuthProvider
	LoggerSource  rcap.LoggerSource
	HTTPClient    rcap.HTTPClient
	GraphQLClient rcap.GraphQLClient
	FileSource    rcap.FileSource
	Cache         rcap.Cache

	// RequestHandler and doFunc are special because they are more
	// sensitive; they could cause memory leaks or expose internal state,
	// so they cannot be swapped out for a different implementation.
	RequestHandler *rcap.RequestHandler
	doFunc         coreDoFunc
}

func defaultCaps(logger *vlog.Logger) Capabilities {
	caps := Capabilities{
		Auth:          rcap.DefaultAuthProvider(nil), // no authentication config is set up by default
		LoggerSource:  rcap.DefaultLoggerSource(logger),
		HTTPClient:    rcap.DefaultHTTPClient(),
		GraphQLClient: rcap.DefaultGraphQLClient(),
		FileSource:    rcap.DefaultFileSource(nil), // set file access to nil by default, it can be set later.
		Cache:         rcap.DefaultCache(),

		// RequestHandler and doFunc don't get set here since they are set by
		// the rt and rwasm internals; a better solution for this should probably be found
	}

	return caps
}
