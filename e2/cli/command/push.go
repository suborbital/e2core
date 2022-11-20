package command

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/suborbital/e2core/e2/cli/util"
	"github.com/suborbital/e2core/e2/project"
	"github.com/suborbital/e2core/e2/publisher"
)

var validPublishTypes = map[string]bool{
	"bindle": true,
	"docker": true,
}

// PushCmd packages the current project into a Bindle and pushes it to a Bindle server.
func PushCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "push",
		Short: "Publish a project",
		Long:  "Publish the current project to a remote server (Docker, Bindle, etc.)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			publishType := args[0]
			if _, valid := validPublishTypes[publishType]; !valid {
				return fmt.Errorf("invalid publish type %s", publishType)
			}

			cwd, err := os.Getwd()
			if err != nil {
				return errors.Wrap(err, "failed to Getwd")
			}

			ctx, err := project.ForDirectory(cwd)
			if err != nil {
				return errors.Wrap(err, "failed to project.ForDirectory")
			}

			pshr := publisher.New(&util.PrintLogger{})
			var pubJob publisher.PublishJob

			switch publishType {
			case publisher.BindlePublishJobType:
				pubJob = publisher.NewBindlePublishJob()
			case publisher.DockerPublishJobType:
				pubJob = publisher.NewDockerPublishJob()
			default:
				return fmt.Errorf("invalid push destination %s", publishType)
			}

			if err := pshr.Publish(ctx, pubJob); err != nil {
				return errors.Wrap(err, "failed to Publish")
			}

			return nil
		},
	}

	return cmd
}
