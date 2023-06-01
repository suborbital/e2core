package handlers

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/suborbital/e2core/foundation/tracing"
	"github.com/suborbital/e2core/nuexecutor/worker"
	httpKit "github.com/suborbital/go-kit/web/http"
)

func Sync(wc chan<- worker.Job, l zerolog.Logger) echo.HandlerFunc {
	return func(c echo.Context) error {
		// grab the request ID
		rid := httpKit.RID(c)

		// create a 5 second request timeout
		ctx, cxl := context.WithTimeout(c.Request().Context(), 5*time.Second)
		defer cxl()

		// start a trace span from the context
		ctx, span := tracing.Tracer.Start(ctx, "handlers.sync", trace.WithAttributes(
			attribute.String("requestID", rid),
		))
		defer span.End()

		// put the context back to the echo context.
		c.SetRequest(c.Request().WithContext(ctx))

		// read the body (input to our wasm function)
		jobBytes, err := io.ReadAll(c.Request().Body)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "something went wrong").SetInternal(errors.Wrap(err, "io.ReadAll body"))
		}

		// create a new job with the context (has the tracing), request ID (can connect to others), and input
		j := worker.NewJob(ctx, rid, jobBytes)

		// send it
		wc <- j
		span.AddEvent("sent job to channel")

		// see what happened with the job
		select {
		case err := <-j.Error():
			span.AddEvent("job errored out")
			return echo.NewHTTPError(http.StatusInternalServerError, "execution failed").SetInternal(errors.Wrap(err, "job errorchan"))
		case result := <-j.Result():
			span.AddEvent("job result came back")
			return c.Blob(http.StatusOK, "application/octet-stream", result.Output())
		case <-ctx.Done():
			span.AddEvent("request timeout reached")
			return c.String(http.StatusRequestTimeout, "request timed out")
		}
	}
}
