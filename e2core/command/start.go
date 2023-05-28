package command

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/suborbital/e2core/e2core/auth"
	"github.com/suborbital/e2core/e2core/backend/satbackend"
	"github.com/suborbital/e2core/e2core/options"
	"github.com/suborbital/e2core/e2core/release"
	"github.com/suborbital/e2core/e2core/server"
	"github.com/suborbital/e2core/e2core/sourceserver"
	"github.com/suborbital/e2core/e2core/syncer"
	"github.com/suborbital/e2core/nuexecutor/overviews"
	"github.com/suborbital/systemspec/system/bundle"
	"github.com/suborbital/systemspec/system/client"
)

const (
	shutdownWaitTime = time.Second * 10
)

func Start() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "start [bundle-path]",
		Short:   "start the e2core server",
		Long:    "starts the e2core server using the provided options",
		Version: release.Version,
		RunE: func(cmd *cobra.Command, args []string) error {
			path := "./modules.wasm.zip"
			if len(args) > 0 {
				path = args[0]
			}

			logger := setupLogger()

			mods, err := modsFromFlags(cmd.Flags())
			if err != nil {
				return errors.Wrap(err, "failed to modsFromFlags")
			}

			opts, err := options.NewWithModifiers(append(mods, options.UseBundlePath(path))...)
			if err != nil {
				return errors.Wrap(err, "options.NewWithModifiers")
			}

			sync := setupSyncer(logger, opts)

			rep := overviews.NewRepository(overviews.Config{Endpoint: opts.ControlPlane}, logger)
			rep.Start()

			// create the three essential parts:
			sourceSrv, err := setupSourceServer(logger, opts)
			if err != nil {
				return errors.Wrap(err, "failed to setupSourceServer")
			}

			backend, err := satbackend.New(logger, opts, sync)
			if err != nil {
				return errors.Wrap(err, "failed to satbackend.New")
			}

			srv, err := server.New(logger, sync, opts, rep)
			if err != nil {
				return errors.Wrap(err, "server.New")
			}

			shutdown := make(chan os.Signal, 1)
			signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

			serverErrors := make(chan error, 1)

			// now start all three parts:

			go func() {
				logger.Info().Msg("starting source server")
				if err := sourceserver.Start(sourceSrv); err != nil {
					serverErrors <- errors.Wrap(err, "sourceserver.Start")
				}

			}()

			go func() {
				logger.Info().Msg("starting backend")
				if err := backend.Start(); err != nil {
					serverErrors <- errors.Wrap(err, "backend.Start")
				}

			}()

			go func() {
				logger.Info().Msgf("starting e2core server on port %d", opts.HTTPPort)
				if err := srv.Start(); err != nil {
					serverErrors <- errors.Wrap(err, "srv.Start")
				}
			}()

			select {
			case err := <-serverErrors:
				return fmt.Errorf("server error: %w", err)
			case sig := <-shutdown:
				rep.Shutdown()

				logger.Info().Str("signal", sig.String()).Str("status", "shutdown started").Msg("shutdown started")
				defer logger.Info().Str("status", "shutdown complete").Msg("all done")

				ctx, cancel := context.WithTimeout(context.Background(), shutdownWaitTime)
				defer cancel()

				if err := srv.Shutdown(ctx); err != nil {
					return errors.Wrap(err, "srv.Shutdown")
				}

				if sourceSrv != nil {
					if err := sourceSrv.Shutdown(ctx); err != nil {
						return errors.Wrap(err, "sourceSrv.Shutdown")
					}
				}

				backend.Shutdown()
			}

			return nil
		},
	}

	cmd.SetVersionTemplate("{{.Version}}\n")

	cmd.Flags().String(domainFlag, "", "if passed, it'll be used as E2CORE_DOMAIN and HTTPS will be used, otherwise HTTP will be used")
	cmd.Flags().Int(httpPortFlag, 8080, "if passed, it'll be used as E2CORE_HTTP_PORT, otherwise '8080' will be used")
	cmd.Flags().Int(tlsPortFlag, 443, "if passed, it'll be used as E2CORE_TLS_PORT, otherwise '443' will be used")

	return cmd
}

func setupLogger() zerolog.Logger {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	logger := zerolog.New(os.Stderr).With().
		Timestamp().
		Str("mode", "mothership").
		Str("version", release.Version).
		Logger().Level(zerolog.InfoLevel)

	return logger
}

func modsFromFlags(flags *pflag.FlagSet) ([]options.Modifier, error) {
	domain, err := flags.GetString(domainFlag)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("get string flag '%s' value", domainFlag))
	}

	httpPort, err := flags.GetInt(httpPortFlag)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("get int flag '%s' value", httpPortFlag))
	}

	tlsPort, err := flags.GetInt(tlsPortFlag)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("get int flag '%s' value", tlsPortFlag))
	}

	opts := []options.Modifier{
		options.Domain(domain),
		options.HTTPPort(httpPort),
		options.TLSPort(tlsPort),
	}

	return opts, nil
}

func setupSyncer(logger zerolog.Logger, opts *options.Options) *syncer.Syncer {
	systemSource := bundle.NewBundleSource(opts.BundlePath)

	if opts.ControlPlane != "" {
		// the HTTP system source gets Server's data from a remote server
		// which can essentially control Server's behaviour.
		systemSource = client.NewHTTPSource(opts.ControlPlane, auth.NewAccessToken(opts.EnvironmentToken))
	}

	sync := syncer.New(opts, logger, systemSource)

	return sync
}

func setupSourceServer(logger zerolog.Logger, opts *options.Options) (*echo.Echo, error) {
	ll := logger.With().Str("method", "setupSourceServer").Logger()

	// if an external control plane hasn't been set, act as the control plane
	// but if one has been set, use it (and launch all children with it configured)
	if opts.ControlPlane == options.DefaultControlPlane || opts.ControlPlane == "" {
		opts.ControlPlane = options.DefaultControlPlane

		ll.Debug().Msg("creating sourceserver from bundle: " + opts.BundlePath)

		sourceSrv, err := sourceserver.FromBundle(opts.BundlePath)
		if err != nil {
			return nil, errors.Wrap(err, "failed to sourceserver.FromBundle")
		}

		sourceSrv.HideBanner = true

		return sourceSrv, nil
	}

	// a nil server is ok if we don't need to run one
	return nil, nil
}
