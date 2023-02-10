package server

import (
	"context"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"

	"github.com/suborbital/e2core/e2core/options"
	"github.com/suborbital/go-kit/observability"
)

// setupTracing configure open telemetry to be used with otel exporter. Returns a tracer closer func and an error.
func setupTracing(config options.TracerConfig, logger zerolog.Logger) (func(ctx context.Context) error, error) {
	l := logger.With().Str("function", "setupTracing").Logger()
	emptyShutdown := func(_ context.Context) error { return nil }

	switch config.TracerType {
	case "honeycomb":
		conn, err := observability.GrpcConnection(context.Background(), config.HoneycombConfig.Endpoint)
		if err != nil {
			return emptyShutdown, errors.Wrapf(err, "observability.GrpcConnection to %s", config.HoneycombConfig.Endpoint)
		}

		tp, err := observability.HoneycombTracer(context.Background(), conn, observability.HoneycombTracingConfig{
			TracingConfig: observability.TracingConfig{
				Probability: config.Probability,
				ServiceName: config.ServiceName,
			},
			APIKey:  config.HoneycombConfig.APIKey,
			Dataset: config.HoneycombConfig.Dataset,
		})
		if err != nil {
			return emptyShutdown, errors.Wrap(err, "observability.HoneycombTracer")
		}

		return tp.Shutdown, nil
	case "collector":
		conn, err := observability.GrpcConnection(context.Background(), config.Collector.Endpoint)
		if err != nil {
			return emptyShutdown, errors.Wrap(err, "observability.GrpcConnection")
		}

		tp, err := observability.OtelTracer(context.Background(), conn, observability.TracingConfig{
			Probability: config.Probability,
			ServiceName: config.ServiceName,
		})

		return tp.Shutdown, nil
	default:
		l.Warn().Str("tracerType", config.TracerType).Msg("unrecognised tracer type configuration. Defaulting to no tracer")
		fallthrough
	case "none", "":
		tp, err := observability.NoopTracer()
		if err != nil {
			return emptyShutdown, errors.Wrap(err, "observability.NoopTracer")
		}

		return tp.Shutdown, nil
	}
}
