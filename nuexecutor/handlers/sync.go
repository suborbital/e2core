package handlers

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"

	"github.com/suborbital/e2core/foundation/tracing"
	"github.com/suborbital/e2core/nuexecutor/worker"
)

func Sync(wc chan<- worker.Job) echo.HandlerFunc {

	return func(c echo.Context) error {
		ctx, cxl := context.WithTimeout(c.Request().Context(), 5*time.Second)
		defer cxl()

		ctx, span := tracing.Tracer.Start(ctx, "handlers.sync")
		defer span.End()

		c.SetRequest(c.Request().WithContext(ctx))

		jobBytes, err := io.ReadAll(c.Request().Body)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "something went wrong").SetInternal(errors.Wrap(err, "io.ReadAll body"))
		}

		j := worker.NewJob(ctx, jobBytes)

		wc <- j

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
