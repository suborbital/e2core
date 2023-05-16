package tracing

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdkTrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"

	"github.com/suborbital/go-kit/observability"
)

const (
	ExporterCollector CollectorType = "collector"
	ExporterHoneycomb CollectorType = "honeycomb"
)

type CollectorType string

type Config struct {
	Type        CollectorType
	ServiceName string
	Probability float64
	Collector   CollectorConfig
	Honeycomb   HoneycombConfig
}

type CollectorConfig struct {
	Endpoint string
}

type HoneycombConfig struct {
	Endpoint string
	APIKey   string
	Dataset  string
}

var Tracer trace.Tracer

// SetupTracing configure open telemetry to be used with otel exporter. Returns a tracer closer func and an error.
func SetupTracing(config Config, logger zerolog.Logger) (*sdkTrace.TracerProvider, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	var traceProvider *sdkTrace.TracerProvider
	var err error

	ll := logger.With().Str("tracerType", string(config.Type)).Logger()

	switch config.Type {
	case ExporterHoneycomb:
		ll.Info().Msg("configuring honeycomb exporter for tracing")

		conn, err := observability.GrpcConnection(ctx, config.Honeycomb.Endpoint, nil)
		if err != nil {
			return nil, errors.Wrap(err, "honeycomb GrpcConnection")
		}

		traceProvider, err = observability.HoneycombTracer(ctx, conn, observability.HoneycombTracingConfig{
			TracingConfig: observability.TracingConfig{
				Probability: config.Probability,
				ServiceName: config.ServiceName,
			},
			APIKey:  config.Honeycomb.APIKey,
			Dataset: config.Honeycomb.Dataset,
		})
		if err != nil {
			return nil, errors.Wrap(err, "observability.HoneycombTracer")
		}

		ll.Info().Msg("created honeycomb sdkTrace exporter")
	case ExporterCollector:
		ll.Info().Msg("configuring collector exporter for tracing")

		conn, err := observability.GrpcConnection(ctx, config.Collector.Endpoint)
		if err != nil {
			ll.Err(err).Msg("observability.GrcpConnection failed")
			return nil, errors.Wrap(err, "collector GrpcConnection")
		}

		traceProvider, err = observability.OtelTracer(ctx, conn, observability.TracingConfig{
			Probability: config.Probability,
			ServiceName: config.ServiceName,
		})
		if err != nil {
			ll.Err(err).Msg("observability.OtelTracer failed")
			return nil, errors.Wrap(err, "observability.OtelTracer")
		}

		ll.Info().Msg("created collector sdkTrace exporter")
	default:
		ll.Warn().Msg("unrecognised tracer type configuration. Defaulting to no tracer")
		fallthrough
	case "none", "":
		// Create the most default sdkTrace provider and escape early.
		traceProvider, err = observability.NoopTracer()
		if err != nil {
			return nil, errors.Wrap(err, "noop Tracer")
		}

		ll.Info().Msg("finished setting up default noop tracer")
	}

	ll.Info().Msg("setting up a global tracer")
	Tracer = traceProvider.Tracer("e2core-bebby-tracing")

	otel.SetTextMapPropagator(propagation.TraceContext{})

	return traceProvider, nil
}
