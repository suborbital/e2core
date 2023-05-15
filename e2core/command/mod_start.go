package command

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/sethvargo/go-envconfig"
	"github.com/spf13/cobra"

	"github.com/suborbital/e2core/e2core/release"
	"github.com/suborbital/e2core/sat/sat"
	"github.com/suborbital/e2core/sat/sat/metrics"
	satOptions "github.com/suborbital/e2core/sat/sat/options"
)

func ModStart() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "start [module path or FQMN]",
		Short:   "start a E2Core module",
		Long:    "starts a single module and connects to the mesh to receive jobs",
		Version: release.Version,
		RunE: func(cmd *cobra.Command, args []string) error {
			path := ""
			if len(args) > 0 {
				path = args[0]
			}

			opts, err := satOptions.Resolve(envconfig.OsLookuper())
			if err != nil {
				return errors.Wrap(err, "options.Resolve")
			}

			zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
			l := zerolog.New(os.Stderr).With().
				Timestamp().
				Str("port", string(opts.Port)).
				Str("procuuid", string(opts.ProcUUID)).
				Int("pid", os.Getpid()).
				Int("ppid", os.Getppid()).
				Str("mode", "bebby").
				Str("fqmn", path).
				Logger()

			config, err := sat.ConfigFromModuleArg(l, opts, path)
			if err != nil {
				return errors.Wrap(err, "failed to ConfigFromModuleArg")
			}

			// Sat will obey SAT_HTTP_PORT from the env and the flag can override that
			// If none present, a random port will be selected
			httpPort, err := cmd.Flags().GetInt(httpPortFlag)
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("reading flag '--%s'", httpPortFlag))
			}
			if httpPort > 0 {
				config.Port = httpPort
				l.Info().Int("port", httpPort).Msg(fmt.Sprintf("Using port :%d for the sat backend", httpPort))
			}

			l.Info().Interface("sdkTrace-config", config.TracerConfig).Msg("this is the sdkTrace config we're using")

			traceProvider, err := sat.SetupTracing(config.TracerConfig, l)
			if err != nil {
				return errors.Wrap(err, "setup tracing")
			}

			l.Info().Msg("successfully set up tracing")

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

			// Wasmtime mmaps huge chunks of memory per module instantiation, so we instruct the GC to aggressively
			// reclaim memory to prevent OOMs
			debug.SetGCPercent(15)

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

	cmd.Flags().Int(httpPortFlag, 0, "if passed, it sets the HTTP service port, otherwise a random high port will be used")

	cmd.SetVersionTemplate("{{.Version}}\n")

	return cmd
}
