package command

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/suborbital/e2core/e2/cli/release"
	"github.com/suborbital/e2core/e2/cli/util"
	"github.com/suborbital/e2core/e2/project"
)

// DevCmd returns the dev command.
func DevCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dev",
		Short: "Run a development server using Docker",
		Long:  `Run a development server using Docker`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return errors.Wrap(err, "failed to Getwd")
			}

			bctx, err := project.ForDirectory(cwd)
			if err != nil {
				return errors.Wrap(err, "failed to project.ForDirectory")
			}

			if bctx.TenantConfig == nil {
				return errors.New("current directory is not a project; tenant.json is missing")
			}

			port, _ := cmd.Flags().GetString("port")
			verbose, _ := cmd.Flags().GetBool("verbose")

			envvar := ""

			if verbose {
				envvar = "-e DELTAV_LOG_LEVEL=debug"
				util.LogInfo("Running DeltaV with debug logging")
			}

			dockerCmd := fmt.Sprintf("docker run -v=%s:/home/atmo -e=DELTAV_HTTP_PORT=%s %s -p=%s:%s suborbital/deltav:%s deltav start", bctx.Cwd, port, envvar, port, port, release.RuntimeVersion)

			_, err = util.Command.Run(dockerCmd)
			if err != nil {
				return errors.Wrap(err, "🚫 failed to run dev server")
			}

			return nil
		},
	}

	cmd.Flags().String("port", "8080", "set the port on which to serve the project")
	cmd.Flags().BoolP("verbose", "v", false, "run with debug level logging")
	cmd.Flags().Lookup("verbose").NoOptDefVal = "true"

	return cmd
}
