package packager

import (
	"github.com/pkg/errors"

	"github.com/suborbital/e2core/e2/cli/util"
	"github.com/suborbital/e2core/e2/project"
)

// Packager is responsible for packaging and publishing projects.
type Packager struct {
	log util.FriendlyLogger
}

// PackageJob represents a specific type of packaging,
// for example modules into bundle, bundle into container image, etc.
type PackageJob interface {
	Type() string
	Package(logger util.FriendlyLogger, pctx *project.Context) error
}

// New creates a new Packager.
func New(log util.FriendlyLogger) *Packager {
	p := &Packager{
		log: log,
	}

	return p
}

// Package executes the given set of PackageJobs, returning an error if any fail.
func (p *Packager) Package(ctx *project.Context, jobs ...PackageJob) error {
	for _, j := range jobs {
		if err := j.Package(p.log, ctx); err != nil {
			return errors.Wrapf(err, "package job %s failed", j.Type())
		}
	}

	return nil
}
