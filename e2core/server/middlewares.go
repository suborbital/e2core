package server

import (
	"net/http"
	"time"

	"go.opentelemetry.io/otel"

	"github.com/suborbital/vektor/vk"
)

type requestScope struct {
	RequestID string `json:"request_id"`
}

func scopeMiddleware(inner vk.HandlerFunc) vk.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, ctx *vk.Ctx) error {
		scope := requestScope{
			RequestID: ctx.RequestID(),
		}

		ctx.UseScope(scope)

		return inner(w, r, ctx)
	}
}

// traceKey is how request values are stored/retrieved.
const traceKey string = "traceValues"

// Values represent state for each request.
type Values struct {
	TraceID   string
	Now       time.Time
	RequestID string
}

func (s *Server) openTelemetryMiddleware() vk.Middleware {
	return func(inner vk.HandlerFunc) vk.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request, ctx *vk.Ctx) error {
			tracedCtx, span := otel.GetTracerProvider().Tracer("").Start(ctx.Context, "coordinator.openTelemetryHandler")
			defer span.End()

			ctx.Context = tracedCtx

			v := Values{
				TraceID:   span.SpanContext().TraceID().String(),
				Now:       time.Now().UTC(),
				RequestID: ctx.RequestID(),
			}

			ctx.Set(traceKey, v)
			return inner(w, r, ctx)
		}
	}
}
