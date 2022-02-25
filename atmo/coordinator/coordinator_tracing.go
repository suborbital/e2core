package coordinator

import (
	"fmt"
	"net/http"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/zipkin"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
	otelTrace "go.opentelemetry.io/otel/trace"

	"github.com/suborbital/vektor/vk"
)

// traceKey is how request values are stored/retrieved.
const traceKey string = "traceValues"

// Values represent state for each request.
type Values struct {
	TraceID    string
	Now        time.Time
	StatusCode int
}

func (c *Coordinator) openTelemetryHandler() vk.Middleware {
	return func(r *http.Request, ctx *vk.Ctx) error {
		// Capture the parent request span from the context.
		span := otelTrace.SpanFromContext(ctx.Context)

		v := Values{
			TraceID: span.SpanContext().TraceID().String(),
			Now:     time.Now().UTC(),
		}

		ctx.Set(traceKey, v)
		return nil
	}
}

// startTracing configure open telemetry to be used with zipkin.
func startTracing(serviceName string, reporterURI string, probability float64) (*trace.TracerProvider, error) {
	exporter, err := zipkin.New(
		reporterURI,
		// zipkin.WithLogger(zap.NewStdLog(log)),
	)
	if err != nil {
		return nil, fmt.Errorf("creating new exporter: %w", err)
	}

	traceProvider := trace.NewTracerProvider(
		trace.WithSampler(trace.TraceIDRatioBased(probability)),
		trace.WithBatcher(exporter,
			trace.WithMaxExportBatchSize(trace.DefaultMaxExportBatchSize),
			trace.WithBatchTimeout(trace.DefaultScheduleDelay*time.Millisecond),
			trace.WithMaxExportBatchSize(trace.DefaultMaxExportBatchSize),
		),
		trace.WithResource(
			resource.NewWithAttributes(
				semconv.SchemaURL,
				semconv.ServiceNameKey.String(serviceName),
				attribute.String("exporter", "zipkin"),
			),
		),
	)

	// I can only get this working properly using the singleton :(
	otel.SetTracerProvider(traceProvider)
	return traceProvider, nil
}
