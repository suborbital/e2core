package sat

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/suborbital/e2core/foundation/scheduler"
	"github.com/suborbital/e2core/sat/engine2"
	"github.com/suborbital/e2core/sat/sat/metrics"
	"github.com/suborbital/systemspec/request"
)

func (s *Sat) handler(engine *engine2.Engine) echo.HandlerFunc {
	return func(c echo.Context) error {
		spanCtx, span := s.tracer.Start(c.Request().Context(), "echoHandler", trace.WithAttributes(
			attribute.String("request_id", c.Request().Header.Get("requestID")),
		))
		defer span.End()

		s.metrics.FunctionExecutions.Add(spanCtx, 1)

		c.SetRequest(c.Request().WithContext(spanCtx))

		req, err := request.FromEchoContext(c)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError).SetInternal(errors.Wrap(err, "request.FromEchoContext"))
		}

		t := metrics.NewTimer()

		var runErr scheduler.RunErr

		if !engine.IsRegistered(s.config.JobType) {
			return echo.NewHTTPError(http.StatusInternalServerError, "unknown error").SetInternal(fmt.Errorf("module %s is not registered", s.config.JobType))
		}

		result, err := engine.Do(scheduler.NewJob(s.config.JobType, req)).Then()
		if err != nil {
			if errors.As(err, &runErr) {
				// runErr would be an actual error returned from a function
				// should find a better way to determine if a RunErr is "non-nil"
				if runErr.Code != 0 || runErr.Message != "" {
					return echo.NewHTTPError(runErr.Code, runErr.Message).SetInternal(err)
				}
			}

			return echo.NewHTTPError(http.StatusInternalServerError, "unknown error").SetInternal(errors.Wrap(err, "engine.Do"))
		}

		s.metrics.FunctionTime.Record(spanCtx, t.Observe(), attribute.String("id", req.ID))

		if result == nil {
			s.logger.Debug().Str("fn", s.config.JobType).Msg("returned a nil result")
			return nil
		}

		resp, ok := result.(*request.CoordinatedResponse)
		if !ok {
			return echo.NewHTTPError(http.StatusInternalServerError).SetInternal(errors.New("result from engine.Do was not a coordinated response struct"))
		}

		for headerKey, headerValue := range resp.RespHeaders {
			c.Response().Header().Add(headerKey, headerValue)
		}

		return c.Blob(http.StatusOK, "application/octet-stream", resp.Output)
	}
}
