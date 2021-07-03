package rt

import (
	"github.com/pkg/errors"
	"github.com/suborbital/reactr/rcap"
)

var ErrCapabilityNotAvailable = errors.New("capability not available")

// Capabilities define the capabilities available to a Runnable
type Capabilities struct {
	LoggerSource   rcap.LoggerSource
	RequestHandler rcap.RequestHandler
	HTTPClient     rcap.HTTPClient
	FileSource     rcap.FileSource
	Cache          rcap.Cache

	// doFunc is not exposed as it would make a private rt function available outside the package
	doFunc coreDoFunc
}

func defaultCaps() Capabilities {
	caps := Capabilities{
		LoggerSource:   rcap.DefaultLoggerSource(),
		RequestHandler: rcap.DefaultRequestHandler(),
		HTTPClient:     rcap.DefaultHTTPClient(),
		FileSource:     rcap.DefaultFileSource(nil), // set file access to nil by default, it can be set later.
		Cache:          rcap.DefaultCache(),
	}

	return caps
}
