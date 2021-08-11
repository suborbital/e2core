package coordinator

import (
	"github.com/pkg/errors"
	"github.com/suborbital/atmo/bundle/load"
	"github.com/suborbital/reactr/rcap"
)

func (c *Coordinator) SyncAppState() {
	c.log.Debug("syncing AppSource state")

	var authConfig *rcap.AuthConfig
	if domains := c.App.Authentication().Domains; domains != nil {
		authConfig = &rcap.AuthConfig{Enabled: true, Headers: domains}
	}

	// take the default capabilites from the Reactr instance
	caps := c.reactr.DefaultCaps()
	// set our own FileSource that is connected to the Bundle's FileFunc
	caps.FileSource = rcap.DefaultFileSource(rcap.FileConfig{Enabled: true, FileFunc: c.App.File})
	// set our own auth provider based on the Directive
	if authConfig != nil {
		caps.Auth = rcap.DefaultAuthProvider(*authConfig)
	}

	// mount all of the Wasm Runnables into the Reactr instance
	// pass 'false' for registerSimpleName since we'll only ever call
	// functions by their FQFNs
	if err := load.Runnables(c.reactr, c.App.Runnables(), caps, false); err != nil {
		c.log.Error(errors.Wrap(err, "failed to load.Runnables"))
	}

	// connect a Grav pod to each function
	for _, fn := range c.App.Runnables() {
		if fn.FQFN == "" {
			c.log.ErrorString("fn", fn.Name, "missing calculated FQFN, will not be available")
			continue
		}

		// check to see if we're already listening for this function
		// on the local Reactr instance, and start if need be
		if _, exists := c.listening.Load(fn.FQFN); !exists {
			c.log.Debug("adding listener for", fn.FQFN)
			c.reactr.Listen(c.grav.Connect(), fn.FQFN)

			c.listening.Store(fn.FQFN, true)
		}
	}
}
