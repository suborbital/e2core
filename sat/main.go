package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"

	"github.com/suborbital/e2core/foundation/signaler"
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
	traceProvider, err := sat.SetupTracing(conf.TracerConfig, conf.Logger)
	if err != nil {
		return errors.Wrap(err, "setup tracing")
	}

	mctx, mcancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer mcancel()

	mtx, err := metrics.ResolveMetrics(mctx, conf.MetricsConfig)
	if err != nil {
		return errors.Wrap(err, "metrics.ResolveMetrics")
	}

	defer traceProvider.Shutdown(context.Background())

	// initialize Reactr, Vektor, and Grav and wrap them in a sat instance
	s, err := sat.New(conf, logger, traceProvider, mtx)
	if err != nil {
		return errors.Wrap(err, "sat.New")
	}

	monitor, err := NewMonitor(conf.Logger, conf)
	if err != nil {
		return errors.Wrap(err, "failed to createProcFile")
	}

	signaler := signaler.Setup()

	signaler.Start(s.Start)
	signaler.Start(monitor.Start)

	return signaler.Wait(time.Second * 5)
}
