//go:build proxy

package atmo

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
	"google.golang.org/grpc/credentials"

	"github.com/suborbital/atmo/atmo/options"
	"github.com/suborbital/vektor/vlog"
)

func setupLogger(_ *vlog.Logger) {
	// do nothing in proxy mode
}

// setupTracing configure open telemetry to be used with otel exporter. Returns a tracer closer func and an error.
func setupTracing(config options.TracerConfig) (func(), error) {
	exporter, err := newExporter(context.Background(), config)
	if err != nil {
		return func() {}, fmt.Errorf("creating OTLP trace exporter: %w", err)
	}

	traceProvider := trace.NewTracerProvider(
		trace.WithSampler(trace.TraceIDRatioBased(config.Probability)),
		trace.WithBatcher(exporter,
			trace.WithMaxExportBatchSize(trace.DefaultMaxExportBatchSize),
			trace.WithBatchTimeout(trace.DefaultScheduleDelay*time.Millisecond),
			trace.WithMaxExportBatchSize(trace.DefaultMaxExportBatchSize),
		),
		trace.WithResource(
			resource.NewWithAttributes(
				semconv.SchemaURL,
				semconv.ServiceNameKey.String(config.ServiceName),
				attribute.String("exporter", "zipkin"),
			),
		),
	)

	// I can only get this working properly using the singleton :(
	otel.SetTracerProvider(traceProvider)

	return func() {
		_ = traceProvider.Shutdown(context.Background())
	}, nil

}

// newExporter encapsulates putting together the exporter for the tracing. In our case it's an otlptracegrpc client.
func newExporter(ctx context.Context, config options.TracerConfig) (*otlptrace.Exporter, error) {
	opts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(config.Endpoint),
		otlptracegrpc.WithHeaders(map[string]string{
			"x-honeycomb-team":    config.APIKey,
			"x-honeycomb-dataset": config.Dataset,
		}),
		otlptracegrpc.WithTLSCredentials(credentials.NewClientTLSFromCert(nil, "")),
	}

	client := otlptracegrpc.NewClient(opts...)
	return otlptrace.New(ctx, client)
}
