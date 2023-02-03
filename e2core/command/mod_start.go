package command

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"

	"github.com/suborbital/e2core/e2core/release"
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

			l := zerolog.New(os.Stderr).With().Timestamp().Str("command", "mod start").Logger()

			config, err := sat.ConfigFromModuleArg(l, path)
			if err != nil {
				return errors.Wrap(err, "failed to ConfigFromModuleArg")
			}

			traceProvider, err := sat.SetupTracing(config.TracerConfig, l)
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

			satInstance, err := sat.New(config, l, traceProvider, mtx)
			if err != nil {
				return errors.Wrap(err, "failed to sat.New")
			}

			shutdown := make(chan os.Signal, 1)
			signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

			serverErrors := make(chan error, 1)

			go func() {
				l.Info().Msg("starting server")
				err := satInstance.Start()
				if err != nil {
					serverErrors <- errors.Wrap(err, "srv.Start")
				}
			}()

			select {
			case err := <-serverErrors:
				return fmt.Errorf("server error: %w", err)

			case sig := <-shutdown:
				l.Info().Str("signal", sig.String()).Str("status", "shutdown started").Msg("shutdown started")
				defer l.Info().Str("status", "shutdown complete").Msg("all done")

				srvErr := satInstance.Shutdown()
				if srvErr != nil {
					return errors.Wrap(srvErr, "srv.Shutdown")
				}
			}

			return nil
		},
	}

	cmd.SetVersionTemplate("{{.Version}}\n")

	return cmd
}
