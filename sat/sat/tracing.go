package sat

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/sdk/trace"

	"github.com/suborbital/e2core/sat/sat/options"
	"github.com/suborbital/go-kit/observability"
	"github.com/suborbital/vektor/vlog"
)

// SetupTracing configure open telemetry to be used with otel exporter. Returns a tracer closer func and an error.
func SetupTracing(config options.TracerConfig, logger *vlog.Logger) (*trace.TracerProvider, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	switch config.TracerType {
	case "honeycomb":
		if config.HoneycombConfig == nil {
			return nil, errors.New("missing honeycomb tracing config values")
		}

		logger.Info("configuring honeycomb exporter for tracing")

		conn, err := observability.GrpcConnection(ctx, config.HoneycombConfig.Endpoint, &tls.Config{})
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

		logger.Info("created honeycomb trace exporter")

		return traceProvider, nil
	case "collector":
		if config.Collector == nil {
			return nil, errors.New("missing collector tracing config values")
		}

		logger.Info("configuring collector exporter for tracing")

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

		logger.Info("created collector trace exporter")

		return traceProvider, nil
	default:
		logger.Warn(fmt.Sprintf("unrecognised tracer type configuration [%s]. Defaulting to no tracer", config.TracerType))
		fallthrough
	case "none", "":
		// Create the most default trace provider and escape early.
		traceProvider, err := observability.NoopTracer()
		if err != nil {
			return nil, errors.Wrap(err, "noop Tracer")
		}

		logger.Debug("finished setting up default noop tracer")

		return traceProvider, nil
	}
}
