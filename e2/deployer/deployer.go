package deployer

import (
	"github.com/pkg/errors"

	"github.com/suborbital/e2core/e2/cli/util"
	"github.com/suborbital/e2core/e2/project"
)

type Deployer struct {
	log util.FriendlyLogger
}

type DeployJob interface {
	Type() string
	Deploy(logger util.FriendlyLogger, pctx *project.Context) error
}

// New creates a new Deployer.
func New(log util.FriendlyLogger) *Deployer {
	d := &Deployer{
		log: log,
	}

	return d
}

// Deploy executes a DeployJob.
func (d *Deployer) Deploy(ctx *project.Context, job DeployJob) error {
	if err := job.Deploy(d.log, ctx); err != nil {
		return errors.Wrapf(err, "deploy job %s failed", job.Type())
	}

	return nil
}
