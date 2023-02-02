package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"

	"github.com/suborbital/e2core/sat/sat"
	"github.com/suborbital/e2core/sat/sat/metrics"
)

func main() {
	logger := zerolog.New(os.Stderr).With().
		Str("service", "sat-module").
		Str("version", sat.SatDotVersion).
		Logger()

	conf, err := sat.ConfigFromArgs(logger)
	if err != nil {
		log.Fatal(err)
	}

	if err = start(logger, conf); err != nil {
		logger.Err(err).Msg("startup")
		os.Exit(1)
	}
}

// start starts up the Sat instance
func start(logger zerolog.Logger, conf *sat.Config) error {
	traceProvider, err := sat.SetupTracing(conf.TracerConfig, logger)
	if err != nil {
		return errors.Wrap(err, "setup tracing")
	}
	defer traceProvider.Shutdown(context.Background())

	mctx, mcancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer mcancel()

	mtx, err := metrics.ResolveMetrics(mctx, conf.MetricsConfig)
	if err != nil {
		return errors.Wrap(err, "metrics.ResolveMetrics")
	}

	// initialize Reactr, Vektor, and Grav and wrap them in a sat instance
	s, err := sat.New(conf, logger, traceProvider, mtx)
	if err != nil {
		return errors.Wrap(err, "sat.New")
	}

	monitor, err := NewMonitor(logger, conf)
	if err != nil {
		return errors.Wrap(err, "failed to createProcFile")
	}

	serverErrors := make(chan error, 1)
	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, syscall.SIGINT, syscall.SIGTERM)

	// Start sat.
	go func() {
		ctx, cxl := context.WithTimeout(context.Background(), 5*time.Second)
		defer cxl()
		err := s.Start(ctx)
		serverErrors <- errors.Wrap(err, "sat start")
	}()

	// Start monitor.
	go func() {
		ctx, cxl := context.WithTimeout(context.Background(), 5*time.Second)
		defer cxl()
		err := monitor.Start(ctx)
		serverErrors <- errors.Wrap(err, "monitor start")
	}()

	select {
	case err := <-serverErrors:
		return errors.Wrap(err, "server error")
	case sig := <-shutdownChan:
		logger.Info().Str("signal", sig.String()).Msg("signal received, shutdown started")

		monitor.Stop()
		satErr := s.Shutdown()
		if satErr != nil {
			return errors.Wrap(satErr, "sat shutdown")
		}
	}

	return nil
}
