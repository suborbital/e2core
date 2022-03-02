package coordinator

import (
	"net/http"
	"time"

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
