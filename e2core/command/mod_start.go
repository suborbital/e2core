package command

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/suborbital/e2core/e2core/release"
	"github.com/suborbital/e2core/foundation/signaler"
	"github.com/suborbital/e2core/sat/sat"
	"github.com/suborbital/e2core/sat/sat/metrics"
)

func ModStart() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "start [module path or FQMN]",
		Short:   "start a E2Core module",
		Long:    "starts a single module and connects to the mesh to receive jobs",
		Version: release.E2CoreServerDotVersion,
		RunE: func(cmd *cobra.Command, args []string) error {
			path := ""
			if len(args) > 0 {
				path = args[0]
			}

			config, err := sat.ConfigFromModuleArg(path)
			if err != nil {
				return errors.Wrap(err, "failed to ConfigFromModuleArg")
			}

			traceProvider, err := sat.SetupTracing(config.TracerConfig, config.Logger)
			if err != nil {
				return errors.Wrap(err, "setup tracing")
			}

			mctx, mcancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer mcancel()

			mtx, err := metrics.ResolveMetrics(mctx, config.MetricsConfig)
			if err != nil {
				return errors.Wrap(err, "metrics.ResolveMetrics")
			}

			defer traceProvider.Shutdown(context.Background())

			sat, err := sat.New(config, traceProvider, mtx)
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
