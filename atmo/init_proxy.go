//go:build proxy

package atmo

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/suborbital/atmo/atmo/options"
	"github.com/suborbital/vektor/vlog"
)

func setupLogger(_ *vlog.Logger) {
	// do nothing in proxy mode
}

// setupTracing configure open telemetry to be used with otel exporter. Returns a tracer closer func and an error.
func setupTracing(config options.TracerConfig, logger *vlog.Logger) (func(), error) {
	exporterString := "collector"
	exporterOpts := make([]otlptracegrpc.Option, 0)

	if config.APIKey != "" {
		exporterString = "honeycomb"
		honeyOpts, err := honeycombExporterOptions(config)
		if err != nil {
			return func() {}, errors.Wrap(err, "honeycombExporterOptions")
		}
		exporterOpts = append(exporterOpts, honeyOpts...)

		logger.Info("created OTLP trace exporter with endpoint and apikey")
	} else {
		collectorOpts, err := collectorExporterOptions()
		if err != nil {
			return func() {}, errors.Wrap(err, "collectorExporterOptions")
		}
		exporterOpts = append(exporterOpts, collectorOpts...)

		logger.Info("created tracer configured to use a collector")
	}

	exporter, err := otlptrace.New(context.Background(), otlptracegrpc.NewClient(exporterOpts...))
	if err != nil {
		return func() {}, errors.Wrapf(err, "oltptrace.New with exporter as %s", exporterString)
	}

	traceOpts := []trace.TracerProviderOption{
		trace.WithSampler(trace.TraceIDRatioBased(config.Probability)),
		trace.WithResource(
			resource.NewWithAttributes(
				semconv.SchemaURL,
				semconv.ServiceNameKey.String(config.ServiceName),
				attribute.String("exporter", exporterString),
			),
		),
		trace.WithBatcher(exporter,
			trace.WithMaxExportBatchSize(trace.DefaultMaxExportBatchSize),
			trace.WithBatchTimeout(trace.DefaultScheduleDelay*time.Millisecond),
			trace.WithMaxExportBatchSize(trace.DefaultMaxExportBatchSize),
		),
		trace.WithResource(
			resource.NewWithAttributes(
				semconv.SchemaURL,
				semconv.ServiceNameKey.String(config.ServiceName),
				attribute.String("exporter", "honeycomb"),
			),
		),
	}

	traceProvider := trace.NewTracerProvider(traceOpts...)

	otel.SetTracerProvider(traceProvider)

	logger.Info("finished setting up tracer")

	return func() {
		_ = traceProvider.Shutdown(context.Background())
	}, nil

}

func collectorExporterOptions() ([]otlptracegrpc.Option, error) {
	conn, err := grpc.DialContext(context.Background(), "localhost:4317", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		return nil, errors.Wrap(err, "grpc.DialContext")
	}

	return []otlptracegrpc.Option{
		otlptracegrpc.WithGRPCConn(conn),
		otlptracegrpc.WithTimeout(500 * time.Millisecond),
	}, nil
}

func honeycombExporterOptions(config options.TracerConfig) ([]otlptracegrpc.Option, error) {
	return []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(config.Endpoint),
		otlptracegrpc.WithHeaders(map[string]string{
			"x-honeycomb-team":    config.APIKey,
			"x-honeycomb-dataset": config.Dataset,
		}),
		otlptracegrpc.WithTLSCredentials(credentials.NewClientTLSFromCert(nil, "")),
	}, nil
}
