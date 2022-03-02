//go:build !proxy

package atmo

import (
	"github.com/suborbital/atmo/atmo/options"
	"github.com/suborbital/reactr/rwasm/runtime"
	"github.com/suborbital/vektor/vlog"
)

// we only initalize the RWASM logger if we're not in proxy mode.
func setupLogger(logger *vlog.Logger) {
	runtime.UseInternalLogger(logger)
}

// setupTracing in non-proxy mode will be a noop.
func setupTracing(_ options.TracerConfig) (func(), error) {
	// do nothing when we're not in proxy mode.
	return func() {}, nil
}
