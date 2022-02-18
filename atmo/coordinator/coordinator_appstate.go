package coordinator

import (
	"github.com/pkg/errors"
)

func (c *Coordinator) SyncAppState() {
	c.log.Debug("syncing AppSource state")

	// mount all the Wasm Runnables into the executor Reactr instance by passing the entire appsource in.
	if err := c.exec.Load(c.App); err != nil {
		c.log.Error(errors.Wrap(err, "failed to exec.Load"))
	}
}
