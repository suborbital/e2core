package publisher

import (
	"github.com/pkg/errors"

	"github.com/suborbital/e2core/e2/cli/util"
	"github.com/suborbital/e2core/e2/project"
)

// Publisher is responsible for publishing projects.
type Publisher struct {
	log util.FriendlyLogger
}

// New creates a new Publisher.
func New(log util.FriendlyLogger) *Publisher {
	p := &Publisher{
		log: log,
	}

	return p
}

// PublishJob represents an attempt to publish a packaged application.
type PublishJob interface {
	Type() string
	Publish(logger util.FriendlyLogger, pctx *project.Context) error
}

// Publish executes a PublishJob.
func (p *Publisher) Publish(ctx *project.Context, job PublishJob) error {
	if err := job.Publish(p.log, ctx); err != nil {
		return errors.Wrapf(err, "publish job %s failed", job.Type())
	}

	return nil
}
