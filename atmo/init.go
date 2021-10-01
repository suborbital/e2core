//go:build !proxy

package atmo

import (
	"github.com/suborbital/reactr/rwasm"
	"github.com/suborbital/vektor/vlog"
)

// we only initalize the RWASM logger if we're not in proxy mode
func setupLogger(logger *vlog.Logger) {
	rwasm.UseInternalLogger(logger)
}
