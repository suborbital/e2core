// Package metrics provides implementation of metrics with otel exporter / gauges.
package metrics

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/global"
	"go.opentelemetry.io/otel/metric/instrument"
	"go.opentelemetry.io/otel/metric/unit"

	"github.com/suborbital/e2core/sat/sat/options"
	"github.com/suborbital/go-kit/observability"
)

const (
	otelCollectionPeriod = 3 * time.Second
)

// setupOtelMetrics delegates setting up the meter and attaching it to a global scope to the observability package in
// the suborbital/go-kit module.
func setupOtelMetrics(ctx context.Context, config options.MetricsConfig) (Metrics, error) {
	if config.OtelMetrics == nil {
		return Metrics{}, errors.New("resolving otel metrics is missing configuration values")
	}

	conn, err := observability.GrpcConnection(ctx, config.OtelMetrics.Endpoint)
	if err != nil {
		return Metrics{}, errors.Wrap(err, "otel metrics grpc connection")
	}

	_, err = observability.OtelMeter(ctx, conn, observability.MeterConfig{CollectPeriod: otelCollectionPeriod})
	if err != nil {
		return Metrics{}, errors.Wrap(err, "observability.OtelMeter")
	}

	return configureMetrics()
}

// ConfigureMetrics returns a struct with the meters that we want to measure in sat. It assumes that the global meter
// has already been set up (see setupOtelMetrics). Shipping the measured values is the task of the provider, so
// from a usage point of view, nothing else is needed.
func configureMetrics() (Metrics, error) {
	m := global.Meter(
		"sat",
		metric.WithInstrumentationVersion("1.0"),
	)

	si64 := m.SyncInt64()

	functionExecutions, err := si64.Counter(
		"function_executions",
		instrument.WithUnit(unit.Dimensionless),
		instrument.WithDescription("How many function execution requests happened"),
	)
	if err != nil {
		return Metrics{}, errors.Wrap(err, "sync int 64 provider function_executions")
	}

	failedFunctionExecutions, err := si64.Counter(
		"failed_function_executions",
		instrument.WithUnit(unit.Dimensionless),
		instrument.WithDescription("How many function execution requests failed"),
	)
	if err != nil {
		return Metrics{}, errors.Wrap(err, "sync int 64 provider failed_function_executions")
	}

	functionTime, err := si64.Histogram(
		"function_time",
		instrument.WithUnit(unit.Milliseconds),
		instrument.WithDescription("How much time was spent doing function executions"),
	)
	if err != nil {
		return Metrics{}, errors.Wrap(err, "sync int 64 provider function_time")
	}

	return Metrics{
		FunctionExecutions:       functionExecutions,
		FailedFunctionExecutions: failedFunctionExecutions,
		FunctionTime:             functionTime,
	}, nil
}
