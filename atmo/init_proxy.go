//go:build proxy

package atmo

import "github.com/suborbital/vektor/vlog"

func setupLogger(*vlog.Logger) {
	// do nothing in proxy mode
}
