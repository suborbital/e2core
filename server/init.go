package server

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
	traceProviders "go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/suborbital/deltav/options"
	"github.com/suborbital/vektor/vlog"
)

// setupTracing configure open telemetry to be used with otel exporter. Returns a tracer closer func and an error.
func setupTracing(config options.TracerConfig, logger *vlog.Logger) (func(), error) {
	exporterString := "none"
	exporterOpts := make([]otlptracegrpc.Option, 0)

	switch config.TracerType {
	case "honeycomb":
		logger.Debug("configuring honeycomb exporter for tracing")

		honeyOpts, err := honeycombExporterOptions(config.HoneycombConfig)
		if err != nil {
			return func() {}, errors.Wrap(err, "honeycombExporterOptions")
		}

		exporterString = "honeycomb"
		exporterOpts = append(exporterOpts, honeyOpts...)

		logger.Debug("created honeycomb trace exporter")
	case "collector":
		logger.Debug("configuring collector exporter for tracing")

		collectorOpts, err := collectorExporterOptions(config.Collector)
		if err != nil {
			return func() {}, errors.Wrap(err, "collectorExporterOptions")
		}

		exporterString = "collector"
		exporterOpts = append(exporterOpts, collectorOpts...)

		logger.Debug("created collector trace exporter")
	default:
		logger.Warn(fmt.Sprintf("unrecognised tracer type configuration [%s]. Defaulting to no tracer", config.TracerType))
		fallthrough
	case "none", "":
		otel.SetTracerProvider(traceProviders.NewNoopTracerProvider())

		logger.Debug("finished setting up noop tracer")

		return func() {}, nil
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

	logger.Debug(fmt.Sprintf("finished setting up tracer [%s] with a trace probability of [%f]",
		exporterString, config.Probability))

	return func() {
		_ = traceProvider.Shutdown(context.Background())
	}, nil

}

func collectorExporterOptions(config *options.CollectorConfig) ([]otlptracegrpc.Option, error) {
	if config == nil {
		return nil, errors.New("empty collector tracer configuration")
	}

	ctx, ctxCancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer ctxCancel()

	conn, err := grpc.DialContext(ctx, config.Endpoint, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		return nil, errors.Wrap(err, "grpc.DialContext")
	}

	return []otlptracegrpc.Option{
		otlptracegrpc.WithGRPCConn(conn),
		otlptracegrpc.WithTimeout(500 * time.Millisecond),
	}, nil
}

func honeycombExporterOptions(config *options.HoneycombConfig) ([]otlptracegrpc.Option, error) {
	if config == nil {
		return nil, errors.New("empty honeycomb tracer configuration")
	}

	return []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(config.Endpoint),
		otlptracegrpc.WithHeaders(map[string]string{
			"x-honeycomb-team":    config.APIKey,
			"x-honeycomb-dataset": config.Dataset,
		}),
		otlptracegrpc.WithTLSCredentials(credentials.NewClientTLSFromCert(nil, "")),
	}, nil
}
