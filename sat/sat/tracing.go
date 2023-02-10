package sat

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/sdk/trace"

	"github.com/suborbital/e2core/sat/sat/options"
	"github.com/suborbital/go-kit/observability"
)

// SetupTracing configure open telemetry to be used with otel exporter. Returns a tracer closer func and an error.
func SetupTracing(config options.TracerConfig, logger zerolog.Logger) (*trace.TracerProvider, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	ll := logger.With().Str("tracerType", config.TracerType).Logger()

	switch config.TracerType {
	case "honeycomb":
		if config.HoneycombConfig == nil {
			return nil, errors.New("missing honeycomb tracing config values")
		}

		ll.Info().Msg("configuring honeycomb exporter for tracing")

		conn, err := observability.GrpcConnection(ctx, config.HoneycombConfig.Endpoint, nil)
		if err != nil {
			return nil, errors.Wrap(err, "honeycomb GrpcConnection")
		}

		traceProvider, err := observability.HoneycombTracer(ctx, conn, observability.HoneycombTracingConfig{
			TracingConfig: observability.TracingConfig{
				Probability: config.Probability,
				ServiceName: config.ServiceName,
			},
			APIKey:  config.HoneycombConfig.APIKey,
			Dataset: config.HoneycombConfig.Dataset,
		})
		if err != nil {
			return nil, errors.Wrap(err, "observability.HoneycombTracer")
		}

		ll.Info().Msg("created honeycomb trace exporter")

		return traceProvider, nil
	case "collector":
		if config.Collector == nil {
			return nil, errors.New("missing collector tracing config values")
		}

		ll.Info().Msg("configuring collector exporter for tracing")

		conn, err := observability.GrpcConnection(ctx, config.Collector.Endpoint)
		if err != nil {
			return nil, errors.Wrap(err, "collector GrpcConnection")
		}

		traceProvider, err := observability.OtelTracer(ctx, conn, observability.TracingConfig{
			Probability: config.Probability,
			ServiceName: config.ServiceName,
		})
		if err != nil {
			return nil, errors.Wrap(err, "observability.OtelTracer")
		}

		ll.Info().Msg("created collector trace exporter")

		return traceProvider, nil
	default:
		ll.Warn().Msg("unrecognised tracer type configuration. Defaulting to no tracer")
		fallthrough
	case "none", "":
		// Create the most default trace provider and escape early.
		traceProvider, err := observability.NoopTracer()
		if err != nil {
			return nil, errors.Wrap(err, "noop Tracer")
		}

		ll.Debug().Msg("finished setting up default noop tracer")

		return traceProvider, nil
	}
}
