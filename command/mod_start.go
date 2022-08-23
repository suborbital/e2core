package command

import (
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel/trace"

	"github.com/suborbital/deltav/server/release"
	"github.com/suborbital/deltav/signaler"
	"github.com/suborbital/sat/sat"
)

func ModStart() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "start [module path or FQMN]",
		Short:   "start a DeltaV module",
		Long:    "starts a single module and connects to the mesh to receive jobs",
		Version: release.DeltavServerDotVersion,
		RunE: func(cmd *cobra.Command, args []string) error {
			path := ""
			if len(args) > 0 {
				path = args[0]
			}

			config, err := sat.ConfigFromRunnableArg(path)
			if err != nil {
				return errors.Wrap(err, "failed to ConfigFromRunnableArg")
			}

			sat, err := sat.New(config, trace.NewNoopTracerProvider())
			if err != nil {
				return errors.Wrap(err, "failed to sat.New")
			}

			signaler := signaler.Setup()
			signaler.Start(sat.Start)

			return signaler.Wait(time.Second * 5)
		},
	}

	cmd.SetVersionTemplate("{{.Version}}\n")

	return cmd
}
