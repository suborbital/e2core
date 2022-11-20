package publisher

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/suborbital/e2core/e2/cli/util"
	"github.com/suborbital/e2core/e2/project"
)

const (
	DockerPublishJobType = "docker"
)

type DockerPublishJob struct{}

// NewDockerPublishJob returns a new PublishJob for Docker images.
func NewDockerPublishJob() PublishJob {
	d := &DockerPublishJob{}

	return d
}

// Type returns the publish job's type.
func (b *DockerPublishJob) Type() string {
	return DockerPublishJobType
}

// Publish publishes the application.
func (b *DockerPublishJob) Publish(log util.FriendlyLogger, ctx *project.Context) error {
	if ctx.TenantConfig == nil {
		return errors.New("cannot publish without tenant.json")
	}

	if !ctx.Bundle.Exists {
		return errors.New("cannot publish without runnables.wasm.zip, run `e2 build` first")
	}

	imageName, err := project.DockerNameFromConfig(ctx.TenantConfig)
	if err != nil {
		return errors.Wrap(err, "failed to DockerNameFromConfig")
	}

	if _, err := util.Command.Run(fmt.Sprintf("docker buildx build . --platform linux/amd64,linux/arm64 -t %s --push", imageName)); err != nil {
		return errors.Wrap(err, "failed to Run docker")
	}

	util.LogDone(fmt.Sprintf("pushed Docker image -> %s", imageName))

	return nil
}
