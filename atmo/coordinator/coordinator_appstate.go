package coordinator

import (
	"github.com/pkg/errors"
)

func (c *Coordinator) SyncAppState() {
	c.log.Debug("syncing AppSource state")

	// mount all of the Wasm Runnables into the executor Reactr instance
	if err := c.exec.Load(c.App.Runnables()); err != nil {
		c.log.Error(errors.Wrap(err, "failed to exec.Load"))
	}
}
