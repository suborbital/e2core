package rt

import (
	"github.com/pkg/errors"
	"github.com/suborbital/reactr/rcap"
	"github.com/suborbital/vektor/vlog"
)

var ErrCapabilityNotAvailable = errors.New("capability not available")

// Capabilities define the capabilities available to a Runnable
type Capabilities struct {
	LoggerSource rcap.LoggerSource
	HTTPClient   rcap.HTTPClient
	FileSource   rcap.FileSource
	Cache        rcap.Cache

	// RequestHandler and doFunc are special because they are more
	// sensitive; they could cause memory leaks or expose internal state,
	// so they cannot be swapped out for a different implementation.
	RequestHandler *rcap.RequestHandler
	doFunc         coreDoFunc
}

func defaultCaps(logger *vlog.Logger) Capabilities {
	caps := Capabilities{
		LoggerSource: rcap.DefaultLoggerSource(logger),
		HTTPClient:   rcap.DefaultHTTPClient(),
		FileSource:   rcap.DefaultFileSource(nil), // set file access to nil by default, it can be set later.
		Cache:        rcap.DefaultCache(),

		// RequestHandler and doFunc don't get set here since they are set by
		// the rt and rwasm internals; a better solution for this should probably be found
	}

	return caps
}
